package images_test

import (
	"reflect"
	"testing"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/assets"
	"github.com/RayleaBot/starrail/internal/gacha"
	"github.com/RayleaBot/starrail/internal/images"
)

func TestGachaStatFollowsGachaData(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pools := application.Game.Data.Resources.Pools
	first, second, third := pools[0], pools[1], pools[2]
	record := func(id, pool, at, name, kind, rank string) gacha.Record {
		return gacha.Record{ID: id, GachaType: pool, Time: at, Name: name, ItemType: kind, Rank: rank}
	}
	archive := gacha.Archive{Records: []gacha.Record{
		record("1", "11", first.From[:10]+" 20:00:00", "桂乃芬", "角色", "4"),
		record("2", "11", first.From[:10]+" 20:01:00", first.Characters5[0], "角色", "5"),
		// A pull in no period is not the rate-up; the pulls before it still
		// find their periods.
		record("3", "11", second.To[:10]+" 23:00:00", "姬子", "角色", "5"),
		record("4", "11", third.From[:10]+" 20:00:00", "桂乃芬", "角色", "4"),
		record("5", "11", third.From[:10]+" 20:01:00", "桂乃芬", "角色", "4"),
		record("6", "12", third.From[:10]+" 20:02:00", third.Weapons5[0], "光锥", "5"),
		record("7", "2", first.From[:10]+" 21:00:00", "桂乃芬", "角色", "4"),
		record("8", "21", third.From[:10]+" 21:00:00", "Saber", "角色", "5"),
	}}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog}
	stats := func(data map[string]any) []any {
		values := []any{}
		for _, stat := range data["stats"].([]any) {
			values = append(values, stat.(map[string]any)["value"])
		}
		return values
	}

	stat, ok := images.GachaStat(context, app.GachaImage{Word: "角色统计", Archive: archive})
	if !ok {
		t.Fatal("stat not drawn")
	}
	if got, want := stats(stat.Data), []any{5, 2, 1, 3, "5.0"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("stat totals = %v, want %v", got, want)
	}
	versions := stat.Data["versions"].([]any)
	newest, unknown, oldest := versions[0].(map[string]any), versions[1].(map[string]any), versions[2].(map[string]any)
	if len(versions) != 3 || newest["version"] != third.Version || stats(newest)[0] != 2 || unknown["version"] != "未知" || unknown["from"] != "" || oldest["version"] != first.Version {
		t.Fatalf("stat versions = %v", versions)
	}
	// 抽卡统计 reads both limited warps; 全部统计 includes collaboration
	// and standard warps, but not departure, in one block.
	up, _ := images.GachaStat(context, app.GachaImage{Word: "抽卡统计", Archive: archive})
	all, _ := images.GachaStat(context, app.GachaImage{Word: "全部统计", Archive: archive})
	if stats(up.Data)[0] != 6 || stats(all.Data)[0] != 7 || len(all.Data["versions"].([]any)) != 1 || all.Data["versions"].([]any)[0].(map[string]any)["version"] != "全部统计" {
		t.Fatalf("up = %v, all = %v", up.Data["stats"], all.Data["versions"])
	}

	for word, label := range map[string]string{"新手统计": "新手池", "联动统计": "联动池"} {
		page, ok := images.GachaStat(context, app.GachaImage{Word: word, Archive: archive})
		if !ok {
			t.Fatalf("%s not drawn", word)
		}
		versions := page.Data["versions"].([]any)
		if stats(page.Data)[0] != 1 || len(versions) != 1 || versions[0].(map[string]any)["version"] != label {
			t.Errorf("%s = %v", word, page.Data)
		}
	}
}
