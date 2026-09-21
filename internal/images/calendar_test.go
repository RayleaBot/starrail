package images

import (
	"testing"
	"time"
)

func TestCalendarTimesFollowMiaoStarRail(t *testing.T) {
	contents := []any{
		map[string]any{"ann_id": "1", "title": "3.6版本「对月当歌」更新概览", "content": "■更新时间 2026/09/10 06:00:00 ■"},
		map[string]any{"ann_id": "2", "title": "「活动跃迁」", "img": "https://example.invalid/warp.png",
			"content": `<h1 style="">活动跃迁</h1><p>3.6版本更新后 - 2026/09/30 14:59:59</p>`},
		map[string]any{"ann_id": "3", "title": "限时活动", "content": `<h1 style="">活动时间</h1><p>2026/09/12 10:00:00 - 2026/09/20 03:59:59</p>`},
	}
	times, gacha := calendarTimes(contents)
	// A version update's 06:00 reads as 11:00.
	if times["2"] != [2]string{"2026-09-10 11:00:00", "2026-09-30 14:59:59"} || len(gacha) != 1 {
		t.Errorf("warp = %v, pictures = %v", times["2"], gacha)
	}
	if times["3"] != [2]string{"2026-09-12 10:00:00", "2026-09-20 03:59:59"} {
		t.Errorf("event = %v", times["3"])
	}
}

func TestCalendarRotationsCountFromMonday(t *testing.T) {
	// Tuesday 2026-09-22: the week's Monday is 09-21, 161 days after 04-13,
	// so Memory of Chaos began 35 days earlier, on 08-17.
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, chinaTime)
	periods := calendarAbyss(now, now.AddDate(0, 0, 7), [2]string{"2026-09-10 12:00:00", "2026-10-21 12:00:00"})
	found := map[string]calendarPeriod{}
	for _, period := range periods {
		found[period.kind] = period
	}
	if chaos := found["abyss-1"]; !chaos.start.Equal(time.Date(2026, 8, 17, 4, 0, 0, 0, chinaTime)) || !chaos.end.Equal(time.Date(2026, 9, 28, 4, 0, 0, 0, chinaTime)) {
		t.Errorf("chaos = %v", chaos)
	}
	if peak := found["abyss-4"]; !peak.start.Equal(time.Date(2026, 9, 10, 11, 0, 0, 0, chinaTime)) || !peak.end.Equal(time.Date(2026, 10, 21, 3, 59, 0, 0, chinaTime)) {
		t.Errorf("peak = %v", peak)
	}
}
