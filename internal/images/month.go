package images

import (
	"fmt"
	"math"
	"strconv"

	"github.com/RayleaBot/starrail/internal/app"
)

// monthColors is ECharts' default palette, which upstream's chart uses.
var monthColors = []string{"#5470c6", "#91cc75", "#fac858", "#ee6666", "#73c0de", "#3ba272", "#fc8452", "#9a60b4", "#ea7ccc"}

// monthArtwork maps the images the month page names to StarRail-plugin's
// paths.
var monthArtwork = [][2]string{
	{"tttgbnumber", "resources/common/font/tttgbnumber.ttf"},
	{"month-bg", "resources/month/kll.jpg"},
	{"month-hcoin", "resources/month/星琼.png"},
	{"month-pass", "resources/month/车票.png"},
}

// Month draws 月历 the way StarRail-plugin's month page does: the month, this
// month's, today's and yesterday's Stellar Jade and passes, and this month's
// Stellar Jade sources as a ring chart with a legend. Upstream draws the ring
// with ECharts; this draws the same ring (radius 70% to 100% of the 300px
// chart, centred at 65%, rounded 10px corners, white borders) as SVG.
func Month(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	month, _ := result.Data["month_data"].(map[string]any)
	day, _ := result.Data["day_data"].(map[string]any)
	if month == nil {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range monthArtwork {
		resources.Artwork(item[0], "starrail-plugin", item[1])
	}
	groups, _ := month["group_by"].([]any)
	total := 0
	for _, raw := range groups {
		group, _ := raw.(map[string]any)
		total += app.Int(group["num"])
	}
	const cx, cy, outer, inner, corner = 377.0, 150.0, 150.0, 105.0, 10.0
	legend, sectors := []any{}, []any{}
	angle := -math.Pi / 2
	for index, raw := range groups {
		group, _ := raw.(map[string]any)
		num := app.Int(group["num"])
		color := monthColors[index%len(monthColors)]
		legend = append(legend, map[string]any{"color": color, "name": app.Text(group["action_name"]) + " " + strconv.Itoa(num), "y": fmt.Sprintf("%.1f", 150+(float64(index)-float64(len(groups)-1)/2)*30)})
		if total == 0 || num == 0 {
			continue
		}
		share := float64(num) / float64(total)
		end := angle + share*2*math.Pi
		middle := (angle + end) / 2
		sectors = append(sectors, map[string]any{"path": roundedSector(cx, cy, inner, outer, corner, angle, end), "color": color,
			"label": strconv.Itoa(int(math.Round(share*100))) + "%", "x": fmt.Sprintf("%.1f", cx+(inner+outer)/2*math.Cos(middle)), "y": fmt.Sprintf("%.1f", cy+(inner+outer)/2*math.Sin(middle))})
		angle = end
	}
	return app.Image{Template: "month", Data: map[string]any{
		"month": app.Text(result.Data["month"]), "legend": legend, "sectors": sectors,
		"month_hcoin": app.Text(month["current_hcoin"]), "month_pass": app.Text(month["current_rails_pass"]),
		"today_hcoin": app.Text(day["current_hcoin"]), "today_pass": app.Text(day["current_rails_pass"]),
		"yesterday_hcoin": app.Text(day["last_hcoin"]), "yesterday_pass": app.Text(day["last_rails_pass"]),
	}, Resources: resources.List}, true
}

// roundedSector is the SVG path of a ring sector from angle a0 to a1
// (clockwise, radians) with rounded corners, as ECharts draws a pie item with
// borderRadius; a sector too thin for the corners is drawn square.
func roundedSector(cx, cy, r0, r1, c, a0, a1 float64) string {
	point := func(radius, at float64) string {
		return fmt.Sprintf("%.2f %.2f", cx+radius*math.Cos(at), cy+radius*math.Sin(at))
	}
	large := 0
	if a1-a0 > math.Pi {
		large = 1
	}
	if a1-a0 >= 2*math.Pi-1e-6 {
		// A full ring is two half rings without corners.
		top, bottom := -math.Pi/2, math.Pi/2
		return fmt.Sprintf("M %s A %.0f %.0f 0 1 1 %s A %.0f %.0f 0 1 1 %s M %s A %.0f %.0f 0 1 0 %s A %.0f %.0f 0 1 0 %s Z",
			point(r1, top), r1, r1, point(r1, bottom), r1, r1, point(r1, top), point(r0, top), r0, r0, point(r0, bottom), r0, r0, point(r0, top))
	}
	outerTurn, innerTurn := math.Asin(c/(r1-c)), math.Asin(c/(r0+c))
	if a1-a0 < 2*outerTurn {
		return fmt.Sprintf("M %s L %s A %.0f %.0f 0 %d 1 %s L %s A %.0f %.0f 0 %d 0 %s Z", point(r0, a0), point(r1, a0), r1, r1, large, point(r1, a1), point(r0, a1), r0, r0, large, point(r0, a0))
	}
	outerFoot, innerFoot := (r1-c)*math.Cos(outerTurn), (r0+c)*math.Cos(innerTurn)
	largeOuter, largeInner := 0, 0
	if a1-a0-2*outerTurn > math.Pi {
		largeOuter = 1
	}
	if a1-a0-2*innerTurn > math.Pi {
		largeInner = 1
	}
	return fmt.Sprintf("M %s L %s A %.0f %.0f 0 0 1 %s A %.0f %.0f 0 %d 1 %s A %.0f %.0f 0 0 1 %s L %s A %.0f %.0f 0 0 1 %s A %.0f %.0f 0 %d 0 %s A %.0f %.0f 0 0 1 %s Z",
		point(innerFoot, a0), point(outerFoot, a0), c, c, point(r1, a0+outerTurn), r1, r1, largeOuter, point(r1, a1-outerTurn), c, c, point(outerFoot, a1),
		point(innerFoot, a1), c, c, point(r0, a1-innerTurn), r0, r0, largeInner, point(r0, a0+innerTurn), c, c, point(innerFoot, a0))
}
