package images

import (
	"cmp"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

// panelArtwork maps the miao images named in the panel stylesheet to their
// repository paths; the stylesheet reads them as --render-resource-<id>.
var panelArtwork = [][2]string{
	{"character-imgs-crown-sr", "resources/character/imgs/crown-sr.webp"},
	{"character-imgs-icon-sr", "resources/character/imgs/icon-sr.png"},
	{"character-imgs-up-num-icon0", "resources/character/imgs/up-num-icon0.png"},
	{"character-imgs-up-num-icon1", "resources/character/imgs/up-num-icon1.png"},
	{"character-imgs-up-num-icon2", "resources/character/imgs/up-num-icon2.png"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-star", "resources/common/item/star.png"},
	{"meta-sr-public-icons-attr-atk", "resources/meta-sr/public/icons/attr-atk.webp"},
	{"meta-sr-public-icons-attr-def", "resources/meta-sr/public/icons/attr-def.webp"},
	{"meta-sr-public-icons-attr-hp", "resources/meta-sr/public/icons/attr-hp.webp"},
	// miao's own font families.
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
}

// elementClasses is miao's element class for each Star Rail element.
var elementClasses = map[string]string{"火": "pyro", "冰": "cryo", "风": "anemo", "雷": "electro", "量子": "quantum", "虚数": "geo", "物理": "sr"}

// elementDamageIDs are the official property IDs of each element's damage bonus.
var elementDamageIDs = map[string]string{"物理": "12", "火": "14", "冰": "16", "雷": "18", "风": "20", "量子": "22", "虚数": "24"}

// panelAttrs are the attribute rows of miao's Star Rail panel, with the
// official property IDs they read; Elation has no official field yet.
var panelAttrs = []struct{ key, title, id string }{
	{"hp", "生命值", "1"}, {"atk", "攻击力", "2"}, {"def", "防御力", "3"}, {"speed", "速度", "4"},
	{"cpct", "暴击率", "5"}, {"cdmg", "暴击伤害", "6"}, {"dmg", "伤害加成", ""}, {"stance", "击破特攻", "58"},
	{"effPct", "效果命中", "10"}, {"effDef", "效果抵抗", "11"}, {"heal", "治疗加成", "7"}, {"joy", "欢愉度", ""},
}

// shortTitles shortens the combined substat list as miao does.
var shortTitles = map[string]string{"暴击率": "暴击", "暴击伤害": "爆伤", "充能效率": "充能", "元素精通": "精通", "效果命中": "命中", "效果抵抗": "抵抗", "击破特攻": "击破", "大生命": "生命", "大攻击": "攻击", "大防御": "防御", "小生命": "生命", "小攻击": "攻击", "小防御": "防御"}

var (
	skillPlaceholder = regexp.MustCompile(`\$(\d)\[(i|f1|f2)\](%?)`)
	nobr             = regexp.MustCompile(`<nobr>(.*?)</nobr>`)
	markup           = regexp.MustCompile(`<[^>]*>`)
	treeSuffix       = regexp.MustCompile(`([12][01][0-9])$`)
)

// chinaTime is the time zone the upstream images print times in.
var chinaTime = time.FixedZone("UTC+8", 8*3600)

// panelSources are miao's names for the services a panel came from; a panel
// ark hands over keeps the name miao gave it there.
var panelSources = map[string]string{"mihoyo": "mysPanelHSR", "enka": "enka", "mihomo": "homo", "change": "面板变换", "share": "share"}

// Panel draws a single character the way miao's profile-detail does for Star
// Rail: the splash with level, eidolons and attributes, eidolon icons,
// skills and traces, the light cone, the relic rating with every substat,
// the six relics, and the reference damage table.
func Panel(context app.ImageContext, image app.PanelImage) (app.Image, bool) {
	panel, record := image.Panel, image.Record
	if record.Name == "" {
		return app.Image{}, false
	}
	resources := []rayleabot.RenderImageResource{}
	add := func(id, name string) {
		if resource, ok := context.ArtworkResource(id, "miao-plugin", name); ok {
			resources = append(resources, resource)
		}
	}
	for _, item := range panelArtwork {
		add(item[0], item[1])
	}
	elem := elementClasses[record.Element]
	add("common-bg-bg-"+elem, "resources/common/bg/bg-"+elem+".webp")
	add("common-bg-talent-"+elem, "resources/common/bg/talent-"+elem+".webp")

	images := "resources/meta-sr/character/" + record.Name + "/imgs/"
	// An uploaded 面板图 the app picked replaces the splash, as miao's.
	if image.Splash.Path != "" {
		resources = append(resources, image.Splash)
	} else {
		add("splash", images+"splash.webp")
	}
	talentKeys := [][2]string{{"a", "普攻"}, {"e", "战技"}, {"t", "天赋"}, {"q", "爆发"}}
	switch record.WeaponType {
	case "记忆":
		talentKeys = append(talentKeys, [2]string{"me", "忆灵技"}, [2]string{"mt", "忆灵天赋"})
	case "欢愉":
		talentKeys = append(talentKeys, [2]string{"xe", "欢愉技"})
	}
	levels := app.PanelTalents(panel, record)
	talents := []any{}
	for _, talent := range talentKeys {
		level := levels[talent[0]]
		add("talent-"+talent[0], images+"talent-"+talent[0]+".webp")
		talents = append(talents, map[string]any{"icon": "talent-" + talent[0], "name": talent[1], "level": level.Level, "plus": level.Level > level.Original, "crown": level.Original >= 10})
	}
	// Eidolons 3 and 5 show the skill they raise, as upstream does.
	talentCons, _ := record.Data["talentCons"].(map[string]any)
	cons := []any{}
	for index := 1; index <= 6; index++ {
		id := "cons-" + strconv.Itoa(index)
		file := images + id + ".webp"
		if raised := app.Text(talentCons[strconv.Itoa(index)]); index == 3 || index == 5 {
			file = images + "talent-" + raised + ".webp"
		}
		add(id, file)
		cons = append(cons, map[string]any{"icon": id, "off": index > panel.Rank})
	}

	// miao names a panel's source as its data services do, and a changed
	// panel 面板变换.
	updated := context.Now
	if panel.UpdatedAtMS > 0 {
		updated = time.UnixMilli(panel.UpdatedAtMS)
	}
	data := map[string]any{
		"mode": "profile", "elem": elem, "name": record.Name, "uid": image.UID, "level": panel.Level, "cons": panel.Rank,
		"memosprite": len(talentKeys) > 4, "talents": talents, "trees": traces(panel, record, images, add),
		"attrs": attributes(panel, record), "cons_icons": cons,
		"data_source": cmp.Or(panelSources[panel.Source], panel.Source), "update_time": updated.In(chinaTime).Format("2006-01-02 15:04:05"),
		"artifact_hint": context.Game.Prefix + abbreviation(context.Catalog, record) + "圣遗物",
		"damage_hint":   context.Game.Prefix + abbreviation(context.Catalog, record) + "伤害",
	}
	if weapon := panel.Weapon; weapon != nil {
		data["weapon"] = lightCone(context, weapon, add)
	}
	if detail := panel.ScoreDetail; detail != nil {
		data["rating"], data["artifacts"] = rating(panel, detail, add)
	}
	if image.Damage != nil {
		table := damage(image.Damage, image.Original)
		if rank := image.Rank; rank != nil {
			// ark adds its ranks as rows of the table.
			rows := table["rows"].([]any)
			for _, row := range rank.Rows {
				rows = append(rows, map[string]any{"index": len(rows) + 1, "title": row.Label, "full": row.Value})
			}
			table["rows"] = rows
			if chart := rank.Chart; chart != nil {
				artisMax := 0.0
				if chart.Artis != nil {
					artisMax = math.Trunc(chart.Artis.Top)
				}
				data["rank_chart"] = map[string]any{"hint": context.Game.Prefix + abbreviation(context.Catalog, record) + "排名统计", "places": chart.Places,
					"charts": []any{panelRankChart("damage", chart.Damage, "伤害评分", 100, 20), panelRankChart("artis", chart.Artis, "圣遗物评分", artisMax, 50)}}
			}
		}
		// ark's DealLongDmgTitle for a changed panel, over every row with its
		// own: 1 cuts a title past 20 widths, 2 lets them wrap.
		wrap := false
		for _, row := range table["rows"].([]any) {
			item := row.(map[string]any)
			title := strings.TrimSpace(item["title"].(string))
			switch image.LongTitles {
			case 1:
				item["title"] = cutDamageTitle(title)
			case 2:
				wrap = wrap || damageTitleWidth(title) > 20
			}
		}
		table["wrap"] = wrap
		data["damage"] = table
		if matrix := image.Damage.Baseline.Matrix; image.DamageMode {
			data["mode"] = "dmg"
			if matrix != nil {
				data["matrix"] = damageMatrixData(*matrix)
				// The buffs are those of the detail the trades are for.
				buffs := []any{}
				if detail := matrix.Index - 1; detail >= 0 && detail < len(image.Damage.Baseline.Results) {
					for _, buff := range image.Damage.Baseline.Results[detail].Buffs {
						title, text, _ := strings.Cut(strings.Replace(buff, ":", "：", 1), "：")
						buffs = append(buffs, map[string]any{"title": title, "text": text})
					}
				}
				data["buffs"] = buffs
			}
		}
	}
	if image.Change != "" {
		data["change"] = image.Change
	}
	return app.Image{Template: "panel", Data: data, Resources: resources}, true
}

// traces lays out the trace tree as upstream does: ten stat nodes in pairs
// around the three ability nodes, each lit when unlocked.
func traces(panel app.CharacterPanel, record reference.Character, images string, add func(id, name string)) []any {
	lit := map[string]bool{}
	for _, skill := range panel.Skills {
		if skill.Active && skill.PointType != 2 && skill.ID != "" {
			if match := treeSuffix.FindStringSubmatch(app.RecordSkillID(panel, record, skill)); match != nil {
				lit[match[1]] = true
			}
		}
	}
	statIcon := map[string]string{}
	tree, _ := record.Data["tree"].(map[string]any)
	for id, raw := range tree {
		if match := treeSuffix.FindStringSubmatch(id); match != nil && match[1][0] == '2' {
			node, _ := raw.(map[string]any)
			statIcon[match[1]] = "resources/meta-sr/public/icons/tree-" + app.Text(node["key"]) + ".webp"
		}
	}
	groups := make([][]any, 9)
	for index, position := range "0113355778" {
		id := strconv.Itoa(201 + index)
		icon := statIcon[id]
		if icon == "" {
			icon = "resources/meta-sr/public/icons/tree-cpct.webp"
		}
		add("tree-"+id, icon)
		slot := int(position - '0')
		groups[slot] = append(groups[slot], map[string]any{"icon": "tree-" + id, "on": lit[id]})
	}
	result := []any{}
	for slot, group := range groups {
		if slot == 2 || slot == 4 || slot == 6 {
			index := slot/2 - 1
			id := strconv.Itoa(101 + index)
			add("tree-"+id, images+"tree-"+strconv.Itoa(index+1)+".webp")
			result = append(result, map[string]any{"talent": true, "icon": "tree-" + id, "on": lit[id]})
			continue
		}
		result = append(result, map[string]any{"talent": false, "nodes": group})
	}
	return result
}

func attributes(panel app.CharacterPanel, record reference.Character) []any {
	stats := map[string]app.PanelStat{}
	for _, stat := range panel.Stats {
		stats[stat.ID] = stat
	}
	weights := map[string]float64{}
	if panel.ScoreDetail != nil {
		weights = panel.ScoreDetail.Weights
	}
	result := []any{}
	for _, row := range panelAttrs {
		id := row.id
		if row.key == "dmg" {
			id = elementDamageIDs[record.Element]
		}
		stat, known := stats[id]
		item := map[string]any{"key": row.key, "title": row.title}
		if known {
			base := statText(stat.Base)
			item["value"], item["base"], item["plus"], item["zero"] = statText(stat.Value), base, statText(stat.Added), base == "0.0%"
		}
		if w := weights[row.key]; w > 0 {
			item["weight"] = strconv.FormatFloat(w, 'f', -1, 64)
			item["gold"] = w >= 80
		}
		result = append(result, item)
	}
	return result
}

// lightCone is the light cone card: miao's splash, its three base stats and
// the skill text with this superimposition's values.
func lightCone(context app.ImageContext, weapon *app.PanelEquipment, add func(id, name string)) map[string]any {
	result := map[string]any{"name": weapon.Name, "level": weapon.Level, "affix": weapon.Refinement}
	for _, stat := range append(append([]app.PanelStat{}, weapon.Main...), weapon.Sub...) {
		if key := strings.TrimSuffix(stat.Key, "Plus"); key == "hp" || key == "atk" || key == "def" {
			result[key] = statText(stat.Value)
		}
	}
	if context.Game.Calc == nil {
		return result
	}
	for _, entry := range context.Game.Calc.Metadata().Weapons {
		if entry.ID != weapon.ID {
			continue
		}
		add("weapon-splash", "resources/meta-sr/weapon/"+entry.Type+"/"+entry.Name+"/splash.webp")
		skill, _ := entry.Data["skill"].(map[string]any)
		tables, _ := skill["tables"].(map[string]any)
		desc := skillPlaceholder.ReplaceAllStringFunc(app.Text(skill["desc"]), func(match string) string {
			parts := skillPlaceholder.FindStringSubmatch(match)
			series, _ := tables[parts[1]].([]any)
			index := weapon.Refinement - 1
			if index < 0 || index >= len(series) {
				return ""
			}
			value, _ := strconv.ParseFloat(app.Text(series[index]), 64)
			if parts[3] == "%" {
				digits := 1
				if parts[2] == "f2" {
					digits = 2
				}
				return strconv.FormatFloat(value, 'f', digits, 64) + "%"
			}
			return comma(value)
		})
		// Upstream prints the text as HTML; the template gets the plain text
		// and the <nobr> values as separate pieces.
		parts := []any{}
		last := 0
		for _, match := range nobr.FindAllStringSubmatchIndex(desc, -1) {
			parts = append(parts, map[string]any{"text": markup.ReplaceAllString(desc[last:match[0]], ""), "value": markup.ReplaceAllString(desc[match[2]:match[3]], "")})
			last = match[1]
		}
		parts = append(parts, map[string]any{"text": markup.ReplaceAllString(desc[last:], ""), "value": ""})
		result["desc"] = parts
	}
	return result
}

// rating is the relic rating block and the six relics.
func rating(panel app.CharacterPanel, detail *app.ScoreDetail, add func(id, name string)) (map[string]any, []any) {
	class := func(key string) string {
		switch w := detail.Weights[key]; {
		case w > 79.9:
			return "great"
		case w > 0:
			return "useful"
		}
		return "nouse"
	}
	all := []any{}
	for _, attr := range detail.AllAttrs {
		name := detail.Titles[attr.Key]
		if short := shortTitles[name]; short != "" {
			name = short
		}
		all = append(all, map[string]any{"class": class(attr.Key), "eff": attr.Eff, "title": name, "value": attr.Value})
	}
	// Upstream shows exactly nine cells: the nine best, padded when fewer.
	all = all[:min(len(all), 9)]
	for len(all) < 9 {
		all = append(all, map[string]any{})
	}
	pieces := []any{}
	for slot := 1; slot <= 6; slot++ {
		piece, scored := detail.Pieces[slot]
		var equipment *app.PanelEquipment
		for index := range panel.Equipment {
			if panel.Equipment[index].Slot == slot {
				equipment = &panel.Equipment[index]
			}
		}
		if !scored || equipment == nil {
			pieces = append(pieces, map[string]any{"empty": true})
			continue
		}
		id := "artifact-" + strconv.Itoa(slot)
		add(id, "resources/meta-sr/artifact/"+equipment.SetName+"/arti-"+strconv.Itoa(slot)+".webp")
		subs := []any{}
		for _, attr := range piece.Attrs {
			subs = append(subs, map[string]any{"class": class(attr.Key), "eff": attr.Eff, "up": attr.UpNum, "title": detail.Titles[attr.Key], "value": attr.Value})
		}
		pieces = append(pieces, map[string]any{"icon": id, "name": equipment.Name, "level": equipment.Level, "mark": piece.Mark, "grade": piece.Grade,
			"main": map[string]any{"title": detail.Titles[piece.Main.Key], "value": piece.Main.Value}, "attrs": subs})
	}
	return map[string]any{"mark": detail.Mark, "grade": detail.Grade, "rule": detail.Title, "all": all}, pieces
}

// damage is the damage table; a changed panel compares each result with
// original's of the same title, as ark does.
func damage(result, original *app.BuildResult) map[string]any {
	kept := map[string]app.BuildSkillResult{}
	if original != nil {
		for _, item := range original.Baseline.Results {
			if _, seen := kept[item.Title]; !seen {
				kept[item.Title] = item
			}
		}
	}
	rows := []any{}
	for index, item := range result.Baseline.Results {
		row := map[string]any{"index": index + 1, "title": item.Title}
		dmg, avg := damageTexts(item)
		if dmg != "" {
			row["dmg"], row["avg"] = dmg, avg
		} else {
			row["full"] = avg
		}
		if old, ok := kept[item.Title]; ok {
			oldDmg, oldAvg := damageTexts(old)
			row["dmg_diff"], row["avg_diff"] = damageDiff(dmg, oldDmg), damageDiff(avg, oldAvg)
		}
		rows = append(rows, row)
	}
	enemy := result.Baseline.EnemyName
	if enemy == "" {
		enemy = "小宝"
	}
	return map[string]any{"rows": rows, "enemy_level": result.EnemyLevel, "enemy_name": enemy, "created_by": result.Baseline.CreatedBy}
}

// damageTexts are a result's critical and average damage as the table shows
// them; a result with one number or a text shows it as the average.
func damageTexts(result app.BuildSkillResult) (string, string) {
	switch {
	case result.Critical != nil && result.Expected != nil:
		return comma(*result.Critical), comma(*result.Expected)
	case result.Expected != nil:
		return "", comma(*result.Expected)
	}
	return "", result.Text
}

// damageNumber reads a shown damage as ark's getDiff does: without its
// thousands separators, the number it starts with, NaN when none.
var damageNumber = regexp.MustCompile(`^\s*[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?`)

// damageDiff is ark's getDiff: the change from the kept panel's value in
// percent to one place, with an up or down arrow, or -- when either value is
// not a number or the kept one is 0. Upstream compares the fixed text with
// the number 0, so an unchanged value reads " ↓0%"; it reads 0.0% here, as
// meant.
func damageDiff(current, original string) map[string]any {
	read := func(text string) float64 {
		number, err := strconv.ParseFloat(damageNumber.FindString(strings.ReplaceAll(text, ",", "")), 64)
		if err != nil {
			return math.NaN()
		}
		return number
	}
	now, was := read(current), read(original)
	if math.IsNaN(now) || math.IsNaN(was) || was == 0 {
		return map[string]any{"text": "--", "class": "same"}
	}
	change := math.Round((now-was)/was*1000) / 10
	switch {
	case change > 0:
		return map[string]any{"text": " ↑" + strconv.FormatFloat(change, 'f', -1, 64) + "%", "class": "up"}
	case change < 0:
		return map[string]any{"text": " ↓" + strconv.FormatFloat(-change, 'f', -1, 64) + "%", "class": "down"}
	}
	return map[string]any{"text": "0.0%", "class": "same"}
}

// damageTitleWidth is how wide ark counts a title: 1, ( and ) take half.
func damageTitleWidth(title string) float64 {
	width := 0.0
	for _, char := range title {
		width += damageCharWidth(char)
	}
	return width
}

func damageCharWidth(char rune) float64 {
	if char == '1' || char == '(' || char == ')' {
		return 0.5
	}
	return 1
}

// cutDamageTitle cuts a title before the character that takes it past 20
// widths and adds "...", as ark's truncTitle.
func cutDamageTitle(title string) string {
	width := 0.0
	for index, char := range title {
		if width += damageCharWidth(char); width > 20 {
			return title[:index] + "..."
		}
	}
	return title
}

// panelRankXs are the x of the points ark draws in 排名统计: the bottom, the
// ranking percentiles from the lowest (100 - percentile) and the top.
var panelRankXs = []float64{0, 1, 5, 10, 30, 50, 70, 80, 90, 95, 99, 100}

// panelRankChart is one of ark-plugin's 排名统计 charts, as its ECharts
// option draws it: the distribution with the x labels reading 100 - x, the
// axis names, the panel's score written 15 right of and 5 above its place,
// and the area's gradient, which turns red from the panel's place for 2.5
// points (ark's stops fall out of order, and later stops cannot go back).
// A curve ark did not send draws 暂无数据.
func panelRankChart(id string, curve *app.RankCurve, name string, yMax, interval float64) map[string]any {
	if curve == nil {
		return map[string]any{}
	}
	values := []float64{0}
	for index := len(curve.Scores) - 1; index >= 0; index-- {
		values = append(values, curve.Scores[index])
	}
	values = append(values, curve.Top)
	chart, at := app.EChartsDistribution(panelRankXs, values, yMax, interval)
	left, top := at(0, yMax)
	right, bottom := at(100, 0)
	for _, tick := range chart["x_ticks"].([]any) {
		item := tick.(map[string]any)
		item["label"], item["y"] = strconv.FormatFloat(100-item["value"].(float64), 'f', -1, 64), bottom+8
	}
	for _, tick := range chart["y_ticks"].([]any) {
		item := tick.(map[string]any)
		item["label"], item["x"] = strconv.FormatFloat(item["value"].(float64), 'f', -1, 64), left-8
	}
	x, y := at(100-curve.Percent, curve.Score)
	chart["mark"] = map[string]any{"x": x + 15, "y": y - 5, "text": strconv.FormatFloat(curve.Score, 'f', 2, 64)}
	chart["x_name"] = map[string]any{"x": right + 15, "y": bottom}
	chart["y_name"] = map[string]any{"x": left, "y": top - 15, "text": name}
	place := (100 - curve.Percent) / 100
	cyan, red := map[string]any{"color": "rgb(51, 204, 204)", "opacity": 0.5}, map[string]any{"color": "rgb(255, 0, 0)", "opacity": 1}
	stops := []any{}
	for _, stop := range []struct {
		offset float64
		color  map[string]any
	}{{0, cyan}, {place, cyan}, {place - 0.025, red}, {place + 0.025, red}, {place + 0.025, cyan}, {1, cyan}} {
		stops = append(stops, map[string]any{"offset": stop.offset, "color": stop.color["color"], "opacity": stop.color["opacity"]})
	}
	chart["stops"], chart["id"] = stops, "rank-"+id
	return chart
}

// damageMatrixData is miao's 词条伤害计算 as ProfileDetail formats it: each
// trade's change of the average, and its average and critical damage.
func damageMatrixData(matrix app.DamageMatrix) map[string]any {
	critical := func(value *float64) string {
		if value == nil {
			return ""
		}
		return comma(*value)
	}
	rows := []any{}
	for index, row := range matrix.Rows {
		cells := []any{}
		for _, cell := range row {
			if cell.Type == "na" {
				cells = append(cells, map[string]any{"type": "na"})
				continue
			}
			change := comma(cell.Avg - matrix.Avg)
			if cell.Avg > matrix.Avg {
				change = "+" + change
			}
			cells = append(cells, map[string]any{"type": cell.Type, "val": change, "avg": comma(cell.Avg), "dmg": critical(cell.Dmg)})
		}
		rows = append(rows, map[string]any{"title": matrix.Attrs[index].Title, "text": matrix.Attrs[index].Text, "cells": cells})
	}
	return map[string]any{"index": matrix.Index, "title": matrix.Title, "avg": comma(matrix.Avg), "dmg": critical(matrix.Dmg), "attrs": matrix.Attrs, "rows": rows}
}

// statText shows official numbers the way miao does: thousands separators on
// whole numbers, percentages unchanged.
func statText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasSuffix(value, "%") {
		return value
	}
	number, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64)
	if err != nil {
		return value
	}
	return comma(number)
}

// comma formats a whole number with thousands separators, as miao's
// Format.comma does without decimals.
func comma(value float64) string {
	digits := strconv.FormatFloat(value, 'f', 0, 64)
	negative := strings.HasPrefix(digits, "-")
	digits = strings.TrimPrefix(digits, "-")
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	if negative {
		return "-" + digits
	}
	return digits
}

// abbreviation is the short name upstream uses in command hints and cards:
// miao's abbreviation when the name has one, otherwise the full name, as
// miao's character model does.
func abbreviation(catalog app.Catalog, record reference.Character) string {
	if entry, ok := catalog.Get(record.ID); ok && entry.Abbr != "" {
		return entry.Abbr
	}
	if abbr := app.Text(record.Data["abbr"]); abbr != "" {
		return abbr
	}
	return record.Name
}
