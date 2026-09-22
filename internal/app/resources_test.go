package app

import "testing"

func TestCalendarVersionDateAndCoverageBoundaries(t *testing.T) {
	a := &App{Game: testGame(t)}
	result, err := a.resourceQuery("calendar.query", map[string]any{"date": "2023-04-26", "version": "1.0"})
	if err != nil || result["total"].(int) != 1 || result["pools"].([]PoolInfo)[0].Characters5[0] != "希儿" {
		t.Fatal(result, err)
	}
	if _, err = a.resourceQuery("calendar.query", map[string]any{"date": "2026-02-30"}); err == nil {
		t.Fatal("invalid date accepted")
	}
	result, _ = a.resourceQuery("calendar.query", map[string]any{"version": "unknown"})
	if result["total"].(int) != 0 {
		t.Fatal("calendar extrapolated unknown version")
	}
}
