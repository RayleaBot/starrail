package images

import (
	"slices"
	"strconv"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// currencyWarsArtwork maps the images and font both 货币战争 pages name to
// StarRail-plugin's paths.
var currencyWarsArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"grid-fight-header", "resources/gridFight/entry-bg-top.253f0fd5.webp"},
	{"grid-fight-record", "resources/gridFight/record-header-bg--home.c09dc5c8.png"},
	{"grid-fight-heart", "resources/gridFight/gridFight/heart.webp"},
	{"grid-fight-money", "resources/gridFight/gridFight/money.png"},
	{"grid-fight-front", "resources/gridFight/gridFight/front.webp"},
	{"grid-fight-star", "resources/gridFight/gridFight/star.png"},
}

// currencyWarsFrames maps the frames the archive page sets bonds, by quality,
// investment environments and strategies, by quality, in.
var currencyWarsFrames = [][2]string{
	{"grid-fight-bond-0", "resources/gridFight/gridFight/bond_bronze.png"},
	{"grid-fight-bond-1", "resources/gridFight/gridFight/bond_silver.png"},
	{"grid-fight-bond-2", "resources/gridFight/gridFight/bond_gold.png"},
	{"grid-fight-bond-3", "resources/gridFight/gridFight/bond_rainbow.png"},
	{"grid-fight-bond-4", "resources/gridFight/gridFight/bond_unique.png"},
	{"grid-fight-environment", "resources/gridFight/gridFight/environment.png"},
	{"grid-fight-strategy-1", "resources/gridFight/gridFight/strategy_silver.png"},
	{"grid-fight-strategy-2", "resources/gridFight/gridFight/strategy_gold.png"},
	{"grid-fight-strategy-3", "resources/gridFight/gridFight/strategy-icon-bg-3.b1898bcc.webp"},
}

// CurrencyWars draws 货币战争 the way StarRail-plugin's gridFight page does:
// the promotion level, this week's points and the rank, the data bank, bond
// link and expected income progress, and the latest cleared run's rank,
// mode, time, grade, team HP, economy and front and back lineup.
func CurrencyWars(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	brief, _ := result.Data["grid_fight_brief"].(map[string]any)
	if brief == nil {
		return app.Image{}, false
	}
	records, _ := result.Data["grid_fight_archive_list"].([]any)
	var latest map[string]any
	if len(records) > 0 {
		latest, _ = records[0].(map[string]any)
	}
	resources := newChallengeResources(context)
	for _, item := range currencyWarsArtwork {
		resources.local(item[0], item[1])
	}
	resources.Prefetch("mihoyo", currencyWarsImages(brief, latest, false)...)
	data := currencyWarsHeader(resources, result.Role.UID, brief)
	data["handbook"], data["trait"] = app.Text(brief["handbook_progress"]), app.Text(brief["trait_progress"])
	data["quest"] = app.Text(brief["quest_cur"]) + "/" + app.Text(brief["quest_max"])
	if latest != nil {
		record := currencyWarsRecord(resources, latest, false)
		// This page reads the mode from is_promotion, the archive page from
		// archive_type, as upstream does.
		recordBrief, _ := latest["brief"].(map[string]any)
		record["mode"] = currencyWarsMode(recordBrief["is_promotion"] == true)
		data["latest"] = record
	}
	return app.Image{Template: "currency-wars", Data: data, Resources: resources.List}, true
}

// CurrencyWarsArchive draws 货币战绩 the way StarRail-plugin's
// gridFightArchive page does for the record the command word names: the
// header, which record of how many, the record as the latest run shows with
// each character's first three pieces of equipment, its bonds, investment
// environments and strategies, and the five characters or bonds that dealt
// the most damage, each with a bar against the most.
func CurrencyWarsArchive(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	brief, _ := result.Data["grid_fight_brief"].(map[string]any)
	records, _ := result.Data["grid_fight_archive_list"].([]any)
	index := app.CurrencyWarsRecord(context.Word)
	if brief == nil || len(records) < index {
		return app.Image{}, false
	}
	archived, _ := records[index-1].(map[string]any)
	resources := newChallengeResources(context)
	for _, item := range slices.Concat(currencyWarsArtwork, currencyWarsFrames) {
		resources.local(item[0], item[1])
	}
	resources.Prefetch("mihoyo", currencyWarsImages(brief, archived, true)...)
	data := currencyWarsHeader(resources, result.Role.UID, brief)
	data["index"] = strconv.Itoa(index) + " / " + strconv.Itoa(len(records))
	record := currencyWarsRecord(resources, archived, true)
	recordBrief, _ := archived["brief"].(map[string]any)
	record["mode"] = currencyWarsMode(app.Text(recordBrief["archive_type"]) == "ArchiveTypePromotion")
	data["record"] = record

	lineup, _ := archived["lineup"].(map[string]any)
	// Damage names a character by role ID and a bond by trait ID, in one map
	// as upstream keeps them.
	entities := map[string]map[string]any{}
	for _, side := range []string{"front", "back"} {
		roles, _ := lineup[side+"_roles"].([]any)
		for index, raw := range roles {
			role, _ := raw.(map[string]any)
			entities[app.Text(role["role_id"])] = record[side].([]any)[index].(map[string]any)
		}
	}
	traits, portals, augments := []any{}, []any{}, []any{}
	traitList, _ := lineup["trait_list"].([]any)
	for _, raw := range traitList {
		item, _ := raw.(map[string]any)
		trait := map[string]any{"quality": app.Text(item["quality"]), "icon": resources.official(item["icon"]), "count": app.Text(item["trait_count"]), "name": app.Text(item["trait_name"])}
		traits = append(traits, trait)
		entities[app.Text(item["trait_id"])] = trait
	}
	portalList, _ := lineup["portal_list"].([]any)
	for _, raw := range portalList {
		item, _ := raw.(map[string]any)
		portals = append(portals, map[string]any{"icon": resources.official(item["icon"]), "name": app.Text(item["name"])})
	}
	augmentList, _ := lineup["augment_list"].([]any)
	for _, raw := range augmentList {
		item, _ := raw.(map[string]any)
		augments = append(augments, map[string]any{"quality": app.Text(item["quality"]), "icon": resources.official(item["icon"]), "name": app.Text(item["name"])})
	}
	data["traits"], data["portals"], data["augments"], data["damage"] = traits, portals, augments, currencyWarsDamage(lineup, entities)
	return app.Image{Template: "currency-wars-archive", Data: data, Resources: resources.List}, true
}

// currencyWarsHeader is the header both pages share: the promotion level,
// this week's points and the rank with its badge.
func currencyWarsHeader(resources *challengeResources, uid string, brief map[string]any) map[string]any {
	division, _ := brief["division"].(map[string]any)
	return map[string]any{"uid": uid, "season_level": app.Text(brief["season_level"]), "weekly_score": app.Text(brief["weekly_score_cur"]) + "/" + app.Text(brief["weekly_score_max"]),
		"rank_icon": resources.official(division["icon_with_bg"]), "rank_badge": currencyWarsBadge(division), "rank": app.Text(division["name_with_num"])}
}

// currencyWarsRecord is a run as both pages show it: its rank, time and
// grade, the team HP and economy, and the front and back lineup, with the
// characters' equipment on the archive page.
func currencyWarsRecord(resources *challengeResources, record map[string]any, equipment bool) map[string]any {
	brief, _ := record["brief"].(map[string]any)
	division, _ := brief["division"].(map[string]any)
	lineup, _ := record["lineup"].(map[string]any)
	// Upstream shows the grade after the 20 characters of
	// "GridFightArchiveRank".
	grade := []rune(app.Text(brief["archive_rank"]))
	out := map[string]any{"badge": currencyWarsBadge(division), "icon": resources.official(division["icon"]), "division": app.Text(division["name_with_num"]),
		"time": app.Text(brief["archive_time"]), "grade": string(grade[min(20, len(grade)):]),
		"hp": app.Text(brief["remain_hp"]), "coin": app.Text(brief["total_coin"]), "lineup_coin": app.Text(brief["lineup_coin"])}
	star := resources.local("grid-fight-star", "resources/gridFight/gridFight/star.png")
	for _, side := range []string{"front", "back"} {
		roles, _ := lineup[side+"_roles"].([]any)
		list := []any{}
		for _, raw := range roles {
			role, _ := raw.(map[string]any)
			stars := []any{}
			for range app.Int(role["star"]) {
				stars = append(stars, star)
			}
			card := map[string]any{"rarity": app.Text(role["rarity"]), "icon": resources.official(role["icon"]),
				"trial": role["is_trial"] == true, "stars": stars}
			if equipment {
				equips, _ := role["equip_list"].([]any)
				icons := []any{}
				for _, item := range equips[:min(3, len(equips))] {
					equip, _ := item.(map[string]any)
					icons = append(icons, resources.official(equip["icon"]))
				}
				card["equips"] = icons
			}
			list = append(list, card)
		}
		out[side] = list
	}
	return out
}

// currencyWarsDamage is the archive's damage chart: the five largest entries
// of the damage list, largest first, each with its character or bond, its
// damage in 万 from ten thousand, and its bar as a share of the largest.
// Upstream fails to draw the page when an entry names no one in the lineup;
// such an entry is drawn without its character here.
func currencyWarsDamage(lineup map[string]any, entities map[string]map[string]any) []any {
	type damage struct {
		value  float64
		trait  bool
		entity map[string]any
	}
	list := []damage{}
	damageList, _ := lineup["damage_list"].([]any)
	for _, raw := range damageList {
		item, _ := raw.(map[string]any)
		value, _ := strconv.ParseFloat(app.Text(item["damage"]), 64)
		list = append(list, damage{value, app.Text(item["damage_type"]) == "DamageTypeTrait", entities[app.Text(item["id"])]})
	}
	slices.SortStableFunc(list, func(a, b damage) int {
		switch {
		case a.value > b.value:
			return -1
		case a.value < b.value:
			return 1
		}
		return 0
	})
	rows := []any{}
	for _, item := range list[:min(5, len(list))] {
		percent := "0"
		if most := list[0].value; most > 0 {
			percent = app.JSFixed(item.value/most*100, 2)
		}
		value := app.JSFixed(item.value, 0)
		if item.value >= 10000 {
			value = app.JSFixed(item.value/10000, 1) + "万"
		}
		rows = append(rows, map[string]any{"trait": item.trait, "entity": item.entity, "value": value, "percent": percent})
	}
	return rows
}

// currencyWarsBadge is upstream's rank badge, one below the division level.
func currencyWarsBadge(division map[string]any) string {
	return "A" + strconv.Itoa(app.Int(division["level"])-1)
}

func currencyWarsMode(promotion bool) string {
	if promotion {
		return "快速博弈"
	}
	return "标准博弈"
}

// currencyWarsImages lists the official images a page draws, to fetch them
// together: the rank, and the run's division and characters, with the
// equipment, bonds, environments and strategies on the archive page.
func currencyWarsImages(brief, record map[string]any, archive bool) []any {
	division, _ := brief["division"].(map[string]any)
	urls := []any{division["icon_with_bg"]}
	recordBrief, _ := record["brief"].(map[string]any)
	recordDivision, _ := recordBrief["division"].(map[string]any)
	urls = append(urls, recordDivision["icon"])
	lineup, _ := record["lineup"].(map[string]any)
	lists := []string{"front_roles", "back_roles"}
	if archive {
		lists = append(lists, "trait_list", "portal_list", "augment_list")
	}
	for _, key := range lists {
		items, _ := lineup[key].([]any)
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			urls = append(urls, item["icon"])
			if equips, _ := item["equip_list"].([]any); archive {
				for _, raw := range equips[:min(3, len(equips))] {
					equip, _ := raw.(map[string]any)
					urls = append(urls, equip["icon"])
				}
			}
		}
	}
	return urls
}
