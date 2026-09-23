package app

import (
	"reflect"
	"testing"
	"time"
)

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
