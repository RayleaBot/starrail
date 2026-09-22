package images

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

func TestPeakFollowsStarRailPlugin(t *testing.T) {
	run := func(name string, hard bool) map[string]any {
		return map[string]any{
			"group":     map[string]any{"name_mi18n": name, "game_version": "3.6", "begin_time": clock(2026, 9, 1), "end_time": clock(2026, 10, 13)},
			"boss_info": map[string]any{"name_mi18n": "王棋"},
			"boss_record": map[string]any{"has_challenge_record": true, "hard_mode": hard, "round_num": json.Number("3"), "star_num": json.Number("3"),
				"challenge_time": clock(2026, 9, 5), "avatars": []any{map[string]any{"rarity": json.Number("5"), "rank": json.Number("1"), "element": "Fire"}},
				"buff": map[string]any{"name_mi18n": "增益", "desc_mi18n": "说明"}},
			"mob_infos":   []any{map[string]any{"name": "骑士一", "monster_name": "甲"}, map[string]any{"name": "骑士二", "monster_name": "乙"}, map[string]any{"name": "骑士三", "monster_name": "丙"}},
			"mob_records": []any{map[string]any{"is_fast": true}, map[string]any{"has_challenge_record": true, "star_num": json.Number("2"), "round_num": json.Number("5")}},
			"boss_stars":  json.Number("3"), "mob_stars": json.Number("8"),
		}
	}
	data := map[string]any{"challenge_peak_records": []any{run("本期", true), run("上期", false), run("前期", false)},
		"challenge_peak_best_record_brief": map[string]any{"total_battle_num": json.Number("12")}}
	image, ok := Peak(app.ImageContext{Input: map[string]any{"schedule_type": 1}}, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok || image.Template != "peak" || image.Data["tag"] != "本期" {
		t.Fatalf("image = %v %v", image.Template, image.Data)
	}
	runs := image.Data["records"].([]any)
	current := runs[0].(map[string]any)
	boss := current["boss"].(map[string]any)
	if len(runs) != 1 || current["name"] != "本期" || current["battles"] != "12" || current["begin"] != "2026.09.01" {
		t.Errorf("run = %v", current)
	}
	if boss["hell"] != true || boss["time"] != "2026.09.05 04:00" || len(boss["stars"].([]any)) != 3 || boss["buff"] == nil {
		t.Errorf("boss = %v", boss)
	}
	mobs := current["mobs"].([]any)
	fast, cleared, missing := mobs[0].(map[string]any), mobs[1].(map[string]any), mobs[2].(map[string]any)
	if fast["fast"] != true || cleared["rounds"] != "5" || len(cleared["stars"].([]any)) != 2 || missing["cleared"] != false || cleared["title"] != "骑士二: 乙" {
		t.Errorf("mobs = %v", mobs)
	}

	// The last run is the second of the recent runs.
	image, _ = Peak(app.ImageContext{Input: map[string]any{"schedule_type": 2}}, app.QueryResult{Data: data})
	last := image.Data["records"].([]any)[0].(map[string]any)
	if image.Data["tag"] != "上期" || last["name"] != "上期" || last["boss"].(map[string]any)["hell"] != false {
		t.Errorf("last = %v", image.Data)
	}
	// 往期 puts the three recent runs side by side.
	image, _ = Peak(app.ImageContext{Input: map[string]any{"schedule_type": 3}}, app.QueryResult{Data: data})
	if image.Template != "peak-recent" || len(image.Data["records"].([]any)) != 3 {
		t.Errorf("recent = %v %v", image.Template, image.Data)
	}
	data["challenge_peak_records"] = []any{run("本期", true)}
	if _, ok := Peak(app.ImageContext{Input: map[string]any{"schedule_type": 2}}, app.QueryResult{Data: data}); ok {
		t.Error("without a last run the reply stays in text")
	}
}
