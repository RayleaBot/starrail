package images

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/starrail/internal/app"
)

// ledgerCountArtwork maps the images the ledger-count page names to their
// sources: Yunzai's Star Rail logo, StarRail-plugin's copy of the number font
// Yunzai uses, and the banner miao gives every Star Rail character, which
// genshinLayout's header shows for 黑天鹅.
var ledgerCountArtwork = [][3]string{
	{"tttgbnumber", "starrail-plugin", "resources/common/font/tttgbnumber.ttf"},
	{"starrail-logo", "yunzai-genshin", "resources/StarRail/img/other/logo.png"},
	{"head-banner", "miao-plugin", "resources/meta-sr/character/common/imgs/banner.webp"},
}

// ledgerColors are Yunzai's source colours, by the action IDs its ledger
// model gives the Star Rail sources.
var (
	ledgerColors  = []string{"#73a9c6", "#d56565", "#70b2b4", "#bd9a5a", "#739970", "#7a6da7", "#597ea0", "#ffb6c1"}
	ledgerActions = map[string]int{"other": 0, "adventure_reward": 1, "space_reward": 2, "daily_reward": 3, "abyss_reward": 4, "mail_reward": 5, "event_reward": 6, "combat_reward": 7}
)

var ledgerYearWord = regexp.MustCompile(`(\d{4})年`)

// ledgerMonth is a saved month's Stellar Jade, passes and sources.
type ledgerMonth struct {
	month, jade, passes int
	groups              []any
}

// ledgerSource is one Stellar Jade source summed over the months.
type ledgerSource struct {
	action, color string
	num           int
}

// LedgerCount draws 星琼统计 the way Yunzai's html/ledger/ledger-count-sr
// does (StarRail-plugin has no such page): the saved months of the year the
// command names (去年, 今年 or 2025年), or else those among the last twelve;
// the total Stellar Jade, the pulls it and the passes buy, the best month and
// its Jade, the total passes, the best pass month and the two largest
// sources; then each month's Jade as a column chart and every source, named
// by its first four characters, as a ring with its legend. Upstream draws
// both charts with G2Plot; this draws them as SVG. Upstream picks the best
// pass month by comparing thousands of passes as text, which always names the
// first month; this compares the passes.
func LedgerCount(context app.ImageContext, stats app.MonthlyStats) (app.Image, bool) {
	now := context.Now.In(chinaTime)
	year := 0
	switch {
	case strings.Contains(stats.Word, "去年"):
		year = now.Year() - 1
	case strings.Contains(stats.Word, "今年"):
		year = now.Year()
	default:
		if match := ledgerYearWord.FindStringSubmatch(stats.Word); match != nil {
			year, _ = strconv.Atoi(match[1])
		}
	}
	saved := map[string]map[string]any{}
	for _, item := range stats.Months {
		saved[item.Month] = item.Data
	}
	// A year reads January onwards; the last twelve months read newest first.
	months := []ledgerMonth{}
	for index := range 12 {
		at := time.Date(now.Year(), now.Month()-time.Month(index), 1, 0, 0, 0, 0, chinaTime)
		if year != 0 {
			at = time.Date(year, time.Month(index+1), 1, 0, 0, 0, 0, chinaTime)
		}
		data, _ := saved[at.Format("2006-01")]["month_data"].(map[string]any)
		if data == nil {
			continue
		}
		groups, _ := data["group_by"].([]any)
		months = append(months, ledgerMonth{month: int(at.Month()), jade: app.Int(data["current_hcoin"]), passes: app.Int(data["current_rails_pass"]), groups: groups})
	}
	if len(months) == 0 {
		return app.Image{}, false
	}
	total, passes := 0, 0
	best, bestPasses := months[0], months[0]
	sources := []*ledgerSource{}
	byAction := map[string]*ledgerSource{}
	for _, month := range months {
		total, passes = total+month.jade, passes+month.passes
		if month.jade > best.jade {
			best = month
		}
		if month.passes > bestPasses.passes {
			bestPasses = month
		}
		for _, raw := range month.groups {
			group, _ := raw.(map[string]any)
			name := []rune(app.Text(group["action_name"]))
			action := string(name[:min(4, len(name))])
			if byAction[action] == nil {
				byAction[action] = &ledgerSource{action: action, color: ledgerColors[ledgerActions[app.Text(group["action"])]]}
				sources = append(sources, byAction[action])
			}
			byAction[action].num += app.Int(group["num"])
		}
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range ledgerCountArtwork {
		resources.Artwork(item[0], item[1], item[2])
	}
	labels, values := []string{}, []float64{}
	for index := range months {
		// The last twelve months arrive newest first; the columns run from
		// the oldest.
		month := months[index]
		if year == 0 {
			month = months[len(months)-1-index]
		}
		labels, values = append(labels, strconv.Itoa(month.month)+"月"), append(values, float64(month.jade))
	}
	// Upstream orders the sources by amount, keeping first-seen order on ties.
	slices.SortStableFunc(sources, func(a, b *ledgerSource) int { return b.num - a.num })
	amounts, colors, legend, top, sum := []float64{}, []string{}, []any{}, []any{}, 0
	for index, source := range sources {
		amounts, colors = append(amounts, float64(source.num)), append(colors, source.color)
		legend = append(legend, map[string]any{"color": source.color, "action": source.action, "num": source.num,
			"y": fmt.Sprintf("%.1f", 140+(float64(index)-float64(len(sources)-1)/2)*26)})
		if index < 2 {
			top = append(top, map[string]any{"action": source.action, "num": source.num})
		}
		sum += source.num
	}
	yearText := ""
	if year != 0 {
		yearText = strconv.Itoa(year) + "年-"
	}
	return app.Image{Template: "ledger-count", Data: map[string]any{
		"uid": stats.Role.UID, "year_text": yearText,
		"total": fmt.Sprintf("%d.%02dw", (total+50)/100/100, (total+50)/100%100), "gacha": strconv.Itoa((total + 160*passes + 80) / 160),
		"passes": strconv.Itoa(passes) + "张", "best_month": strconv.Itoa(best.month), "best_value": strconv.Itoa(best.jade),
		"best_passes_month": strconv.Itoa(bestPasses.month), "top": top, "legend": legend, "sum": strconv.Itoa(sum),
		"columns": app.G2Column(labels, values, 470, 300, [4]float64{40, 10, 30, 52}),
		// The ring's box is 115% as wide, 280px high and shifted 60px left.
		// Upstream labels a share of 2% or more.
		"ring": app.G2Ring(amounts, colors, 195, 140, 130, 91, 2),
		// Upstream notes it shows only twelve months once that many are saved.
		"more": year == 0 && len(stats.Months) >= 12,
	}, Resources: resources.List}, true
}
