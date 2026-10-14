package images

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/gacha"
)

func TestGachaGroupsCollaborationAndMaintenanceGaps(t *testing.T) {
	context := app.ImageContext{Game: app.Game{Data: &app.GameData{Resources: app.GameResources{
		CollabPools: []app.PoolInfo{
			{Version: "3.4", Half: "联动", Name: "一期", GachaType: "21", From: "2025-07-11 12:00:00", To: "2036-01-01 00:00:00", Characters5: []string{"Saber", "Archer"}},
			{Version: "4.4", Half: "联动", Name: "二期", GachaType: "21", From: "2026-07-24 12:00:00", To: "2036-01-01 00:00:00", Characters5: []string{"远坂凛", "吉尔伽美什"}},
		},
		Pools: []app.PoolInfo{{Version: "2.0", Name: "跃迁名", From: "2024-02-06 12:00:00", To: "2024-02-26 14:59:59", Characters5: []string{"花火"}}},
	}}}}
	for _, tc := range []struct{ word, kind, at, name, version string }{
		{"联动记录", "21", "2026-08-01 12:00:00", "远坂凛", "4.4"},
		{"联动记录", "21", "2025-08-01 12:00:00", "Saber", "3.4"},
		{"角色记录", "11", "2024-02-26 20:00:00", "花火", "2.0"},
		{"角色记录", "11", "2024-02-28 12:00:00", "花火", ""},
	} {
		image, ok := Gacha(context, app.GachaImage{Word: tc.word, Archive: gacha.Archive{Records: []gacha.Record{{ID: "1", GachaType: tc.kind, Time: tc.at, Name: tc.name, Rank: "5", ItemType: "角色"}}}})
		if !ok {
			t.Fatal(tc)
		}
		group := image.Data["lists"].([]any)[0].(map[string]any)["groups"].([]any)[0].(map[string]any)
		pool := group["pool"].(map[string]any)
		if app.Text(pool["version"]) != tc.version || group["records"].([]any)[0].(map[string]any)["up"] != (tc.version != "") {
			t.Fatal(tc, group)
		}
		if tc.version != "" && pool["name"] == "" {
			t.Fatal(pool)
		}
		if tc.kind == "21" && len(image.Data["pities"].([]any)) != 1 {
			t.Fatal(image.Data["pities"])
		}
	}
}

func TestGachaSelectionReadsTheWord(t *testing.T) {
	for _, tc := range []struct {
		words []string
		pools []string
		all   bool
	}{
		{[]string{"跃迁记录", "up统计", "up记录", "喵喵抽卡记录", "集录池池统计", "版本统计"}, nil, false},
		{[]string{"全部记录", "全部光锥联动记录", "up全部统计"}, nil, true},
		{[]string{"角色联动池祈愿"}, []string{"21"}, false},
		{[]string{"武器联动分析"}, []string{"22"}, false},
		{[]string{"联动统计"}, []string{"21", "22"}, false},
		{[]string{"新手池统计"}, []string{"2"}, false},
		{[]string{"常驻池记录"}, []string{"1"}, false},
		{[]string{"角色记录100000001", "up角色祈愿", "角色光锥记录"}, []string{"11"}, false},
		{[]string{"光锥抽卡统计", "武器分析"}, []string{"12"}, false},
	} {
		for _, word := range tc.words {
			pools, all := gachaSelection(word)
			if !slices.Equal(pools, tc.pools) || all != tc.all {
				t.Errorf("%s = %v, %v; want %v, %v", word, pools, all, tc.pools, tc.all)
			}
		}
	}
}

func TestGachaHoldingRateCountsTheCatalog(t *testing.T) {
	context := app.ImageContext{Catalog: app.Catalog{Entries: []app.Entry{
		{ID: "1102", Kind: "character", Rarity: 5},
		{ID: "1001", Kind: "character", Rarity: 4},
		{ID: "2102", Kind: "character", Rarity: 5},
		{ID: "8001", Kind: "character", Rarity: 5},
		{ID: "23000", Kind: "weapon", Rarity: 5},
		{ID: "21000", Kind: "weapon", Rarity: 4},
		{ID: "20000", Kind: "weapon", Rarity: 3},
	}}}
	archive := gacha.Archive{Records: []gacha.Record{
		{ID: "1", GachaType: "11", Rank: "5"},
		{ID: "2", GachaType: "11", Rank: "5"},
	}}
	image, ok := Gacha(context, app.GachaImage{Word: "跃迁记录", Archive: archive})
	if !ok || image.Data["hold_rate"] != [2]int{0, 25} {
		t.Fatalf("holding rates = %v", image.Data)
	}
}

func TestGachaOverviewAndCollaborationPages(t *testing.T) {
	archive := gacha.Archive{Records: []gacha.Record{
		{ID: "1", GachaType: "21", Name: "Saber", ItemType: "角色", Rank: "5"},
		{ID: "2", GachaType: "22", Name: "没有回报的加冕", ItemType: "光锥", Rank: "5"},
		{ID: "3", GachaType: "2", Name: "桂乃芬", ItemType: "角色", Rank: "4"},
	}}
	for _, tc := range []struct {
		word     string
		overview bool
		pools    []string
		total    int
	}{
		{"全部记录", true, []string{"21", "22"}, 2},
		{"联动记录", false, []string{"21"}, 1},
		{"光锥联动记录", false, []string{"22"}, 1},
		{"联动光锥记录", false, []string{"22"}, 1},
		{"联动光锥分析", false, []string{"22"}, 1},
		{"联动武器记录", false, []string{"22"}, 1},
		{"新手记录", false, []string{"2"}, 1},
	} {
		image, ok := Gacha(app.ImageContext{}, app.GachaImage{Word: tc.word, Archive: archive})
		if !ok || image.Data["overview"] != tc.overview || image.Data["total"] != tc.total {
			t.Fatalf("%s = %v", tc.word, image.Data)
		}
		pools := []string{}
		for _, list := range image.Data["lists"].([]any) {
			pools = append(pools, list.(map[string]any)["type"].(string))
		}
		if !slices.Equal(pools, tc.pools) {
			t.Errorf("%s pools = %v, want %v", tc.word, pools, tc.pools)
		}
	}
}

func TestGachaFollowsStarRailPlugin(t *testing.T) {
	banner := app.PoolInfo{Version: "2.0", Half: "上半", From: "2024-02-06 12:00:00", To: "2024-02-26 14:59:59", Characters5: []string{"花火"}}
	context := app.ImageContext{Game: app.Game{Data: &app.GameData{Resources: app.GameResources{Pools: []app.PoolInfo{banner}}}}, Now: time.Date(2026, 9, 21, 12, 0, 0, 0, chinaTime)}
	records := []gacha.Record{}
	add := func(count int, name, rank, itemType string) {
		for range count {
			records = append(records, gacha.Record{ID: strconv.Itoa(5000 + len(records)), ItemID: "1306", GachaType: "11", Name: name, Rank: rank, ItemType: itemType, Time: "2024-02-10 12:00:00"})
		}
	}
	// Oldest first: a four-star on pull 10, Sparkle on pull 80, five more.
	add(9, "锋镝", "3", "光锥")
	add(1, "桂乃芬", "4", "角色")
	add(69, "锋镝", "3", "光锥")
	add(1, "花火", "5", "角色")
	add(5, "锋镝", "3", "光锥")

	image, ok := Gacha(context, app.GachaImage{UID: "100000001", Word: "跃迁记录", Archive: gacha.Archive{Records: records}})
	if !ok || image.Data["overview"] != true || image.Data["total"] != 85 || image.Data["pities"].([]any)[0].(map[string]any)["count"] != 5 || image.Data["pities"].([]any)[0].(map[string]any)["label"] != "角色未出五星" || image.Data["current_total"] != 0 {
		t.Fatalf("data = %v", image.Data)
	}
	if hold := image.Data["hold"].([2]int); hold != [2]int{1, 1} {
		t.Errorf("hold = %v", hold)
	}
	shown := []map[string]any{}
	for _, raw := range image.Data["lists"].([]any)[0].(map[string]any)["records"].([]any) {
		if record := raw.(map[string]any); record["show"] == true {
			shown = append(shown, record)
		}
	}
	// The overview shows five-stars only.
	if len(shown) != 1 || shown[0]["until"] != 80 || shown[0]["up"] != true {
		t.Errorf("overview cards = %v", shown)
	}

	image, _ = Gacha(context, app.GachaImage{Word: "角色记录", Archive: gacha.Archive{Records: records}})
	groups := image.Data["lists"].([]any)[0].(map[string]any)["groups"].([]any)
	group := groups[0].(map[string]any)
	if image.Data["overview"] != false || len(groups) != 1 || group["total"] != 85 || group["pool"].(map[string]any)["version"] != "2.0" {
		t.Fatalf("groups = %v", groups)
	}
	shown = shown[:0]
	for _, raw := range group["records"].([]any) {
		if record := raw.(map[string]any); record["show"] == true {
			shown = append(shown, record)
		}
	}
	if len(shown) != 2 || shown[0]["name"] != "花火" || shown[1]["until"] != 10 {
		t.Errorf("pool cards = %v", shown)
	}
}
