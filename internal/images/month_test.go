package images_test

import (
	"strings"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func TestMonthFollowsStarRailPlugin(t *testing.T) {
	data := decodeJSON(t, `{"month":"202609","month_data":{"current_hcoin":4000,"current_rails_pass":18,"group_by":[{"action_name":"每日活跃","num":3000},{"action_name":"其他","num":1000},{"action_name":"邮件奖励","num":0}]},
		"day_data":{"current_hcoin":60,"current_rails_pass":0,"last_hcoin":180,"last_rails_pass":1}}`)
	image, ok := images.Month(gamekit.ImageContext{}, gamekit.QueryResult{Data: data})
	if !ok || image.Data["month_hcoin"] != "4000" || image.Data["yesterday_hcoin"] != "180" {
		t.Fatalf("image = %v", image.Data)
	}
	// Every source is in the legend with its amount; empty ones draw no
	// sector. The ring starts at the top with rounded corners.
	legend, sectors := image.Data["legend"].([]any), image.Data["sectors"].([]any)
	first := sectors[0].(map[string]any)
	if len(legend) != 3 || legend[2].(map[string]any)["name"] != "邮件奖励 0" || len(sectors) != 2 || first["label"] != "75%" || strings.Count(first["path"].(string), "A 10 10") != 4 {
		t.Errorf("legend = %v sectors = %v", legend, sectors)
	}
}
