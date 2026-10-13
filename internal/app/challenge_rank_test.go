package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestChallengeScoresSeparatePeriodsAndMissingMetrics(t *testing.T) {
	fixtures := []struct {
		game, kind, raw, key string
		value                float64
		period               int
	}{
		{"starrail", "challenge", `{"schedule_id":12,"max_floor":12,"all_floor_detail":[{"star_num":3,"round_num":4}]}`, "round", 4, 1},
		{"starrail", "story", `{"schedule_id":12,"max_floor":4,"all_floor_detail":[{"star_num":3,"score":80000}]}`, "score", 80000, 1},
		{"starrail", "boss", `{"schedule_id":12,"max_floor":4,"all_floor_detail":[{"star_num":3,"score":7600}]}`, "score", 7600, 1},
		{"starrail", "peak", `{"challenge_peak_records":[{"group":{"group_id":13},"boss_stars":3},{"group":{"group_id":12},"boss_stars":2,"mob_stars":9,"boss_record":{"round_num":7,"hard_mode":true}}]}`, "round", 7, 2},
	}
	for _, f := range fixtures {
		t.Run(f.game+"."+f.kind, func(t *testing.T) {
			var data map[string]any
			json.Unmarshal([]byte(f.raw), &data)
			kind, _ := challengeKind(f.kind)
			r, e := challengeExtract(kind, data, f.period)
			if e != nil || r.Metrics[f.key] != f.value {
				t.Fatalf("%+v %v", r, e)
			}
			if f.kind == "peak" && r.Season != "12" {
				t.Fatal("previous period mixed", r)
			}
			if _, e = challengeExtract(kind, map[string]any{}, 1); e == nil {
				t.Fatal("missing data accepted")
			}
		})
	}
	kind, _ := challengeKind("challenge")
	a := ChallengeEntry{Metrics: map[string]float64{"floor": 12, "star": 3, "round": 4}}
	b := ChallengeEntry{Metrics: map[string]float64{"floor": 12, "star": 3}}
	if challengeCompare(kind, "", a, b) >= 0 {
		t.Fatal("missing round ranked as zero")
	}
	b.Metrics["round"] = 3
	if challengeCompare(kind, "", a, b) <= 0 {
		t.Fatal("lower round must win")
	}
}
func TestChallengeGroupPaginationRestartWithdrawalAndRevision(t *testing.T) {
	dir := t.TempDir()
	s := &GroupStore{Directory: filepath.Join(dir, "groups")}
	scope := GroupScope{"onebot11", "bot", "group"}
	now := time.Now().UnixMilli()
	for i := 0; i < 23; i++ {
		e := ChallengeEntry{Kind: "story", Season: "12", Region: "prod_gf_cn", ActorID: fmt.Sprint(i), UID: fmt.Sprint(100000000 + i), Metrics: map[string]float64{"floor": float64(i)}, FirstMS: now, UpdatedMS: now}
		if err := s.SubmitChallenge(scope, e); err != nil {
			t.Fatal(err)
		}
	}
	a := App{Game: Game{ID: "starrail"}, Groups: &GroupStore{Directory: s.Directory}}
	kind, _ := challengeKind("story")
	out, err := a.challengeList(scope, kind, "", 0, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	group := out["groups"].([]map[string]any)[0]
	if len(group["items"].([]ChallengeEntry)) != 3 || group["items"].([]ChallengeEntry)[0].ActorID != "2" {
		t.Fatal(group)
	}
	other := scope
	other.BotID = "other"
	empty, _ := a.challengeList(other, kind, "", 0, "", 0)
	if len(empty["groups"].([]map[string]any)) != 0 {
		t.Fatal("cross bot leak")
	}
	_, err = a.manageChallenge("challenge.clear", map[string]any{"scope": scope, "confirmed": true, "revision": 0})
	if err == nil {
		t.Fatal("stale destructive clear")
	}
	_, err = a.manageChallenge("challenge.clear", map[string]any{"scope": scope, "confirmed": true, "revision": out["revision"], "kind": "story"})
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := s.Read(scope)
	if len(fresh.Challenges) != 0 {
		t.Fatal(fresh)
	}
}

func TestChallengeKindsTakeRecordNames(t *testing.T) {
	for word, want := range map[string]string{
		"混沌回忆": "challenge", "忘却之庭": "challenge", "混沌": "challenge",
		"虚构叙事": "story", "末日幻影": "boss", "异相仲裁": "peak", "仲裁": "peak", "peak": "peak",
	} {
		if kind, ok := challengeKind(word); !ok || kind.ID != want {
			t.Errorf("%s: %+v, %v", word, kind, ok)
		}
	}
}

func TestChallengePeriodsFollowOfficialTimes(t *testing.T) {
	clock := func(month, day int) map[string]any {
		return map[string]any{"year": 2026, "month": month, "day": day, "hour": 4, "minute": 0}
	}
	previous := map[string]any{"group_id": 1, "status": "End", "begin_time": clock(7, 27), "end_time": clock(9, 7)}
	current := map[string]any{"group_id": 2, "status": "New", "begin_time": clock(9, 7), "end_time": clock(10, 19)}
	for _, tc := range []struct {
		kind       string
		period     int
		data       map[string]any
		begin, end string
	}{
		{"story", 2, map[string]any{"max_floor": 4, "groups": []any{current, previous}}, "2026-07-27T04:00", "2026-09-07T04:00"},
		{"boss", 1, map[string]any{"max_floor": 4, "groups": []any{previous, current}}, "2026-09-07T04:00", "2026-10-19T04:00"},
		{"challenge", 1, map[string]any{"schedule_id": 1, "max_floor": 12, "begin_time": clock(9, 7), "end_time": clock(10, 19)}, "2026-09-07T04:00", "2026-10-19T04:00"},
		{"peak", 1, map[string]any{"challenge_peak_records": []any{map[string]any{"group": current, "boss_stars": 3}}}, "2026-09-07T04:00", "2026-10-19T04:00"},
	} {
		kind, _ := challengeKind(tc.kind)
		entry, err := challengeExtract(kind, tc.data, tc.period)
		if err != nil || entry.Begin != tc.begin || entry.End != tc.end {
			t.Errorf("%s period %d: %+v, %v", tc.kind, tc.period, entry, err)
		}
	}
	entries := []ChallengeEntry{
		{Region: "prod_gf_cn", Season: "A", Begin: "2026-09-07T04:00", End: "2026-10-19T04:00"},
		{Region: "prod_gf_cn", Season: "B", Begin: "2026-07-27T04:00", End: "2026-09-07T04:00"},
		{Region: "prod_gf_cn", Season: "C", Begin: "2026-06-15T04:00", End: "2026-07-27T04:00"},
		{Region: "prod_gf_cn", Season: "X"},
		{Region: "prod_official_usa", Season: "USA", Begin: "2026-10-13T00:00", End: "2026-11-24T00:00"},
	}
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.FixedZone("", 8*3600))
	for period, want := range map[int]string{1: "A", 2: "B"} {
		got := challengePeriodSeasons(entries, period, now)
		if len(got) != 1 || got["prod_gf_cn"] != want {
			t.Errorf("period %d: %v", period, got)
		}
	}
	// With nobody in this period, the last ended season is not current.
	if got := challengePeriodSeasons(entries[1:], 1, now); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestChallengeRankChatSelectsPeriodAndChineseMetric(t *testing.T) {
	a := pluginApp(t)
	scope := GroupScope{"onebot11", "bot", "g"}
	now := time.Now().In(time.FixedZone("", 8*3600))
	for _, tc := range []struct {
		actor, season string
		score         float64
		begin, end    time.Time
	}{
		{"低分", "当前期", 40000, now.AddDate(0, 0, -7), now.AddDate(0, 0, 7)},
		{"高分", "当前期", 80000, now.AddDate(0, 0, -7), now.AddDate(0, 0, 7)},
		{"往期", "上一期", 90000, now.AddDate(0, 0, -49), now.AddDate(0, 0, -7)},
	} {
		entry := ChallengeEntry{Kind: "story", Region: "prod_gf_cn", Season: tc.season, ActorID: tc.actor, Nickname: tc.actor, UID: "100000001", Metrics: map[string]float64{"floor": 4, "score": tc.score}, Begin: tc.begin.Format(challengeClockLayout), End: tc.end.Format(challengeClockLayout), FirstMS: now.UnixMilli(), UpdatedMS: now.UnixMilli()}
		if err := a.Groups.SubmitChallenge(scope, entry); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Groups.SubmitChallenge(scope, ChallengeEntry{Kind: "story", Region: "prod_gf_cn", Season: "当前期", ActorID: "无时间", Nickname: "无时间", UpdatedMS: now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	host := newSDKHost(t, a, nil)
	end, _ := host.groupMessage("*挑战排行 虚构叙事 本期 得分", "挑战排行", "虚构叙事", "本期", "得分")
	text := terminalText(end)
	if !strings.Contains(text, "当前期") || strings.Contains(text, "上一期") || strings.Contains(text, "无时间") || !strings.Contains(text, "高分") || !strings.Contains(text, "低分") || strings.Index(text, "高分") > strings.Index(text, "低分") {
		t.Fatal(text)
	}
	end, _ = host.groupMessage("*挑战排行 虚构 全部 星星", "挑战排行", "虚构", "全部", "星星")
	if text := terminalText(end); !strings.Contains(text, "得分") || !strings.Contains(text, "综合") {
		t.Fatal(text)
	}
	for _, word := range []string{"得分", "score", "综合", "未知期次"} {
		end, _ := host.groupMessage("*挑战排行 虚构 "+word, "挑战排行", "虚构", word)
		text := terminalText(end)
		if !strings.Contains(text, "期次写本期、上期或全部") || !strings.Contains(text, "指标写在第三项") || strings.Contains(text, "本群还没有") {
			t.Errorf("%s: %s", word, text)
		}
		if word != "未知期次" && !strings.Contains(text, "参数顺序") {
			t.Errorf("%s: %s", word, text)
		}
	}
	end, _ = host.groupMessage("*挑战排行 虚构 当前期 得分", "挑战排行", "虚构", "当前期", "得分")
	if text := terminalText(end); !strings.Contains(text, "高分") || strings.Contains(text, "上一期") {
		t.Fatal(text)
	}
	end, _ = host.groupMessage("*挑战排行 混沌 本期", "挑战排行", "混沌", "本期")
	if text := terminalText(end); !strings.Contains(text, "本群还没有本期混沌成绩") {
		t.Fatal(text)
	}
}
