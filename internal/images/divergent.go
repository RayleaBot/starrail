package images

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/RayleaBot/starrail/internal/app"
)

// divergentArtwork maps the images the Divergent Universe pages name to
// StarRail-plugin's paths.
var divergentArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"tttgbnumber", "resources/common/font/tttgbnumber.ttf"},
	{"divergent-header", "resources/rogue/tourn/simulated-s3-bg-top.e5a5b60a.webp"},
	{"rogue-oblivious", "resources/rogue/oblivious_bg@2x.7b5d7b38.png"},
}

// divergentTitles are upstream's names for the weekly difficulty IDs.
var divergentTitles = []string{"", "I", "II", "III", "IV", "V"}

var (
	divergentIndexWord = regexp.MustCompile(`(一|二|三)`)
	divergentIndexes   = map[string]int{"一": 1, "二": 2, "三": 3}
)

// divergentHeader is the header every Divergent Universe page shares: the
// equilibrium level, this week's extra points and the current star rank.
func divergentHeader(resources *challengeResources, result app.QueryResult) map[string]any {
	basic, _ := result.Data["basic"].(map[string]any)
	brief, _ := basic["normal_record_brief"].(map[string]any)
	info, _ := brief["common_info_v2"].(map[string]any)
	return map[string]any{
		"uid": result.Role.UID, "season_level": app.Text(basic["season_level"]),
		"weekly_score": app.Text(basic["weekly_score"]), "weekly_score_max": app.Text(basic["weekly_score_max"]),
		"rank_level": app.Text(info["level"]), "rank_icon": resources.official(brief["icon"]), "rank_title": app.Text(brief["title"]),
	}
}

func newDivergentResources(context app.ImageContext) *challengeResources {
	resources := newChallengeResources(context)
	for _, item := range divergentArtwork {
		resources.local(item[0], item[1])
	}
	return resources
}

// Divergent draws 差分宇宙 the way StarRail-plugin's rogueTourn page does:
// the header, the Gallery of Possibilities and Stable Array rewards, the
// latest Protocol run's time, star level and lineup, and this week's
// first-clear reward and latest difficulty.
func Divergent(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	basic, _ := result.Data["basic"].(map[string]any)
	if basic == nil {
		return app.Image{}, false
	}
	resources := newDivergentResources(context)
	data := divergentHeader(resources, result)
	data["gallery"] = app.Text(basic["possibility_gallery_finished"]) + "/" + app.Text(basic["possibility_gallery_total"])
	data["tasks"] = app.Text(basic["season_task_finished"]) + "/" + app.Text(basic["season_task_total"])
	brief, _ := basic["normal_record_brief"].(map[string]any)
	info, _ := brief["common_info_v2"].(map[string]any)
	if exists, _ := info["exist_data"].(bool); exists {
		detail, _ := result.Data["normal_detail"].(map[string]any)
		records, _ := detail["records"].([]any)
		latest := map[string]any{}
		if len(records) > 0 {
			latest, _ = records[0].(map[string]any)
		}
		latestInfo, _ := latest["common_info_v2"].(map[string]any)
		data["latest"] = map[string]any{"time": divergentTime(latest["finish_time"]), "level": app.Text(latestInfo["level"]), "lineup": resources.lineup(latest)}
	}
	finished, _ := basic["finished_weekly"].(bool)
	data["weekly_finished"] = finished
	// Upstream names the difficulty by its ID when there is one.
	weekly, _ := basic["weekly_record_brief"].(map[string]any)
	weeklyInfo, _ := weekly["common_info_v2"].(map[string]any)
	title := app.Text(weekly["title"])
	if id := app.Int(weeklyInfo["id"]); id != 0 {
		title = ""
		if id > 0 && id < len(divergentTitles) {
			title = divergentTitles[id]
		}
	}
	data["weekly_title"] = title
	return app.Image{Template: "divergent", Data: data, Resources: resources.List}, true
}

// DivergentNormal draws one Protocol run the way StarRail-plugin's
// rogueTournNormal page does; the command word picks the first, second or
// third most recent run.
func DivergentNormal(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	detail, _ := result.Data["normal_detail"].(map[string]any)
	return divergentRun(context, result, detail, "差分宇宙·常规演算", false)
}

// DivergentWeek draws one weekly run the way StarRail-plugin's
// rogueTournWeek page does: 上期 or 上周 reads last week, the command word
// picks the run, and the week's name and effects head the page.
func DivergentWeek(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	key, period := "cur_week_detail", "本期"
	if strings.Contains(context.Word, "上期") || strings.Contains(context.Word, "上周") {
		key, period = "last_week_detail", "上期"
	}
	detail, _ := result.Data[key].(map[string]any)
	return divergentRun(context, result, detail, "差分宇宙·周期演算 ("+period+")", true)
}

// divergentRun is the page both run commands share: the run's lineup, time
// and star level, Path, weighted Curios, expanded Equations, Blessing counts
// by Path, Curios and the domains passed. Upstream answers in text when the
// run does not exist.
func divergentRun(context app.ImageContext, result app.QueryResult, detail map[string]any, title string, weekly bool) (app.Image, bool) {
	records, _ := detail["records"].([]any)
	index := 1
	if match := divergentIndexWord.FindStringSubmatch(context.Word); match != nil {
		index = divergentIndexes[match[1]]
	}
	if len(records) < index {
		return app.Image{}, false
	}
	record, _ := records[index-1].(map[string]any)
	resources := newDivergentResources(context)
	data := divergentHeader(resources, result)
	data["title"] = title
	info, _ := record["common_info_v2"].(map[string]any)
	persona, _ := record["persona_style"].(map[string]any)
	data["run"] = fmt.Sprintf("%d / %d", index, len(records))
	data["time"], data["level"], data["lineup"] = divergentTime(record["finish_time"]), app.Text(info["level"]), resources.lineup(record)
	data["persona"] = map[string]any{"icon": resources.official(persona["icon"]), "title": app.Text(persona["title"]),
		"level": app.Text(persona["style_level"]), "desc": officialRichText(app.Text(persona["desc"]))}
	hex := []any{}
	hexList, _ := record["hex_miracles"].([]any)
	for _, raw := range hexList {
		item, _ := raw.(map[string]any)
		hex = append(hex, map[string]any{"icon": resources.official(item["icon"]), "name": app.Text(item["name"])})
	}
	formulas := []any{}
	formulaList, _ := record["formula_list"].([]any)
	for _, raw := range formulaList {
		item, _ := raw.(map[string]any)
		formulas = append(formulas, map[string]any{"category": app.Text(item["formula_category"]), "icon": resources.official(item["icon"]),
			"sub_icon": resources.official(item["sub_icon"]), "name": app.Text(item["name"])})
	}
	buffs, buffCount := []any{}, 0
	buffList, _ := record["buffs"].([]any)
	for _, raw := range buffList {
		group, _ := raw.(map[string]any)
		baseType, _ := group["base_type"].(map[string]any)
		items, _ := group["items"].([]any)
		buffCount += len(items)
		buffs = append(buffs, map[string]any{"icon": resources.pathIcon(app.Text(baseType["name"])), "count": len(items)})
	}
	miracles := []any{}
	miracleList, _ := record["miracles"].([]any)
	for _, raw := range miracleList {
		item, _ := raw.(map[string]any)
		miracles = append(miracles, map[string]any{"icon": resources.official(item["icon"]), "name": app.Text(item["name"]), "rank": app.Text(item["rank"])})
	}
	rooms := []any{}
	roomList, _ := record["room_card_list"].([]any)
	for _, raw := range roomList {
		room, _ := raw.(map[string]any)
		rooms = append(rooms, map[string]any{"color": app.Text(room["bg_color"]), "icon": resources.official(room["icon_path"]),
			"name": app.Text(room["type_name"]), "level": app.Text(room["level"])})
	}
	data["hex_miracles"], data["formulas"], data["buffs"], data["buff_count"] = hex, formulas, buffs, buffCount
	data["miracles"], data["rooms"] = miracles, rooms
	if weekly {
		// Upstream drops the bullet the official text starts each effect with.
		effects := []any{}
		descs, _ := detail["weekly_buff_desc"].([]any)
		for _, raw := range descs {
			effects = append(effects, officialRichText(strings.TrimPrefix(app.Text(raw), "●")))
		}
		data["weekly"] = map[string]any{"name": app.Text(detail["weekly_name"]), "effects": effects}
	}
	return app.Image{Template: "divergent-run", Data: data, Resources: resources.List}, true
}

// divergentTime is upstream's formatTime: a zero-padded date and time.
func divergentTime(value any) string {
	fields, _ := value.(map[string]any)
	if fields == nil {
		return ""
	}
	return fmt.Sprintf("%s.%02d.%02d %02d:%02d:%02d", app.Text(fields["year"]), app.Int(fields["month"]), app.Int(fields["day"]),
		app.Int(fields["hour"]), app.Int(fields["minute"]), app.Int(fields["second"]))
}

var (
	officialTag   = regexp.MustCompile(`<[^>]*>`)
	officialColor = regexp.MustCompile(`^<(?:color=|span style="color:\s*)(#[0-9A-Fa-f]{3,8})"?;?"?>$`)
)

// officialRichText is an official description upstream prints as HTML,
// keeping only its line breaks, colours and bold text: everything else is
// escaped or dropped.
func officialRichText(text string) string {
	var out strings.Builder
	last := 0
	for _, bounds := range officialTag.FindAllStringIndex(text, -1) {
		out.WriteString(html.EscapeString(text[last:bounds[0]]))
		last = bounds[1]
		tag := strings.ToLower(text[bounds[0]:bounds[1]])
		switch {
		case tag == "<br>" || tag == "<br/>" || tag == "<br />":
			out.WriteString("<br>")
		case tag == "<b>" || tag == "</b>" || tag == "</span>":
			out.WriteString(tag)
		case tag == "</color>":
			out.WriteString("</span>")
		default:
			if match := officialColor.FindStringSubmatch(text[bounds[0]:bounds[1]]); match != nil {
				out.WriteString(`<span style="color: ` + match[1] + `">`)
			}
		}
	}
	out.WriteString(html.EscapeString(text[last:]))
	return out.String()
}
