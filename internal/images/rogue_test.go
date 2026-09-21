package images_test

import (
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func TestRogueFollowsStarRailPlugin(t *testing.T) {
	record := `{"basic":{"schedule_begin":{"year":2026,"month":9,"day":1},"schedule_end":{"year":2026,"month":9,"day":14},"finish_cnt":3},
		"records":[{"name":"第九世界","difficulty":5,"score":8000,"finish_time":{"year":2026,"month":9,"day":5,"hour":20,"minute":3,"second":9},
		"final_lineup":[{"rarity":5,"rank":1,"level":80,"element":"quantum","icon":"http://insecure.example/a.png"}],
		"base_type_list":[{"name":"「巡猎」","cnt":6}],
		"buffs":[{"base_type":{"name":"「巡猎」"},"items":[{"name":"祝福","rank":2,"is_evoluted":true}]}],"miracles":[{"icon":"http://insecure.example/m.png"}]}]}`
	// The kit keeps only the asked period, as either record.
	for _, key := range []string{"current_record", "last_record"} {
		data := decodeJSON(t, `{"role":{"nickname":"开拓者","level":70},"basic_info":{"unlocked_skill_points":40,"unlocked_miracle_num":100,"unlocked_buff_num":200},"`+key+`":`+record+`}`)
		image, ok := images.Rogue(gamekit.ImageContext{}, gamekit.QueryResult{Role: gamekit.Role{UID: "100000001"}, Data: data})
		if !ok || image.Data["begin"] != "2026.9.1" || image.Data["finished"] != "3" || image.Data["nickname"] != "开拓者" {
			t.Fatalf("%s: image = %v", key, image.Data)
		}
		run := image.Data["runs"].([]any)[0].(map[string]any)
		buff := run["buffs"].([]any)[0].(map[string]any)
		if run["time"] != "2026.9.5 20:3:9" || buff["name"] != "「巡猎」" || buff["items"].([]any)[0].(map[string]any)["evolved"] != true {
			t.Errorf("run = %v", run)
		}
	}
	if _, ok := images.Rogue(gamekit.ImageContext{}, gamekit.QueryResult{Data: map[string]any{}}); ok {
		t.Error("no record answers in text")
	}
}
