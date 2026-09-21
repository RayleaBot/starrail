package images

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

func clock(year, month, day int) map[string]any {
	number := func(value int) json.Number { return json.Number(strconv.Itoa(value)) }
	return map[string]any{"year": number(year), "month": number(month), "day": number(day), "hour": json.Number("4"), "minute": json.Number("0")}
}

func TestChallengeFollowsStarRailPlugin(t *testing.T) {
	avatar := map[string]any{"id": json.Number("1102"), "level": json.Number("80"), "icon": "http://insecure.example/1102.png", "rarity": json.Number("5"), "element": "Quantum", "rank": json.Number("2")}
	node := func(score string, defeated bool) map[string]any {
		return map[string]any{"challenge_time": clock(2026, 9, 10), "avatars": []any{avatar}, "score": json.Number(score), "boss_defeated": defeated,
			"buff": map[string]any{"name_mi18n": "终焉公理", "desc_mi18n": "说明", "icon": "https://act-webstatic.mihoyo.com/buff.png"}}
	}
	data := map[string]any{
		"star_num": json.Number("10"), "extra_star_num": json.Number("1"), "max_floor": "难度4", "battle_num": json.Number("5"),
		"groups": []any{
			map[string]any{"status": "End", "begin_time": clock(2026, 7, 27), "end_time": clock(2026, 9, 7)},
			map[string]any{"status": "New", "begin_time": clock(2026, 9, 7), "end_time": clock(2026, 10, 19),
				"upper_boss": map[string]any{"name_mi18n": "上半首领"}, "lower_boss": map[string]any{"name_mi18n": "下半首领"}, "tierce_boss": map[string]any{"name_mi18n": "第三首领"}},
		},
		"all_floor_detail": []any{
			map[string]any{"name": "难度4", "star_num": json.Number("4"), "extra_star_num": json.Number("1"), "is_tierce": true, "is_fast": false,
				"node_1": node("3000", true), "node_2": node("2500", false), "node_3": node("1000", true)},
			map[string]any{"name": "难度1", "star_num": json.Number("3"), "is_fast": true, "node_1": node("4000", true)},
		},
	}
	context := gamekit.ImageContext{Now: time.Date(2026, 9, 12, 12, 0, 0, 0, chinaTime), Input: map[string]any{"schedule_type": 1}}
	image, ok := challengeImage(apocalypticShadow)(context, gamekit.QueryResult{Role: gamekit.Role{UID: "100000001"}, Data: data})
	if !ok {
		t.Fatal("boss record")
	}
	// The current period keeps the New group, freshly opened.
	if image.Data["begin"] != "2026.09.07" || image.Data["end"] != "2026.10.19" || image.Data["badge"] != "new" || image.Data["period"] != "本期" {
		t.Errorf("period = %v %v %v %v", image.Data["begin"], image.Data["end"], image.Data["badge"], image.Data["period"])
	}
	if bosses := image.Data["bosses"].([]any); len(bosses) != 3 || bosses[2].(map[string]any)["third"] != true {
		t.Errorf("bosses = %v", bosses)
	}
	floors := image.Data["floors"].([]any)
	if len(floors) != 1 {
		t.Fatalf("a quick-cleared floor is skipped like upstream: %v", floors)
	}
	floor := floors[0].(map[string]any)
	stars := floor["stars"].([]any)
	if floor["score"] != "6500" || floor["has_round"] != false || len(stars) != 4 || stars[3] != "abyss-star" {
		t.Errorf("floor = %v", floor)
	}
	first := floor["nodes"].([]any)[0].(map[string]any)
	member := first["avatars"].([]any)[0].(map[string]any)
	if first["time"] != "2026.09.10 04:00" || first["boss"] != true || first["score"] != "3000" || member["rank"] != 2 || member["icon"] != "" {
		t.Errorf("node = %v", first)
	}

	// The last period keeps the ended group and carries no mark.
	context.Input = map[string]any{"schedule_type": 2}
	image, _ = challengeImage(pureFiction)(context, gamekit.QueryResult{Data: data})
	if image.Data["begin"] != "2026.07.27" || image.Data["badge"] != "" || image.Data["period"] != "上期" {
		t.Errorf("last period = %v %v %v", image.Data["begin"], image.Data["badge"], image.Data["period"])
	}

	// Memory of Chaos reads its own period and totals no score.
	data["begin_time"], data["end_time"] = clock(2026, 8, 10), clock(2026, 9, 21)
	context.Input = nil
	image, _ = challengeImage(memoryOfChaos)(context, gamekit.QueryResult{Data: data})
	if image.Data["begin"] != "2026.08.10" || image.Data["badge"] != "expiring" || image.Data["floors"].([]any)[0].(map[string]any)["score"] != "" {
		t.Errorf("memory of chaos = %v", image.Data)
	}
}

func TestChallengeRoutesFollowStarRailPlugin(t *testing.T) {
	// Upstream counts 14-day turns from Apocalyptic Shadow on 2024-06-24 04:00.
	for at, want := range map[time.Time]string{
		time.Date(2024, 6, 24, 4, 0, 0, 0, chinaTime): "starrail.boss",
		time.Date(2024, 7, 8, 3, 59, 0, 0, chinaTime): "starrail.boss",
		time.Date(2024, 7, 8, 4, 0, 0, 0, chinaTime):  "starrail.story",
		time.Date(2024, 7, 22, 4, 0, 0, 0, chinaTime): "starrail.challenge",
		time.Date(2024, 8, 5, 4, 0, 0, 0, chinaTime):  "starrail.boss",
	} {
		if got := challengeQueries()["starrail.abyss_latest"](at); got != want {
			t.Errorf("%v = %s, want %s", at, got, want)
		}
	}
	if got := challengeQueries()["starrail.abyss"](time.Now()); got != "starrail.challenge" {
		t.Errorf("深渊 starts from %s", got)
	}

	period := map[string]any{"begin_time": clock(2026, 9, 1), "end_time": clock(2026, 10, 13)}
	record := map[string]any{"star_num": json.Number("12"), "groups": []any{period}, "all_floor_detail": []any{}}
	context := gamekit.ImageContext{Now: time.Date(2026, 9, 21, 12, 0, 0, 0, chinaTime)}
	image, ok := latestChallengeImage(context, gamekit.QueryResult{Operation: "starrail.story", Data: record})
	if !ok || image.Data["kind"] != "fiction" {
		t.Errorf("latest = %v", image.Data)
	}

	// 深渊 adds the other two records of the same period.
	asked := []string{}
	context.Input = map[string]any{"schedule_type": 2}
	context.Query = func(operation string, input map[string]any) (gamekit.QueryResult, error) {
		asked = append(asked, operation+"/"+gamekit.Text(input["schedule_type"]))
		return gamekit.QueryResult{Data: record}, nil
	}
	image, ok = allChallengesImage(context, gamekit.QueryResult{Role: gamekit.Role{UID: "100000001"}, Data: record})
	records, _ := image.Data["records"].([]any)
	if !ok || image.Template != "challenge-all" || len(records) != 3 || image.Data["period"] != "上期" || strings.Join(asked, ",") != "starrail.boss/2,starrail.story/2" {
		t.Fatalf("all = %v %v", image.Data, asked)
	}
	for index, kind := range []string{"chaos", "shadow", "fiction"} {
		if records[index].(map[string]any)["kind"] != kind {
			t.Errorf("column %d = %v", index, records[index])
		}
	}
	context.Query = func(string, map[string]any) (gamekit.QueryResult, error) {
		return gamekit.QueryResult{}, errors.New("captcha")
	}
	if _, ok := allChallengesImage(context, gamekit.QueryResult{Data: record}); ok {
		t.Error("a failed record leaves the reply in text")
	}
}
