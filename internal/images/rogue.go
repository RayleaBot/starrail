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
	runs := []any{}
	list, _ := record["records"].([]any)
	for _, raw := range list {
		run, _ := raw.(map[string]any)
		entry := resources.rogueRun(run)
		entry["name"], entry["difficulty"], entry["score"] = gamekit.Text(run["name"]), gamekit.Text(run["difficulty"]), gamekit.Text(run["score"])
		runs = append(runs, entry)
	}
	role, _ := result.Data["role"].(map[string]any)
	basic, _ := result.Data["basic_info"].(map[string]any)
	summary, _ := record["basic"].(map[string]any)
	return gamekit.Image{Template: "rogue", Data: map[string]any{
		"uid": result.Role.UID, "nickname": gamekit.Text(role["nickname"]), "level": gamekit.Text(role["level"]),
		"skill_points": gamekit.Text(basic["unlocked_skill_points"]), "miracles": gamekit.Text(basic["unlocked_miracle_num"]), "buffs": gamekit.Text(basic["unlocked_buff_num"]),
		"begin": rogueDate(summary["schedule_begin"]), "end": rogueDate(summary["schedule_end"]), "finished": gamekit.Text(summary["finish_cnt"]), "runs": runs,
	}, Resources: resources.List}, true
}

// pathIcon is a Path icon, named upstream by the two characters inside 「」.
func (r *challengeResources) pathIcon(name string) string {
	runes := []rune(name)
	if len(runes) < 3 {
		return ""
	}
	return r.local("path-"+strconv.Itoa(len(r.List)), "resources/common/rogue/"+string(runes[1:3])+".webp")
}

// rogueRun is what the Simulated Universe pages show of any run: its clear
// time, final lineup, Path resonances, Blessings by Path and Curios.
func (r *challengeResources) rogueRun(run map[string]any) map[string]any {
	lineup := []any{}
	avatars, _ := run["final_lineup"].([]any)
	for _, rawAvatar := range avatars {
		avatar, _ := rawAvatar.(map[string]any)
		element := gamekit.Text(avatar["element"])
		lineup = append(lineup, map[string]any{"rarity": gamekit.Int(avatar["rarity"]), "rank": gamekit.Int(avatar["rank"]), "level": gamekit.Text(avatar["level"]),
			"icon": r.official(avatar["icon"]), "element": r.local("element-"+element, "resources/common/element/"+element+".png")})
	}
	bases := []any{}
	baseList, _ := run["base_type_list"].([]any)
	for _, rawBase := range baseList {
		base, _ := rawBase.(map[string]any)
		bases = append(bases, map[string]any{"icon": r.pathIcon(gamekit.Text(base["name"])), "count": gamekit.Text(base["cnt"])})
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
		buffs = append(buffs, map[string]any{"icon": r.pathIcon(name), "name": name, "items": items})
	}
	miracles := []any{}
	miracleList, _ := run["miracles"].([]any)
	for _, rawMiracle := range miracleList {
		miracle, _ := rawMiracle.(map[string]any)
		miracles = append(miracles, r.official(miracle["icon"]))
	}
	finish, _ := run["finish_time"].(map[string]any)
	time := gamekit.Text(finish["year"]) + "." + gamekit.Text(finish["month"]) + "." + gamekit.Text(finish["day"]) + " " +
		gamekit.Text(finish["hour"]) + ":" + gamekit.Text(finish["minute"]) + ":" + gamekit.Text(finish["second"])
	return map[string]any{"time": time, "lineup": lineup, "bases": bases, "buffs": buffs, "miracles": miracles}
}

// rogueDate writes a period date the way the pages do, unpadded.
func rogueDate(value any) string {
	fields, _ := value.(map[string]any)
	return gamekit.Text(fields["year"]) + "." + gamekit.Text(fields["month"]) + "." + gamekit.Text(fields["day"])
}

// swarmPaths are the radar's Paths in the order of the official destiny
// list, which upstream reads by index.
var swarmPaths = [][2]string{
	{"preservation", "存护"}, {"remembrance", "记忆"}, {"nihility", "虚无"}, {"abundance", "丰饶"},
	{"hunt", "巡猎"}, {"destruction", "毁灭"}, {"elation", "欢愉"},
}

// swarmArtwork maps the images the swarm page names to StarRail-plugin's
// paths.
var swarmArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"tttgbnumber", "resources/common/font/tttgbnumber.ttf"},
	{"rogue-bg", "resources/rogue/bg.png"},
	{"swarm-bg", "resources/rogue/locust/bg_fanyu.png"},
	{"swarm-radar", "resources/rogue/locust/radar.png"},
	{"swarm-icon", "resources/rogue/locust/locust.png"},
}

// Swarm draws Swarm Disaster the way StarRail-plugin's rogue_locust page
// does: the Pathstrider, Curio and Occurrence counts, each Path's level on
// the radar, then the last three runs with difficulty, time, lineup, Path
// resonances, the disorder countdown or disturbance level, weakening effects,
// stored characters, Blessings, Curios and the domain types passed.
func Swarm(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	basic, _ := result.Data["basic"].(map[string]any)
	detail, _ := result.Data["detail"].(map[string]any)
	if basic == nil || detail == nil {
		return gamekit.Image{}, false
	}
	resources := newChallengeResources(context)
	for _, item := range swarmArtwork {
		resources.local(item[0], item[1])
	}
	destinies, _ := basic["destiny"].([]any)
	radar := map[string]any{}
	for index, path := range swarmPaths {
		level := ""
		if index < len(destinies) {
			entry, _ := destinies[index].(map[string]any)
			level = gamekit.Text(entry["level"])
		}
		radar[path[0]] = map[string]any{"icon": resources.pathIcon("「" + path[1] + "」"), "level": level}
	}
	runs := []any{}
	list, _ := detail["records"].([]any)
	for _, raw := range list {
		run, _ := raw.(map[string]any)
		entry := resources.rogueRun(run)
		fury, _ := run["fury"].(map[string]any)
		weak := []any{}
		weakList, _ := run["worm_weak"].([]any)
		for _, item := range weakList {
			weak = append(weak, gamekit.Text(item))
		}
		cached := []any{}
		cachedList, _ := run["cached_avatars"].([]any)
		for _, rawAvatar := range cachedList {
			avatar, _ := rawAvatar.(map[string]any)
			cached = append(cached, map[string]any{"icon": resources.official(avatar["icon"]), "level": gamekit.Text(avatar["level"])})
		}
		passed := []any{}
		blockList, _ := run["blocks"].([]any)
		for _, rawBlock := range blockList {
			item, _ := rawBlock.(map[string]any)
			name := gamekit.Text(item["name"])
			passed = append(passed, map[string]any{"name": name, "num": gamekit.Text(item["num"]), "icon": resources.local("block-"+strconv.Itoa(len(resources.List)), "resources/rogue/locust/"+name+".png")})
		}
		entry["difficulty"], entry["countdown"], entry["fury"] = gamekit.Text(run["difficulty"]), gamekit.Int(fury["type"]) == 0, gamekit.Text(fury["point"])
		entry["weak"], entry["cached"], entry["blocks"] = weak, cached, passed
		runs = append(runs, entry)
	}
	role, _ := result.Data["role"].(map[string]any)
	counts, _ := basic["cnt"].(map[string]any)
	return gamekit.Image{Template: "swarm", Data: map[string]any{
		"uid": result.Role.UID, "nickname": gamekit.Text(role["nickname"]), "level": gamekit.Text(role["level"]),
		"narrow": gamekit.Text(counts["narrow"]), "miracle": gamekit.Text(counts["miracle"]), "event": gamekit.Text(counts["event"]), "radar": radar, "runs": runs,
	}, Resources: resources.List}, true
}
