package images_test

import (
	"testing"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/images"
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
		image, ok := images.Rogue(app.ImageContext{}, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
		if !ok || image.Data["begin"] != "2026.9.1" || image.Data["finished"] != "3" || image.Data["nickname"] != "开拓者" {
			t.Fatalf("%s: image = %v", key, image.Data)
		}
		run := image.Data["runs"].([]any)[0].(map[string]any)
		buff := run["buffs"].([]any)[0].(map[string]any)
		if run["time"] != "2026.9.5 20:3:9" || buff["name"] != "「巡猎」" || buff["items"].([]any)[0].(map[string]any)["evolved"] != true {
			t.Errorf("run = %v", run)
		}
	}
	if _, ok := images.Rogue(app.ImageContext{}, app.QueryResult{Data: map[string]any{}}); ok {
		t.Error("no record answers in text")
	}
}

func TestSwarmFollowsStarRailPlugin(t *testing.T) {
	data := decodeJSON(t, `{"role":{"nickname":"开拓者","level":70},
		"basic":{"cnt":{"narrow":7,"miracle":90,"event":60},"destiny":[{"level":1},{"level":2},{"level":3},{"level":4},{"level":5},{"level":6},{"level":7}]},
		"detail":{"records":[
			{"difficulty":5,"finish_time":{"year":2026,"month":9,"day":5,"hour":20,"minute":3,"second":9},"fury":{"type":0,"point":"12"},
			 "worm_weak":["弱化"],"cached_avatars":[{"icon":"http://insecure.example/a.png","level":80}],"blocks":[{"name":"战斗•虫群","num":3},{"name":"事件","num":2}]},
			{"difficulty":4,"fury":{"type":1,"point":"3"},"blocks":[{"name":"事件","num":1}]}]}}`)
	image, ok := images.Swarm(app.ImageContext{}, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: data})
	if !ok || image.Data["narrow"] != "7" || image.Data["event"] != "60" {
		t.Fatalf("image = %v", image.Data)
	}
	// Upstream reads the destiny list by index: 存护, 记忆, 虚无, 丰饶, 巡猎, 毁灭, 欢愉.
	radar := image.Data["radar"].(map[string]any)
	if radar["preservation"].(map[string]any)["level"] != "1" || radar["abundance"].(map[string]any)["level"] != "4" || radar["elation"].(map[string]any)["level"] != "7" {
		t.Errorf("radar = %v", radar)
	}
	runs := image.Data["runs"].([]any)
	first, second := runs[0].(map[string]any), runs[1].(map[string]any)
	if first["countdown"] != true || first["fury"] != "12" || second["countdown"] != false || second["fury"] != "3" {
		t.Errorf("fury = %v / %v", first, second)
	}
	if block := first["blocks"].([]any)[0].(map[string]any); block["name"] != "战斗•虫群" || block["num"] != "3" {
		t.Errorf("blocks = %v", first["blocks"])
	}
	if _, ok := images.Swarm(app.ImageContext{}, app.QueryResult{Data: map[string]any{}}); ok {
		t.Error("no record answers in text")
	}
}
