package images

import (
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

// logWords are stripped from the command word before picking the pool, as
// Yunzai's GachaLog.getPool does.
var logWords = regexp.MustCompile(`#|抽卡|记录|祈愿|分析|池|原神|星铁|崩坏星穹铁道|铁道`)

// logPool is getPool for Star Rail: the pool a word shows and its name, the
// character warp by default. Star Rail has no 集录 warp; upstream still
// shows the empty 500 pool for it.
func logPool(word string) (string, string) {
	switch logWords.ReplaceAllString(word, "") {
	case "常驻":
		return "1", "常驻"
	case "武器", "光锥":
		return "12", "光锥"
	case "集录":
		return "500", "集录"
	case "角色联动":
		return "21", "角色联动"
	case "光锥联动", "武器联动":
		return "22", "光锥联动"
	case "新手":
		return "2", "新手"
	}
	return "11", "角色"
}

// logAllPools are the pools getAllGcLogData reads in turn for Star Rail,
// leaving out its 集录, a pool Star Rail records never have.
var logAllPools = [][2]string{{"11", "角色"}, {"21", "角色联动"}, {"12", "光锥"}, {"22", "光锥联动"}, {"1", "常驻"}}

// logRateUpPeriods are the five-stars checkIsUp counts as the rate-up only
// inside these periods, in the records' own UTC+8 time.
var logRateUpPeriods = map[string][][2]string{
	"希儿": {{"2023-04-26 06:00:00", "2023-05-17 17:59:59"}, {"2023-10-27 12:00:00", "2023-11-14 14:59:59"}},
	"刃":  {{"2023-07-19 06:00:00", "2023-08-09 11:59:59"}, {"2023-12-27 06:00:00", "2024-01-17 11:59:59"}},
	"符玄": {{"2023-09-20 12:00:00", "2023-10-10 14:59:59"}, {"2024-05-29 12:00:00", "2024-06-18 14:59:59"}},
	"银狼": {{"2023-06-07 06:00:00", "2023-06-28 11:59:59"}, {"2023-12-06 12:00:00", "2023-12-26 14:59:59"}, {"2025-02-05 12:00:00", "2025-02-25 14:59:59"}, {"2025-09-02 12:00:00", "2025-09-23 14:59:59"}},
	"银枝": {{"2023-12-06 12:00:00", "2023-12-26 14:59:59"}, {"2024-07-10 12:00:00", "2024-07-31 14:59:59"}},
	"云璃": {{"2024-07-31 06:00:00", "2024-08-21 11:59:59"}, {"2025-02-26 06:00:00", "2025-03-19 11:59:59"}},
}

// logRateUp is checkIsUp: whether a five-star character was the rate-up.
func logRateUp(record gacha.Record) bool {
	if slices.Contains(standardFiveStars, record.Name) {
		return false
	}
	periods, limited := logRateUpPeriods[record.Name]
	if !limited {
		return true
	}
	for _, period := range periods {
		if record.Time >= period[0] && record.Time <= period[1] {
			return true
		}
	}
	return false
}

// logArtwork are the fonts and images of Yunzai's gacha-log pages; their
// genshinLayout header shows the banner miao gives every Star Rail character.
var logArtwork = [][3]string{
	{"tttgbnumber", "yunzai-genshin", "resources/font/tttgbnumber.ttf"},
	{"HYWenHei-55W", "yunzai-genshin", "resources/font/HYWenHei-55W.ttf"},
	{"img-other-bg4", "yunzai-genshin", "resources/img/other/bg4.png"},
	{"img-other-bg5", "yunzai-genshin", "resources/img/other/bg5.png"},
	{"logo", "yunzai-genshin", "resources/StarRail/img/other/logo.png"},
	{"head-img", "miao-plugin", "resources/meta-sr/character/common/imgs/banner.webp"},
}

// GachaLog draws Yunzai's Star Rail gacha record: gacha-log's summary lines
// for one pool and every five-star with its pull count, or, for a word with
// 全部, gacha-all-log's block for every pool with records, each five-star row
// padded to the longest; false when no pool has records for 全部.
func GachaLog(context app.ImageContext, image app.GachaImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range logArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	icon := gachaIcon(context, resources)
	if !strings.Contains(image.Word, "全部") {
		pool, poolName := logPool(image.Word)
		return app.Image{Template: "gacha-log", Data: gachaLog(image, icon, pool, poolName), Resources: resources.List}, true
	}
	logs := []map[string]any{}
	most := 0
	for _, pool := range logAllPools {
		if log := gachaLog(image, icon, pool[0], pool[1]); log["all"] != 0 {
			logs = append(logs, log)
			most = max(most, len(log["cards"].([]any)))
		}
	}
	if len(logs) == 0 {
		return app.Image{}, false
	}
	// Upstream reads the label colour, the five-star period and the UP
	// badges' pool from the first block for every block.
	first := logs[0]
	list := []any{}
	for _, log := range logs {
		cards := log["cards"].([]any)
		for _, raw := range cards {
			card := raw.(map[string]any)
			card["up"] = card["rate_up"] == true && first["pool_name"] == "角色"
		}
		for len(cards) < most {
			cards = append(cards, map[string]any{"null": true})
		}
		log["cards"] = cards
		list = append(list, log)
	}
	return app.Image{Template: "gacha-all-log", Data: map[string]any{"pool": first["pool"], "first_time": first["first_time"], "last_time": first["last_time"], "logs": list},
		Resources: resources.List}, true
}

// gachaLog is GachaLog's analyse and randData for one pool: its pull count,
// the summary lines for its type and its five-stars, newest first.
func gachaLog(image app.GachaImage, icon func(name, itemType string) string, pool, poolName string) map[string]any {
	records := []gacha.Record{}
	for _, record := range image.Archive.Records {
		if record.GachaType == pool {
			records = append(records, record)
		}
	}
	// Upstream reads the log newest first; record IDs grow with time.
	sort.Slice(records, func(i, j int) bool {
		if len(records[i].ID) != len(records[j].ID) {
			return len(records[i].ID) > len(records[j].ID)
		}
		return records[i].ID > records[j].ID
	})

	type five struct {
		record gacha.Record
		num    int
		up     bool
	}
	fives := []five{}
	fourCounts, fourOrder := map[string]int{}, []string{}
	fiveNum, fourNum, fiveSince, fourSince, noFive, noFour := 0, 0, 0, 0, 0, 0
	wai, coneFive, coneFour, all := 0, 0, 0, len(records)
	for _, record := range records {
		if record.Rank == "4" {
			// Upstream tells the first four-star by noFour still being 0, so
			// a newest pull that was a four-star let the next one overwrite
			// it; the count is taken at the first four-star only.
			if fourNum == 0 {
				noFour = fourSince
			}
			fourNum++
			fourSince = 0
			if fourCounts[record.Name] == 0 {
				fourOrder = append(fourOrder, record.Name)
			}
			fourCounts[record.Name]++
			if record.ItemType == "光锥" {
				coneFour++
			}
		}
		fourSince++
		if record.Rank == "5" {
			fiveNum++
			if len(fives) > 0 {
				fives[len(fives)-1].num = fiveSince
			} else {
				noFive = fiveSince
			}
			fiveSince = 0
			up := false
			if record.ItemType == "角色" {
				if up = logRateUp(record); !up {
					wai++
				}
			} else {
				coneFive++
			}
			fives = append(fives, five{record: record, up: up})
		}
		fiveSince++
	}
	if len(fives) > 0 {
		fives[len(fives)-1].num = fiveSince
	} else {
		noFive = all
	}
	// Upstream drops the five-stars it could not name, with their pulls. It
	// removes them while walking the list, which skipped the five-star after
	// each; here every one is dropped before the guarantees are counted.
	known := fives[:0]
	for _, item := range fives {
		if item.record.Name == "未知" {
			all -= item.num
			fiveNum--
			continue
		}
		known = append(known, item)
	}
	fives = known
	// A five-star right after an off-banner one came from the guarantee.
	big := 0
	for index := range fives {
		if next := index + 1; next < len(fives) && !fives[next].up {
			big++
		}
	}
	sort.SliceStable(fourOrder, func(i, j int) bool { return fourCounts[fourOrder[i]] > fourCounts[fourOrder[j]] })
	maxFourName, maxFour := "无", 0
	if len(fourOrder) > 0 {
		maxFourName, maxFour = fourOrder[0], fourCounts[fourOrder[0]]
	}
	average := func(total, count int) int {
		if count == 0 {
			return 0
		}
		return int(math.Round(float64(total) / float64(count)))
	}
	fiveAvg, fourAvg := average(all-noFive, fiveNum), average(all-noFour, fourNum)
	valid := 0
	if fiveNum > 0 && fiveNum > wai {
		if len(fives) > 0 && !fives[0].up {
			valid = average(all-noFive-fives[0].num, fiveNum-wai)
		} else {
			valid = average(all-noFive, fiveNum-wai)
		}
	}
	upCost := strconv.Itoa(valid * 160)
	if valid*160 >= 10000 {
		upCost = app.JSFixed(float64(valid*160)/10000, 2) + "w"
	}
	noWaiRate := "0"
	if fiveNum > 0 {
		noWaiRate = app.JSFixed(float64(fiveNum-big-wai)/float64(fiveNum-big)*100, 1)
	}
	worst, best := 0, 0
	for _, item := range fives {
		if worst == 0 || item.num > worst {
			worst = item.num
		}
		if best == 0 || item.num < best {
			best = item.num
		}
	}
	cell := func(label string, num any, unit string) map[string]any {
		return map[string]any{"label": label, "num": num, "unit": unit}
	}
	fourUnit := []rune(maxFourName)
	fourUnit = fourUnit[:min(len(fourUnit), 4)]
	secondLine := []any{cell("未出四星", noFour, "抽"), cell("四星", fourNum, "个"), cell("四星平均", fourAvg, "抽"), cell("四星最多", maxFour, string(fourUnit)), cell("最欧", best, "抽")}
	var lines [][]any
	switch pool {
	case "11", "21":
		lines = [][]any{
			{cell("未出五星", noFive, "抽"), cell("五星", fiveNum, "个"), cell("五星平均", fiveAvg, "抽"), cell("小保底不歪", noWaiRate+"%", ""), cell("最非", worst, "抽")},
			{cell("未出四星", noFour, "抽"), cell("五星常驻", wai, "个"), cell("UP平均", valid, "抽"), cell("UP花费星琼", upCost, ""), cell("最欧", best, "抽")},
		}
	case "1", "2":
		lines = [][]any{{cell("未出五星", noFive, "抽"), cell("五星", fiveNum, "个"), cell("五星平均", fiveAvg, "抽"), cell("五星光锥", coneFive, "个"), cell("最非", worst, "抽")}, secondLine}
	default:
		lines = [][]any{{cell("未出五星", noFive, "抽"), cell("五星", fiveNum, "个"), cell("五星平均", fiveAvg, "抽"), cell("四星光锥", coneFour, "个"), cell("最非", worst, "抽")}, secondLine}
	}

	// A count is coloured by the pool's pity: 80 for the light cone warps,
	// else 90. getAllGcLogData gave 光锥联动 90, unlike its own page; each
	// block here keeps the pool's own pity.
	pity := 90
	if pool == "12" || pool == "22" {
		pity = 80
	}
	cards := []any{}
	for _, item := range fives {
		class := "bad"
		switch {
		case item.num <= 10:
			class = "gold"
		case float64(item.num) < float64(pity)*0.5:
			class = "good"
		case float64(item.num) < float64(pity)*0.83:
			class = "normal"
		}
		cards = append(cards, map[string]any{"class": class, "up": item.up && poolName == "角色", "rate_up": item.up, "icon": icon(item.record.Name, item.record.ItemType), "num": item.num})
	}
	firstTime, lastTime := "", ""
	if len(records) > 0 {
		firstTime, lastTime = records[len(records)-1].Time[:16], records[0].Time[:16]
	}
	return map[string]any{
		"uid": image.UID, "all": all, "pool": pool, "pool_name": poolName, "lines": lines, "cards": cards,
		"first_time": firstTime, "last_time": lastTime,
	}
}
