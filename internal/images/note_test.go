package images

import (
	"encoding/json"
	"testing"
	"time"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

func TestNoteFollowsStarRailPlugin(t *testing.T) {
	now := time.Date(2026, 9, 21, 22, 30, 0, 0, time.FixedZone("UTC+8", 8*3600))
	data := map[string]any{
		"current_stamina": json.Number("180"), "max_stamina": json.Number("240"), "stamina_recover_time": json.Number("21600"),
		"current_train_score": json.Number("500"), "max_train_score": json.Number("500"),
		"current_rogue_score": json.Number("14000"), "max_rogue_score": json.Number("14000"),
		"accepted_epedition_num": json.Number("2"), "total_expedition_num": json.Number("4"),
		"expeditions": []any{
			map[string]any{"name": "毁灭者的覆灭", "remaining_time": json.Number("0"), "avatars": []any{"https://act-webstatic.mihoyo.com/a.png", "https://act-webstatic.mihoyo.com/b.png"}},
			map[string]any{"name": "无名之人", "remaining_time": json.Number("36000"), "avatars": []any{"https://act-webstatic.mihoyo.com/c.png"}},
		},
	}
	queried := ""
	context := gamekit.ImageContext{Now: now, Query: func(operation string, _ map[string]any) (gamekit.QueryResult, error) {
		queried = operation
		return gamekit.QueryResult{Data: map[string]any{"is_sign": true}}, nil
	}}
	image, ok := Note(context, gamekit.QueryResult{Role: gamekit.Role{UID: "100000001", Nickname: "开拓者", Level: 70}, Data: data})
	if !ok || image.Template != "note" || queried != "starrail.sign_info" || image.Data["signed"] != true {
		t.Fatalf("image = %+v, queried %q", image, queried)
	}
	stamina := image.Data["stamina"].(map[string]any)
	if stamina["full"] != false || stamina["wait"] != "06小时00分钟 |" || stamina["day"] != "[明日]" || stamina["clock"] != "04:30完全恢复" || stamina["progress"] != "75.00%" {
		t.Errorf("stamina = %v", stamina)
	}
	expeditions := image.Data["expeditions"].([]any)
	first, second := expeditions[0].(map[string]any), expeditions[1].(map[string]any)
	if first["remaining"] != "已完成" || first["finished"] != true || len(first["avatars"].([]any)) != 2 || first["progress"] != "100.00%" {
		t.Errorf("first expedition = %v", first)
	}
	if second["remaining"] != "10时00分" || second["progress"] != "50.00%" {
		t.Errorf("second expedition = %v", second)
	}
	if image.Data["time"] != "2026-09-21 22:30:00 星期一" || image.Data["accepted"] != "2" || image.Data["train"] != "500/500" {
		t.Errorf("data = %v", image.Data)
	}
	if len(image.Resources) != 3 || image.Resources[0].URL == "" {
		t.Errorf("expedition avatars should be fetched by URL: %v", image.Resources)
	}
}
