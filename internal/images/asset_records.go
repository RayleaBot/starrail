package images

import (
	"strconv"

	"github.com/RayleaBot/starrail/internal/app"
)

// chargeArtwork maps the images StarRail-plugin's charge page and its
// default layout name.
var chargeArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"charge-bg", "resources/charge/bg_main.b435ab02..png"},
	{"charge-deco", "resources/charge/Deco4.png"},
}

// PayLog draws 星琼记录 and the like the way StarRail-plugin's charge page
// does: the UID, the record's name and English subtitle over the background,
// and the page's gains with their time and reason.
func PayLog(context app.ImageContext, log app.PayLogImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	for _, item := range chargeArtwork {
		resources.Artwork(item[0], "starrail-plugin", item[1])
	}
	list := []any{}
	for _, record := range log.Records {
		list = append(list, map[string]any{"time": record.Time, "add_num": record.Count, "action": record.Action})
	}
	return app.Image{Template: "charge", Data: map[string]any{"uid": log.UID, "type": log.Type, "typeen": log.TypeEN, "page": strconv.Itoa(log.Page), "list": list}, Resources: resources.List}, true
}

// onlineArtwork maps the images and fonts StarRail-plugin's online page
// names.
var onlineArtwork = [][2]string{
	{"hanyi", "resources/common/font/汉仪润圆-75W.ttf"},
	{"tttgbnumber", "resources/common/font/tttgbnumber.ttf"},
	{"online-logo", "resources/common/cont/logo.png"},
	{"online-pamu", "resources/common/cont/pamu.png"},
	{"online-blocks", "resources/online/jeremy-thomas-E0AHdsENmDg-unsplash.jpg"},
}

// regionNames are the server names the official player card shows.
var regionNames = map[string]string{"prod_gf_cn": "星穹列车", "prod_qd_cn": "无名客", "prod_official_usa": "America", "prod_official_eur": "Europe", "prod_official_asia": "Asia", "prod_official_cht": "TW, HK, MO"}

// regionName is a region's server name, or the region for one without.
func regionName(region string) string {
	if name := regionNames[region]; name != "" {
		return name
	}
	return region
}

// Online draws 在线时长 the way StarRail-plugin's online page does: the player
// card, the last seven days' hours as a bar chart and every day's as a
// calendar heatmap, both of which upstream draws with ECharts and this draws
// as the same SVG, and the last seven days' online blocks. Upstream fixes
// the heatmap to 2023, the year it was written; this shows the current one.
func Online(context app.ImageContext, online app.OnlineImage) (app.Image, bool) {
	if len(online.Dates) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range onlineArtwork {
		resources.Artwork(item[0], "starrail-plugin", item[1])
	}
	values := make([]float64, len(online.Hours))
	byDate := map[string]float64{}
	for index, text := range online.Hours {
		values[index], _ = strconv.ParseFloat(text, 64)
		byDate[online.Dates[index]] = values[index]
	}
	from := max(0, len(values)-7)
	labels := make([]string, 0, 7)
	for index := from; index < len(values); index++ {
		labels = append(labels, online.Hours[index])
	}
	bar := app.EChartsBar(labels, values[from:], 800, 400, [4]float64{60, 80, 60, 80}, 5)
	for index, raw := range bar["bars"].([]any) {
		raw.(map[string]any)["date"] = online.Dates[from+index]
	}
	year, _ := strconv.Atoi(online.Dates[len(online.Dates)-1][:4])
	calendar := app.EChartsCalendar(year, byDate, 0, 24, 5, 700, [3]float64{30, 120, 30}, 16)
	details := []any{}
	for _, day := range online.Details {
		blocks := []any{}
		for _, block := range day.Blocks {
			status := "offline"
			if block.Online {
				status = "online"
			}
			blocks = append(blocks, map[string]any{"status": status, "width": strconv.Itoa(block.Width)})
		}
		details = append(details, map[string]any{"date": day.Date, "blocks": blocks})
	}
	return app.Image{Template: "online", Data: map[string]any{"nickname": online.Role.Nickname, "game_uid": online.Role.UID, "region_name": regionName(online.Role.Region), "level": strconv.Itoa(online.Role.Level),
		"bar": bar, "calendar": calendar, "details": details}, Resources: resources.List}, true
}
