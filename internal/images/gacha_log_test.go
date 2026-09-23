package images

import (
	"strconv"
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

func TestGachaLogFollowsYunzai(t *testing.T) {
	// Oldest first: Welt (standard) on pull 70, Sparkle on pull 20 after
	// him, then ten pulls whose sixth and tenth are four-stars.
	records := []gacha.Record{}
	add := func(count int, name, rank, itemType string) {
		for range count {
			records = append(records, gacha.Record{ID: strconv.Itoa(1000 + len(records)), GachaType: "11", Name: name, Rank: rank, ItemType: itemType, Time: "2024-02-10 12:00:00"})
		}
	}
	add(69, "锋镝", "3", "光锥")
	add(1, "瓦尔特", "5", "角色")
	add(19, "锋镝", "3", "光锥")
	add(1, "花火", "5", "角色")
	add(5, "锋镝", "3", "光锥")
	add(1, "桂乃芬", "4", "角色")
	add(3, "锋镝", "3", "光锥")
	add(1, "桂乃芬", "4", "角色")
	image, ok := GachaLog(app.ImageContext{}, app.GachaImage{UID: "100000001", Word: "角色记录", Archive: gacha.Archive{Records: records}})
	if !ok || image.Template != "gacha-log" || image.Data["pool"] != "11" || image.Data["all"] != 100 {
		t.Fatalf("image = %+v", image.Data)
	}
	// The newest pull is a four-star, so none is missing since: upstream's
	// sentinel let the next four-star overwrite that with 4.
	want := map[string]any{"未出五星": 10, "五星": 2, "五星平均": 45, "小保底不歪": "0.0%", "最非": 70, "未出四星": 0, "五星常驻": 1, "四星平均": 50, "UP平均": 90, "UP花费星琼": "1.44w", "最欧": 20}
	for _, line := range image.Data["lines"].([][]any) {
		for _, raw := range line {
			cell := raw.(map[string]any)
			if expected, ok := want[cell["label"].(string)]; ok && cell["num"] != expected {
				t.Errorf("%s = %v, want %v", cell["label"], cell["num"], expected)
			}
		}
	}
	cards := image.Data["cards"].([]any)
	sparkle, welt := cards[0].(map[string]any), cards[1].(map[string]any)
	if sparkle["num"] != 20 || sparkle["up"] != true || sparkle["class"] != "good" || welt["num"] != 70 || welt["up"] != false || welt["class"] != "normal" {
		t.Errorf("cards = %v", cards)
	}
	// An empty pool still draws, as upstream shows its zeros.
	image, ok = GachaLog(app.ImageContext{}, app.GachaImage{Word: "角色联动记录", Archive: gacha.Archive{Records: records}})
	if !ok || image.Data["pool"] != "21" || image.Data["all"] != 0 || image.Data["first_time"] != "" {
		t.Errorf("empty pool = %v", image.Data)
	}
}

func TestGachaLogPoolAndRateUp(t *testing.T) {
	for word, pool := range map[string]string{"角色联动记录": "21", "光锥联动分析": "22", "武器联动祈愿": "22", "新手池记录": "2", "常驻池池记录": "1", "光锥池池祈愿": "12", "抽奖池池记录": "11", "集录池池记录": "500"} {
		if got, _ := logPool(word); got != pool {
			t.Errorf("%s shows pool %s, want %s", word, got, pool)
		}
	}
	inside := gacha.Record{Name: "希儿", Time: "2023-10-30 12:00:00"}
	outside := gacha.Record{Name: "希儿", Time: "2024-03-01 12:00:00"}
	if !logRateUp(inside) || logRateUp(outside) || logRateUp(gacha.Record{Name: "白露"}) || !logRateUp(gacha.Record{Name: "花火"}) {
		t.Error("rate-up periods do not follow upstream")
	}
}

func TestGachaAllLogFollowsYunzai(t *testing.T) {
	records := []gacha.Record{}
	add := func(count int, pool, name, rank, itemType string) {
		for range count {
			records = append(records, gacha.Record{ID: strconv.Itoa(1000 + len(records)), GachaType: pool, Name: name, Rank: rank, ItemType: itemType, Time: "2024-05-01 10:00:00"})
		}
	}
	// Two five-stars in the character warp, one in the collaboration light
	// cone warp and a novice warp, which 全部记录 leaves out.
	add(30, "11", "锋镝", "3", "光锥")
	add(1, "11", "花火", "5", "角色")
	add(20, "11", "锋镝", "3", "光锥")
	add(1, "11", "姬子", "5", "角色")
	add(69, "22", "锋镝", "3", "光锥")
	add(1, "22", "无可取代的东西", "5", "光锥")
	add(10, "2", "锋镝", "3", "光锥")
	image, ok := GachaLog(app.ImageContext{}, app.GachaImage{UID: "100000001", Word: "全部记录", Archive: gacha.Archive{Records: records}})
	if !ok || image.Template != "gacha-all-log" || image.Data["pool"] != "11" {
		t.Fatalf("image = %v", image.Data)
	}
	logs := image.Data["logs"].([]any)
	if len(logs) != 2 {
		t.Fatalf("logs = %d", len(logs))
	}
	// Every block's five-star row is padded to the longest, and the light
	// cone warp colours its 70 pulls by its own pity of 80.
	cones := logs[1].(map[string]any)["cards"].([]any)
	if cones[0].(map[string]any)["class"] != "bad" || cones[1].(map[string]any)["null"] != true {
		t.Errorf("light cone cards = %v", cones)
	}
	if _, ok := GachaLog(app.ImageContext{}, app.GachaImage{Word: "全部记录", Archive: gacha.Archive{Records: records[len(records)-10:]}}); ok {
		t.Error("全部记录 without records in its pools should not draw")
	}
}
