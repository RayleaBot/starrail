package images_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/images"
)

func TestLedgerCountFollowsYunzaiStarRail(t *testing.T) {
	month := func(key, jade, passes string, groups ...any) app.SavedMonth {
		return app.SavedMonth{Month: key, Data: map[string]any{"month_data": map[string]any{
			"current_hcoin": json.Number(jade), "current_rails_pass": json.Number(passes), "group_by": groups}}}
	}
	group := func(action, name, num string) any {
		return map[string]any{"action": action, "action_name": name, "num": json.Number(num)}
	}
	stats := app.MonthlyStats{Months: []app.SavedMonth{
		month("2026-08", "8000", "10", group("daily_reward", "每日活跃奖励", "3000")),
		month("2026-09", "4000", "30", group("daily_reward", "每日活跃奖励", "1000"), group("space_reward", "模拟宇宙奖励", "500")),
	}}
	image, ok := images.LedgerCount(app.ImageContext{Now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}, stats)
	// Pulls count the Jade at 160 each plus every pass.
	if !ok || image.Data["total"] != "1.20w" || image.Data["gacha"] != "115" || image.Data["passes"] != "40张" ||
		image.Data["best_month"] != "8" || image.Data["best_passes_month"] != "9" {
		t.Fatalf("image = %v", image.Data)
	}
	// Sources go by the first four characters of their names.
	legend := image.Data["legend"].([]any)
	if len(legend) != 2 || legend[0].(map[string]any)["action"] != "每日活跃" || legend[0].(map[string]any)["num"] != 4000 {
		t.Errorf("legend = %v", legend)
	}
}
