package images

import (
	"strconv"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// rogueArtwork maps the images the rogue page names to StarRail-plugin's
// paths.
var rogueArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"tttgbnumber", "resources/common/font/tttgbnumber.ttf"},
	{"rogue-ball", "resources/rogue/ball.png"},
	{"rogue-bg", "resources/rogue/bg.png"},
	{"rogue-oblivious", "resources/rogue/oblivious_bg@2x.7b5d7b38.png"},
}

// Rogue draws the Simulated Universe the way StarRail-plugin's rogue page
// does: the unlocked skill points, Curios and Blessings, the period and its
// clears, then every run with its world, difficulty, score, time, lineup,
// Path resonances, Blessings by Path and Curios.
func Rogue(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	// The kit keeps the asked period only; upstream reads both as last_record.
	record, _ := result.Data["current_record"].(map[string]any)
	if record == nil {
		record, _ = result.Data["last_record"].(map[string]any)
	}
	if record == nil {
		return gamekit.Image{}, false
	}
	resources := newChallengeResources(context)
	for _, item := range rogueArtwork {
		resources.local(item[0], item[1])
	}
	// Upstream names a Path icon by the two characters inside 「」.
	paths := map[string]string{}
	path := func(name string) string {
		runes := []rune(name)
		if len(runes) < 3 {
			return ""
		}
		short := string(runes[1:3])
		if _, seen := paths[short]; !seen {
			paths[short] = resources.local("path-"+strconv.Itoa(len(paths)), "resources/common/rogue/"+short+".webp")
		}
		return paths[short]
	}
	element := func(name string) string {
		return resources.local("element-"+name, "resources/common/element/"+name+".png")
	}
	clock := func(value any) string {
		fields, _ := value.(map[string]any)
		return gamekit.Text(fields["year"]) + "." + gamekit.Text(fields["month"]) + "." + gamekit.Text(fields["day"]) + " " +
			gamekit.Text(fields["hour"]) + ":" + gamekit.Text(fields["minute"]) + ":" + gamekit.Text(fields["second"])
	}
	date := func(value any) string {
		fields, _ := value.(map[string]any)
		return gamekit.Text(fields["year"]) + "." + gamekit.Text(fields["month"]) + "." + gamekit.Text(fields["day"])
	}
	runs := []any{}
	list, _ := record["records"].([]any)
	for _, raw := range list {
		run, _ := raw.(map[string]any)
		lineup := []any{}
		avatars, _ := run["final_lineup"].([]any)
		for _, rawAvatar := range avatars {
			avatar, _ := rawAvatar.(map[string]any)
			lineup = append(lineup, map[string]any{"rarity": gamekit.Int(avatar["rarity"]), "rank": gamekit.Int(avatar["rank"]), "level": gamekit.Text(avatar["level"]),
				"icon": resources.official(avatar["icon"]), "element": element(gamekit.Text(avatar["element"]))})
		}
		bases := []any{}
		baseList, _ := run["base_type_list"].([]any)
		for _, rawBase := range baseList {
			base, _ := rawBase.(map[string]any)
			bases = append(bases, map[string]any{"icon": path(gamekit.Text(base["name"])), "count": gamekit.Text(base["cnt"])})
		}
		buffs := []any{}
		buffList, _ := run["buffs"].([]any)
		for _, rawBuff := range buffList {
			buff, _ := rawBuff.(map[string]any)
			baseType, _ := buff["base_type"].(map[string]any)
			items := []any{}
			entries, _ := buff["items"].([]any)
			for _, rawItem := range entries {
				item, _ := rawItem.(map[string]any)
				evolved, _ := item["is_evoluted"].(bool)
				items = append(items, map[string]any{"name": gamekit.Text(item["name"]), "rank": gamekit.Text(item["rank"]), "evolved": evolved})
			}
			name := gamekit.Text(baseType["name"])
			buffs = append(buffs, map[string]any{"icon": path(name), "name": name, "items": items})
		}
		miracles := []any{}
		miracleList, _ := run["miracles"].([]any)
		for _, rawMiracle := range miracleList {
			miracle, _ := rawMiracle.(map[string]any)
			miracles = append(miracles, resources.official(miracle["icon"]))
		}
		runs = append(runs, map[string]any{"name": gamekit.Text(run["name"]), "difficulty": gamekit.Text(run["difficulty"]), "score": gamekit.Text(run["score"]),
			"time": clock(run["finish_time"]), "lineup": lineup, "bases": bases, "buffs": buffs, "miracles": miracles})
	}
	role, _ := result.Data["role"].(map[string]any)
	basic, _ := result.Data["basic_info"].(map[string]any)
	summary, _ := record["basic"].(map[string]any)
	return gamekit.Image{Template: "rogue", Data: map[string]any{
		"uid": result.Role.UID, "nickname": gamekit.Text(role["nickname"]), "level": gamekit.Text(role["level"]),
		"skill_points": gamekit.Text(basic["unlocked_skill_points"]), "miracles": gamekit.Text(basic["unlocked_miracle_num"]), "buffs": gamekit.Text(basic["unlocked_buff_num"]),
		"begin": date(summary["schedule_begin"]), "end": date(summary["schedule_end"]), "finished": gamekit.Text(summary["finish_cnt"]), "runs": runs,
	}, Resources: resources.List}, true
}
