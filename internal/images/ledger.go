package images

import (
	"math/rand/v2"
	"strconv"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// ledgerArtwork maps the images and font the Star Rail ledger page names to
// Yunzai's paths.
var ledgerArtwork = [][2]string{
	{"HYWenHei-55W", "resources/font/HYWenHei-55W.ttf"},
	{"ledger-bg", "resources/StarRail/img/note/bg.png"},
	{"ledger-uid-bg", "resources/StarRail/img/note/uidbg.png"},
	{"ledger-pass", "resources/StarRail/img/other/rails_pass.png"},
	{"ledger-chart", "resources/StarRail/img/other/chart.png"},
}

// ledgerJade is the Stellar Jade icon the page's stylesheet loads from the
// official site.
const ledgerJade = "https://webstatic.mihoyo.com/upload/event/2023/03/28/77cb5426637574ba524ac458fa963da0_8938800417123864478.png"

// ledgerFaces are the characters Yunzai's dealData picks the card's face
// from.
var ledgerFaces = []string{"希儿", "白露", "艾丝妲", "布洛妮娅", "姬子", "卡芙卡", "克拉拉", "停云", "佩拉", "黑塔", "希露瓦", "银狼"}

// Ledger draws 星琼 the way Yunzai's html/ledger/ledger-sr does: the UID and
// the month, dated to the day for this month, a random character's face on a
// card with today's date, this and last month's Stellar Jade with the pulls
// it buys at 160 each and the passes, and the month's sources, named by
// their first four characters, as a ring with a legend. Upstream draws the
// ring with G2Plot; this draws the same ring as SVG. genshinLayout's banner
// styles a header this page does not have.
func Ledger(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	month, _ := result.Data["month_data"].(map[string]any)
	if month == nil {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range ledgerArtwork {
		resources.Artwork(item[0], "yunzai-genshin", item[1])
	}
	face := resources.Artwork("ledger-face", "miao-plugin", "resources/meta-sr/character/"+ledgerFaces[rand.IntN(len(ledgerFaces))]+"/imgs/face.webp")
	jade := resources.URL("mihoyo", ledgerJade)
	// Upstream writes amounts over ten thousand in ten thousands.
	amount := func(value any, digits int) string {
		number := app.Int(value)
		if number > 10000 {
			return jsFixed(float64(number)/10000, digits) + " w"
		}
		return strconv.Itoa(number)
	}
	pulls := func(value any) string { return jsFixed(float64(app.Int(value))/160, 0) }

	groups, _ := month["group_by"].([]any)
	legend, values, colors, total := []any{}, []float64{}, []string{}, 0
	for _, raw := range groups {
		group, _ := raw.(map[string]any)
		name := []rune(app.Text(group["action_name"]))
		color := ledgerColors[ledgerActions[app.Text(group["action"])]]
		legend = append(legend, map[string]any{"color": color, "action": string(name[:min(4, len(name))]), "num": app.Int(group["num"]), "percent": app.Text(group["percent"])})
		values, colors = append(values, float64(app.Int(group["num"]))), append(colors, color)
		total += app.Int(group["num"])
	}
	now := context.Now.In(chinaTime)
	data := app.Int(result.Data["data_month"])
	day := strconv.Itoa(data%100) + "月"
	if data == now.Year()*100+int(now.Month()) {
		day += strconv.Itoa(now.Day()) + "号"
	}
	return app.Image{Template: "ledger", Data: map[string]any{
		"uid": result.Role.UID, "day": day, "face": face, "date": now.Format("2006年01月02日"), "weekday": weekdays[now.Weekday()], "jade": jade,
		"current_hcoin": amount(month["current_hcoin"], 2), "gacha": pulls(month["current_hcoin"]), "current_rails_pass": amount(month["current_rails_pass"], 1),
		"last_hcoin": amount(month["last_hcoin"], 2), "last_gacha": pulls(month["last_hcoin"]), "last_rails_pass": amount(month["last_rails_pass"], 1),
		// G2Plot's ring spans radius 0.7 to 1 of the 440px chart less its
		// 10px padding.
		"legend": legend, "slices": app.G2Ring(values, colors, 220, 220, 210, 147), "total": strconv.Itoa(total),
	}, Resources: resources.List}, true
}
