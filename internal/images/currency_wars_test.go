package images_test

import (
	"testing"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/images"
)

func TestCurrencyWarsArchiveFollowsStarRailPlugin(t *testing.T) {
	data := decodeJSON(t, `{"grid_fight_brief":{"season_level":12,"division":{"level":3}},"grid_fight_archive_list":[{},{
		"brief":{"archive_type":"ArchiveTypePromotion","archive_rank":"GridFightArchiveRankSS","division":{"level":2}},
		"lineup":{"front_roles":[{"role_id":1310,"rarity":5,"star":2},{"role_id":1212,"rarity":4,"star":1}],"back_roles":[],
			"trait_list":[{"trait_id":90,"quality":2,"trait_count":3}],
			"damage_list":[{"id":1212,"damage":"8000","damage_type":"DamageTypeAvatar"},{"id":90,"damage":"1234567.8","damage_type":"DamageTypeTrait"},
				{"id":1310,"damage":"2000.5"},{"id":1,"damage":"10"},{"id":2,"damage":"9"},{"id":3,"damage":"8"}]}}]}`)
	// The word names the second record; there is no third.
	if _, ok := images.CurrencyWarsArchive(app.ImageContext{Word: "货币战绩三"}, app.QueryResult{Data: data}); ok {
		t.Fatal("third record drawn")
	}
	image, ok := images.CurrencyWarsArchive(app.ImageContext{Word: "货币战绩二"}, app.QueryResult{Data: data})
	if !ok {
		t.Fatal("archive")
	}
	record := image.Data["record"].(map[string]any)
	if image.Data["index"] != "2 / 2" || image.Data["rank_badge"] != "A2" || record["badge"] != "A1" || record["grade"] != "SS" || record["mode"] != "快速博弈" {
		t.Errorf("header = %v record = %v", image.Data, record)
	}
	// The five largest, largest first, in 万 from ten thousand, with bars
	// against the largest; a bond is found by its trait ID.
	damage := image.Data["damage"].([]any)
	first, second := damage[0].(map[string]any), damage[1].(map[string]any)
	if len(damage) != 5 || first["value"] != "123.5万" || first["percent"] != "100.00" || first["trait"] != true ||
		first["entity"].(map[string]any)["count"] != "3" || second["value"] != "8000" || second["percent"] != "0.65" {
		t.Errorf("damage = %v", damage)
	}
	if third := damage[2].(map[string]any); third["value"] != "2001" || len(third["entity"].(map[string]any)["stars"].([]any)) != 2 {
		t.Errorf("third = %v", third)
	}
}
