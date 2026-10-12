package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
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
	out, err := a.challengeList(scope, kind, "", "", 20)
	if err != nil {
		t.Fatal(err)
	}
	group := out["groups"].([]map[string]any)[0]
	if len(group["items"].([]ChallengeEntry)) != 3 || group["items"].([]ChallengeEntry)[0].ActorID != "2" {
		t.Fatal(group)
	}
	other := scope
	other.BotID = "other"
	empty, _ := a.challengeList(other, kind, "", "", 0)
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
