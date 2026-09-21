package images_test

import (
	"testing"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-starrail/internal/assets"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func TestPanelFollowsMiaoRules(t *testing.T) {
	app, err := gamekit.New(assets.Kit(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var image gamekit.PanelImage
	for _, record := range app.Game.Calc.Metadata().Characters {
		if record.Key == "sr_1306" {
			image.Record = record
		}
	}
	skills := []gamekit.PanelSkill{{ID: "1306001", Kind: "普攻", Level: 6, Active: true}, {ID: "1306002", Kind: "战技", Level: 12, ExtraLevel: 2, Active: true}}
	for _, id := range []string{"1306101", "1306201", "1306202"} {
		skills = append(skills, gamekit.PanelSkill{ID: id, PointType: 1, Active: id != "1306202"})
	}
	image.Panel = gamekit.CharacterPanel{ID: "1306", Level: 80, Rank: 1, Element: "量子", Skills: skills,
		Weapon: &gamekit.PanelEquipment{ID: "23003", Name: "但战斗还未结束", Level: 80, Refinement: 5, Main: []gamekit.PanelStat{{Key: "hpPlus", Value: "1164"}}}}
	game := app.Game
	game.Prefix = "*"
	drawn, ok := images.Panel(gamekit.ImageContext{Game: game, Now: time.Now()}, image)
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
