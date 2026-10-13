package app

import (
	"testing"
	"time"
)

func TestCurrentPoolFollowsChinaTime(t *testing.T) {
	pools := []PoolInfo{
		{Version: "4.4", Half: "下半", From: "2026-08-05 12:00:00", To: "2026-08-25 14:59:59"},
		{Version: "4.5", Half: "上半", From: "2026-08-26 06:00:00", To: "2026-09-12 11:59:59"},
		{Version: "4.5", Half: "下半", From: "2026-09-12 12:00:00", To: "2026-09-28 03:59:59"},
	}
	for _, tc := range []struct {
		at      time.Time
		index   int
		current bool
	}{
		{time.Date(2026, 9, 1, 4, 0, 0, 0, time.UTC), 1, true},
		{time.Date(2026, 8, 25, 19, 0, 0, 0, time.UTC), 1, true},
		{time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC), 2, false},
	} {
		pool, current := currentPool(pools, tc.at)
		if pool.Version != pools[tc.index].Version || pool.Half != pools[tc.index].Half || current != tc.current {
			t.Errorf("%s = %+v, %v", tc.at, pool, current)
		}
	}
	if pool, current := currentPool(nil, time.Now()); pool.Version != "" || pool.From != "" || pool.To != "" || current {
		t.Fatalf("empty = %+v, %v", pool, current)
	}
}

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
