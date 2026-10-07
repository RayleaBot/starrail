package images_test

import (
	"math"
	"testing"
	"time"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/assets"
	"github.com/RayleaBot/starrail/internal/images"
)

func TestPanelFollowsMiaoRules(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var image app.PanelImage
	for _, record := range application.Game.Calc.Metadata().Characters {
		if record.Key == "sr_1306" {
			image.Record = record
		}
	}
	skills := []app.PanelSkill{{ID: "1306001", Kind: "普攻", Level: 6, Active: true}, {ID: "1306002", Kind: "战技", Level: 12, ExtraLevel: 2, Active: true}}
	for _, id := range []string{"1306101", "1306201", "1306202"} {
		skills = append(skills, app.PanelSkill{ID: id, PointType: 1, Active: id != "1306202"})
	}
	image.Panel = app.CharacterPanel{ID: "1306", Level: 80, Rank: 1, Element: "量子", Skills: skills,
		Weapon: &app.PanelEquipment{ID: "23003", Name: "但战斗还未结束", Level: 80, Refinement: 5, Main: []app.PanelStat{{Key: "hpPlus", Value: "1164"}}}}
	game := application.Game
	game.Prefix = "*"
	drawn, ok := images.Panel(app.ImageContext{Game: game, Now: time.Now()}, image)
	if !ok || drawn.Template != "panel" || drawn.Data["elem"] != "quantum" {
		t.Fatalf("drawn = %+v", drawn)
	}
	talents := drawn.Data["talents"].([]any)
	if len(talents) != 4 || talents[1].(map[string]any)["plus"] != true || talents[1].(map[string]any)["name"] != "战技" {
		t.Errorf("talents = %v", talents)
	}
	// Traces: ten stat nodes in six groups around three ability nodes.
	trees := drawn.Data["trees"].([]any)
	if len(trees) != 9 || trees[2].(map[string]any)["talent"] != true || trees[2].(map[string]any)["on"] != true {
		t.Fatalf("trees = %v", trees)
	}
	first := trees[0].(map[string]any)["nodes"].([]any)
	second := trees[1].(map[string]any)["nodes"].([]any)
	if len(first) != 1 || first[0].(map[string]any)["on"] != true || len(second) != 2 || second[0].(map[string]any)["on"] != false {
		t.Errorf("stat nodes = %v %v", first, second)
	}
	weapon := drawn.Data["weapon"].(map[string]any)
	if weapon["hp"] != "1,164" || weapon["desc"].([]any)[0].(map[string]any)["value"] != "18.0%" {
		t.Errorf("light cone = %v", weapon)
	}
	if drawn.Data["artifact_hint"] != "*花火圣遗物" {
		t.Errorf("hint = %v", drawn.Data["artifact_hint"])
	}
}

func TestPanelDrawsArkRanks(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var image app.PanelImage
	for _, record := range application.Game.Calc.Metadata().Characters {
		if record.Key == "sr_1102" {
			image.Record = record
		}
	}
	expected := 45892.0
	image.Panel = app.CharacterPanel{ID: "1102", Level: 80, Element: "量子"}
	image.Damage = &app.BuildResult{Baseline: app.BuildScenario{Results: []app.BuildSkillResult{{Title: "战技伤害", Expected: &expected}}}}
	game := application.Game
	game.Prefix = "*"
	context := app.ImageContext{Game: game, Catalog: application.Catalog, Now: time.Now()}
	// ark's ranks follow the damage rows, numbered on.
	image.Rank = &app.PanelRank{Rows: []app.Row{{Label: "总伤害排名", Value: "1148 / 60392 (1.90%)"}}}
	drawn, _ := images.Panel(context, image)
	rows := drawn.Data["damage"].(map[string]any)["rows"].([]any)
	if last := rows[len(rows)-1].(map[string]any); len(rows) != 2 || last["index"] != 2 || last["title"] != "总伤害排名" || last["full"] != "1148 / 60392 (1.90%)" {
		t.Fatalf("rows = %v", rows)
	}
	scores := []float64{99, 90, 80, 70, 60, 50, 40, 30, 20, 10}
	image.Rank = &app.PanelRank{Chart: &app.PanelRankChart{Damage: &app.RankCurve{Scores: scores, Top: 100, Percent: 1.9, Score: 74.172}, Places: [2]string{"1148 / 60392 (1.90%)", ""}}}
	drawn, _ = images.Panel(context, image)
	chart := drawn.Data["rank_chart"].(map[string]any)
	damage, artis := chart["charts"].([]any)[0].(map[string]any), chart["charts"].([]any)[1].(map[string]any)
	if chart["hint"] != "*希儿排名统计" || len(artis) != 0 || damage["line"] == "" {
		t.Fatalf("chart = %v", chart)
	}
	// x reads 100 - x, and the panel is marked at 100 - its percent with its
	// score to two places.
	ticks := damage["x_ticks"].([]any)
	if ticks[0].(map[string]any)["label"] != "100" || ticks[5].(map[string]any)["label"] != "0" || damage["mark"].(map[string]any)["text"] != "74.17" {
		t.Fatalf("damage = %v", damage)
	}
	if stops := damage["stops"].([]any); stops[2].(map[string]any)["color"] != "rgb(255, 0, 0)" || math.Abs(stops[1].(map[string]any)["offset"].(float64)-0.981) > 1e-9 {
		t.Fatalf("stops = %v", stops)
	}
}

func TestPanelComparesChangedDamageAsArk(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var image app.PanelImage
	for _, record := range application.Game.Calc.Metadata().Characters {
		if record.Key == "sr_1102" {
			image.Record = record
		}
	}
	f := func(v float64) *float64 { return &v }
	long := "一二三四五六七八九十(1)二三四五六七八九十二三"
	image.Panel = app.CharacterPanel{ID: "1102", Level: 80, Element: "量子"}
	image.Damage = &app.BuildResult{Baseline: app.BuildScenario{Results: []app.BuildSkillResult{
		{Title: "战技伤害", Critical: f(110), Expected: f(55)}, {Title: "护盾量", Text: "12,345"}, {Title: long, Expected: f(1)},
	}}}
	image.Original = &app.BuildResult{Baseline: app.BuildScenario{Results: []app.BuildSkillResult{{Title: "战技伤害", Critical: f(100), Expected: f(55)}, {Title: "护盾量", Text: "12,000"}}}}
	image.LongTitles = 1
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Now()}
	drawn, _ := images.Panel(context, image)
	rows := drawn.Data["damage"].(map[string]any)["rows"].([]any)
	first, second, third := rows[0].(map[string]any), rows[1].(map[string]any), rows[2].(map[string]any)
	diff := func(row map[string]any, key string) string { return row[key].(map[string]any)["text"].(string) }
	// An unchanged value reads 0.0% (upstream: " ↓0%"); a text has no critical
	// damage to compare.
	if diff(first, "dmg_diff") != " ↑10%" || diff(first, "avg_diff") != "0.0%" || diff(second, "avg_diff") != " ↑2.9%" || diff(second, "dmg_diff") != "--" || third["avg_diff"] != nil {
		t.Fatalf("rows = %v", rows)
	}
	// 1, ( and ) count half: the title is cut before its 21st width.
	if third["title"] != "一二三四五六七八九十(1)二三四五六七八九..." {
		t.Fatalf("title = %v", third["title"])
	}
	image.LongTitles = 2
	drawn, _ = images.Panel(context, image)
	if damage := drawn.Data["damage"].(map[string]any); damage["wrap"] != true || damage["rows"].([]any)[2].(map[string]any)["title"] != long {
		t.Fatalf("damage = %v", damage)
	}
}

// 排名统计 draws the character's distributions as the panel page's block
// does, with no panel placed on them.
func TestRankStatsDrawsPanelRankCharts(t *testing.T) {
	scores := []float64{99, 90, 80, 70, 60, 50, 40, 30, 20, 10}
	none := func(top float64) *app.RankCurve {
		return &app.RankCurve{Scores: scores, Top: top, Percent: -100, Score: -100}
	}
	image := app.RankStatsImage{Character: app.Entry{ID: "1102", Name: "希儿", Element: "量子"}, Damage: none(100), Artis: none(338.2), DamageTitle: "终结技伤害", Totals: [2]string{"11471", "10994"}}
	drawn, ok := images.RankStats(app.ImageContext{}, image)
	if !ok || drawn.Template != "rank-stats" || drawn.Data["elem"] != "quantum" || drawn.Data["title"] != "终结技伤害" {
		t.Fatalf("drawn = %v", drawn)
	}
	for _, raw := range drawn.Data["charts"].([]any) {
		chart := raw.(map[string]any)
		if _, marked := chart["mark"]; marked || len(chart["stops"].([]any)) != 2 || chart["line"] == "" {
			t.Fatalf("chart = %v", chart)
		}
	}
	// The relic axis tops at the best score, cut to a whole number.
	artis := drawn.Data["charts"].([]any)[1].(map[string]any)["y_ticks"].([]any)
	if last := artis[len(artis)-1].(map[string]any); last["label"] != "338" {
		t.Fatalf("ticks = %v", artis)
	}
	if _, ok := images.RankStats(app.ImageContext{}, app.RankStatsImage{}); ok {
		t.Fatal("drew no distributions")
	}
}
