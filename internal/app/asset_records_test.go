package app

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestPayLogReadsCategoryDirectionAndPage(t *testing.T) {
	var mu sync.Mutex
	var inputs []map[string]any
	role := map[string]any{"ref": "role", "game": "starrail", "uid": "100000001", "region": "prod_gf_cn", "nickname": "开拓者", "level": 70}
	a := pluginApp(t)
	host := newSDKHost(t, a, func(request rayleabot.ServiceCallRequest, _ bool) (map[string]any, string) {
		switch request.Method {
		case "list":
			owner := map[string]any{"source_protocol": "onebot11", "bot_id": "bot", "actor_id": "u"}
			return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "account_label": "账号", "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
		case "execute":
			input := asObject(request.Params["input"])
			mu.Lock()
			inputs = append(inputs, input)
			mu.Unlock()
			return map[string]any{"operation": "starrail.billing", "role": role, "data": map[string]any{
				"items": []any{
					map[string]any{"add_num": "60", "action": "活动奖励"},
					map[string]any{"add_num": "-160", "action": "兑换"},
					map[string]any{"add_num": "1", "action": "随时间回复开拓力"},
				},
				"has_more": false,
			}}, ""
		}
		return nil, "plugin.method_not_found"
	})
	for _, tc := range []struct {
		message, word  string
		args           []string
		category, page string
		contains       []string
		excludes       string
	}{
		{"*光锥消耗记录2", "光锥消耗记录2", nil, "cone", "2", []string{"160 · 兑换"}, "活动奖励"},
		{"*资产记录 体力 全部", "资产记录", []string{"体力", "全部"}, "power", "1", []string{"-160", "60 · 活动奖励"}, "随时间回复开拓力"},
		{"*资产记录", "资产记录", nil, "", "", []string{"星琼、古老梦华、开拓力（体力）、遗器、光锥"}, ""},
	} {
		mu.Lock()
		before := len(inputs)
		mu.Unlock()
		end, _ := host.groupMessage(tc.message, tc.word, tc.args...)
		text := terminalText(end)
		mu.Lock()
		after := len(inputs)
		var input map[string]any
		if after > before {
			input = inputs[after-1]
		}
		mu.Unlock()
		if tc.category == "" {
			if after != before {
				t.Fatalf("%s executed: %v", tc.message, input)
			}
		} else if after != before+1 || asText(input["category"]) != tc.category || asText(input["page"]) != tc.page || len(input) != 2 {
			t.Fatalf("%s input = %v", tc.message, input)
		}
		for _, want := range tc.contains {
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q: %s", tc.message, want, text)
			}
		}
		if tc.excludes != "" && strings.Contains(text, tc.excludes) {
			t.Errorf("%s includes %q: %s", tc.message, tc.excludes, text)
		}
	}
}

func TestOnlineTimeFollowsStarRailPlugin(t *testing.T) {
	today := time.Date(2026, 9, 22, 21, 0, 0, 0, chinaZone)
	at := func(day int, clock string, count int) onlineRecord {
		when, _ := time.ParseInLocation(time.DateTime, "2026-09-"+time.Date(2026, 9, day, 0, 0, 0, 0, chinaZone).Format("02")+" "+clock, chinaZone)
		return onlineRecord{when, count}
	}
	// Newest first, as the official record lists them. A point recovered
	// with others after being offline marks no time.
	records := []onlineRecord{at(22, "10:06:00", 1), at(22, "00:03:00", 1), at(21, "08:00:00", 12), at(14, "12:00:00", 1)}
	days := onlinePeriods(records, today)
	if len(days) != 7 || days[0].Date != "2026-09-22" || days[6].Date != "2026-09-16" {
		t.Fatalf("days = %v", days)
	}
	// Upstream's scan leaves each block a minute short: 0–3 online reads as
	// two minutes, as it does upstream.
	want := []OnlineBlock{{true, 0}, {false, 269}, {true, 2}, {false, 376}}
	if !reflect.DeepEqual(days[0].Blocks, want) {
		t.Errorf("today = %v", days[0].Blocks)
	}
	if blocks := days[1].Blocks; len(blocks) != 2 || !blocks[1].Online || blocks[0].Width != 648 {
		t.Errorf("the minutes before midnight = %v", blocks)
	}
	dates, hours := onlineDays(records, today)
	if len(dates) != 9 || dates[0] != "2026-09-14" || hours[0] != "0.1" || hours[7] != "0.0" || hours[8] != "0.2" {
		t.Errorf("dates %v hours %v", dates, hours)
	}
}

func TestEChartsChartsFollowTheirDefaults(t *testing.T) {
	bar := EChartsBar([]string{"0.4", "3.1", "5.6"}, []float64{0.4, 3.1, 5.6}, 800, 400, [4]float64{60, 80, 60, 80}, 5)
	ticks, bars := bar["ticks"].([]any), bar["bars"].([]any)
	if len(ticks) != 7 || ticks[6].(map[string]any)["label"] != "6" || bars[1].(map[string]any)["color"] != "#91cc75" || bars[0].(map[string]any)["width"] != "170.7" {
		t.Errorf("bar = %v", bar)
	}
	calendar := EChartsCalendar(2026, map[string]float64{"2026-01-01": 4.8, "2026-01-02": 20}, 0, 24, 5, 700, [3]float64{30, 120, 30}, 16)
	pieces, filled := calendar["pieces"].([]any), calendar["filled"].([]any)
	if pieces[0].(map[string]any)["label"] != "0.0 - 4.8" || pieces[4].(map[string]any)["label"] != "19.2 - 24.0" || pieces[1].(map[string]any)["color"] != "rgba(231,185,141,1)" {
		t.Errorf("pieces = %v", pieces)
	}
	// 2026 starts on a Thursday and needs 53 weeks; 4.8 is the first
	// piece's upper bound.
	if calendar["cell_width"] != "12.1" || filled[0].(map[string]any)["color"] != "rgba(246,239,166,1)" || filled[1].(map[string]any)["color"] != "rgba(191,68,76,1)" || filled[0].(map[string]any)["y"] != "184" {
		t.Errorf("calendar = %v %v", calendar["cell_width"], filled)
	}
}
