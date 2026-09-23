package images_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func TestLedgerFollowsYunzaiStarRail(t *testing.T) {
	group := func(action, name, num, percent string) any {
		return map[string]any{"action": action, "action_name": name, "num": json.Number(num), "percent": json.Number(percent)}
	}
	data := map[string]any{"data_month": "202609", "month_data": map[string]any{
		"current_hcoin": json.Number("12345"), "current_rails_pass": json.Number("12"), "last_hcoin": json.Number("8000"), "last_rails_pass": json.Number("20"),
		"group_by": []any{group("daily_reward", "每日活跃奖励", "9000", "73"), group("adventure_reward", "冒险奖励", "3345", "27")}}}
	context := app.ImageContext{Now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}
	image, ok := images.Ledger(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok {
		t.Fatal("ledger")
	}
	// Upstream dates this month to the day, writes ten thousands with a w
	// and counts pulls at 160 each.
	for key, want := range map[string]any{"day": "9月21号", "current_hcoin": "1.23 w", "gacha": "77", "current_rails_pass": "12", "last_hcoin": "8000", "last_gacha": "50", "total": "12345"} {
		if image.Data[key] != want {
			t.Errorf("%s = %v, want %v", key, image.Data[key], want)
		}
	}
	// Sources show their first four characters in their action's colour.
	if first := image.Data["legend"].([]any)[0].(map[string]any); first["action"] != "每日活跃" || first["color"] != "#bd9a5a" {
		t.Errorf("legend = %v", first)
	}
	data["data_month"] = "202607"
	if image, _ := images.Ledger(context, app.QueryResult{Data: data}); image.Data["day"] != "7月" {
		t.Errorf("day = %v", image.Data["day"])
	}
}
