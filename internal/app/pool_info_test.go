package app

import (
	"slices"
	"testing"
	"time"
)

func TestPoolOrderUsesStartTimeWithinVersion(t *testing.T) {
	pools := []PoolInfo{
		{Version: "3.8", Half: "上半", From: "2025-12-17 06:00:00"},
		{Version: "3.8", Half: "中场", From: "2026-01-07 12:00:00"},
		{Version: "3.8", Half: "下半", From: "2026-01-28 12:00:00"},
	}
	sortPoolsDesc(pools)
	if got := []string{pools[0].Half, pools[1].Half, pools[2].Half}; !slices.Equal(got, []string{"下半", "中场", "上半"}) {
		t.Fatal(got)
	}
}

func TestPoolOrderAndReprint(t *testing.T) {
	pools := []PoolInfo{
		{Version: "4.5", Half: "上半", From: "2026-09-01 12:00:00", To: "2026-09-21 11:59:59"},
		{Version: "4.5", Half: "下半", From: "2026-09-21 12:00:00", To: "2026-10-12 11:59:59"},
		{Version: "4.6", Half: "上半", From: "2026-10-12 12:00:00", To: "2026-11-01 11:59:59"},
	}
	sortPoolsDesc(pools)
	if pools[0].Version != "4.6" || pools[1].Half != "下半" {
		t.Fatal(pools)
	}
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, chinaZone)
	if got := poolReprint(pools, now); got == nil || !got.Up || got.Text != "该角色/武器正在UP" {
		t.Fatal(got)
	}
	end, _ := time.ParseInLocation(time.DateTime, pools[0].To, chinaZone)
	if got := poolReprint(pools, end.Add(10*24*time.Hour)); got == nil || got.Up || got.Text != "该角色/武器已 10 天未复刻" {
		t.Fatal(got)
	}
	if got := poolReprint(pools, now.AddDate(0, -2, 0)); got != nil {
		t.Fatal(got)
	}
}
