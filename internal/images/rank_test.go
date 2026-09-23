package images_test

import (
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/assets"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func TestCloudRankFollowsArk(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	game := application.Game
	game.Prefix = "*"
	character, _ := application.Catalog.Get("1102")
	damage := 64248.529
	entries := []app.CustomRankEntry{
		{UID: "10****543", Cons: 0, Talents: map[string]app.PanelTalent{"a": {Level: 6, Original: 6}}, Weapon: "23001", WeaponLevel: 80, WeaponAffix: 1,
			Sets: []string{"野穗伴行的快枪手"}, SetName: "快枪手4", Mark: 337.9, Damage: &damage},
		{UID: "10****107", Mark: 293.9},
		{UID: "10****494", Mark: 41.9},
	}
	drawn, ok := images.CloudRank(app.ImageContext{Game: game, Catalog: application.Catalog}, app.CloudRankImage{Character: character, Mode: "mark", DamageTitle: "战技伤害",
		Limit: "排序: 圣遗物评分 降序 / 筛选: 无", Number: 20, Entries: entries})
	if !ok || drawn.Template != "rank" || drawn.Data["title"] != "*希儿圣遗物评分排行" || drawn.Data["time"] != "全服数据" || drawn.Data["number"] != 20 {
		t.Fatalf("drawn = %+v", drawn.Data)
	}
	rows := drawn.Data["rows"].([]any)
	first := rows[0].(map[string]any)
	weapon := first["weapon"].(map[string]any)
	if first["uid"] != "10****543" || first["mark"] != "337.9" || weapon["name"] != "于夜色中" || weapon["affix"] != 1 || weapon["star"] != 5 || first["set_name"] != "快枪手4" {
		t.Fatalf("first row = %v", first)
	}
	if dmg := first["damage"].(map[string]any); dmg["title"] != "战技伤害" || dmg["value"] != "64,248.5" {
		t.Fatalf("damage = %v", dmg)
	}
	// ark grades the average of the six relics, MAX from 56 on.
	for index, grade := range []string{"MAX", "SSS", "D"} {
		if got := rows[index].(map[string]any)["grade"]; got != grade {
			t.Errorf("row %d grade %v, want %s", index, got, grade)
		}
	}
	// The talents ark does not send have no level.
	if talents := first["talents"].([]any); len(talents) != 4 || talents[0].(map[string]any)["level"] != 6 || talents[1].(map[string]any)["level"] != "" {
		t.Errorf("talents = %v", talents)
	}
	if _, has := rows[1].(map[string]any)["damage"]; has {
		t.Error("damage drawn without ark's dmg_avg")
	}
}
