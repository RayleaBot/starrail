package images

import (
	"strconv"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

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
	if !ok || image.Data["overview"] != true || image.Data["total"] != 85 || image.Data["pity_role"] != 5 || image.Data["current_total"] != 0 {
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
