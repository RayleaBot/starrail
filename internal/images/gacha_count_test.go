package images_test

import (
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/assets"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func TestGachaCountFollowsYunzai(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	events := application.Game.Data.Resources.Pools
	first, second, third := events[0], events[1], events[2]
	record := func(id, pool, at, name, kind, rank string) gacha.Record {
		return gacha.Record{ID: id, GachaType: pool, Time: at, Name: name, ItemType: kind, Rank: rank}
	}
	archive := gacha.Archive{Records: []gacha.Record{
		record("1", "11", first.From[:10]+" 20:00:00", "桂乃芬", "角色", "4"),
		record("2", "11", first.From[:10]+" 20:01:00", "桂乃芬", "角色", "4"),
		record("3", "11", first.From[:10]+" 20:02:00", "锋镝", "光锥", "3"),
		record("4", "11", second.From, "彦卿", "角色", "5"),
		record("5", "12", second.From, "唯有沉默", "光锥", "4"),
		// A pull between two banners ends the counting, as upstream drops
		// every pool before the one a pull falls in.
		record("6", "11", second.To[:10]+" 23:00:00", "桂乃芬", "角色", "4"),
		record("7", "11", third.From, "桂乃芬", "角色", "4"),
	}}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog}
	image, ok := images.GachaCount(context, app.GachaImage{Word: "角色池统计", Archive: archive})
	if !ok {
		t.Fatal("not drawn")
	}
	pools := image.Data["pools"].([]any)
	if len(pools) != 2 {
		t.Fatalf("pools = %v", pools)
	}
	newest, oldest := pools[0].(map[string]any), pools[1].(map[string]any)
	if newest["count"] != 1 || oldest["count"] != 3 || len(oldest["items"].([]any)) != 1 || oldest["items"].([]any)[0].(map[string]any)["count"] != 2 {
		t.Fatalf("pools = %v", pools)
	}
	if image.Data["type_name"] != "角色" || image.Data["dated"] != true {
		t.Fatalf("data = %v", image.Data)
	}
	image, ok = images.GachaCount(context, app.GachaImage{Word: "光锥池统计", Archive: archive})
	if !ok || image.Data["type_name"] != "光锥" || image.Data["pools"].([]any)[0].(map[string]any)["count"] != 1 {
		t.Fatalf("光锥池统计 = %v", image.Data)
	}
}
