package images

import (
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// challengeKind is one of the three records StarRail-plugin draws with
// challenge/index.html, with the title and art that record uses.
type challengeKind struct {
	operation, class, title string
	// art names the record's background, summary and floor images.
	art [3]string
	// scored records total each floor's node scores.
	scored bool
}

var (
	// Memory of Chaos keeps upstream's older name in the title.
	memoryOfChaos     = challengeKind{"starrail.challenge", "chaos", "忘却之庭", [3]string{"bg_1.bc4384de", "top_bg_1.ef4dfad4", "floor_1.aa901da5"}, false}
	pureFiction       = challengeKind{"starrail.story", "fiction", "虚构叙事", [3]string{"bg_2.8d088acd", "top_bg_2.4b303ec9", "floor_2.5d63517c"}, true}
	apocalypticShadow = challengeKind{"starrail.boss", "shadow", "末日幻影", [3]string{"bg_3.f75b3067", "top_bg_3.32f46b6a", "floor_3.73c272d4"}, true}
	// challengeKinds is upstream's rotation order: Apocalyptic Shadow,
	// Pure Fiction, then Memory of Chaos.
	challengeKinds = []challengeKind{apocalypticShadow, pureFiction, memoryOfChaos}
)

// challengeArtwork maps the fixed images of the challenge pages to their
// StarRail-plugin paths.
var challengeArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"tttgbnumber", "resources/common/font/tttgbnumber.ttf"},
	{"rogue-star", "resources/common/cont/rogue-star.png"},
	{"golden-clock", "resources/common/cont/golden-clock.png"},
	{"red-clock", "resources/common/cont/red-clock.png"},
	{"abyss-star", "resources/challenge/images/abyss_star.f89de036.png"},
	{"character-r4", "resources/challenge/images/character_r_4.24f329b7.png"},
	{"character-r5", "resources/challenge/images/character_r_5.99d42eb7.png"},
	{"boss-node3-bg", "resources/challenge/images/boss_node3_bg.923d912f.png"},
	{"node3-time-bg", "resources/challenge/images/memory_node3_time_bg.ccb847b8.png"},
	{"node3-crown", "resources/challenge/images/node3_crown.png"},
}

// challengeQueries routes 深渊, which starts from Memory of Chaos, and
// 最新深渊, which runs the record upstream's schedule says opened last.
func challengeQueries() map[string]func(time.Time) string {
	return map[string]func(time.Time) string{
		"starrail.abyss":        func(time.Time) string { return memoryOfChaos.operation },
		"starrail.abyss_latest": func(now time.Time) string { return latestChallenge(now).operation },
	}
}

// latestChallenge follows upstream's getCurrentChallengeType: the three
// records open in turn every 14 days, counted from Apocalyptic Shadow on
// 2024-06-24 04:00.
func latestChallenge(now time.Time) challengeKind {
	first := time.Date(2024, 6, 24, 4, 0, 0, 0, chinaTime)
	periods := int(now.Sub(first) / (14 * 24 * time.Hour))
	return challengeKinds[(periods%3+3)%3]
}

// challengeImage draws a record the way StarRail-plugin's challenge command
// does.
func challengeImage(kind challengeKind) gamekit.ImageBuilder {
	return func(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
		resources := &challengeResources{context: context}
		record, ok := challengeRecordData(context, kind, result.Data, resources)
		if !ok {
			return gamekit.Image{}, false
		}
		record["uid"], record["period"] = result.Role.UID, challengePeriod(context)
		return gamekit.Image{Template: "challenge", Data: record, Resources: resources.list}, true
	}
}

// latestChallengeImage draws 最新深渊 as the record that ran.
func latestChallengeImage(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	for _, kind := range challengeKinds {
		if kind.operation == result.Operation {
			return challengeImage(kind)(context, result)
		}
	}
	return gamekit.Image{}, false
}

// allChallengesImage draws 深渊 the way upstream's index_all does: Memory of
// Chaos, Apocalyptic Shadow and Pure Fiction of the same period side by side.
// Upstream answers nothing when one of them fails; this reply falls back to
// Memory of Chaos in text.
func allChallengesImage(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
	if context.Query == nil {
		return gamekit.Image{}, false
	}
	resources := &challengeResources{context: context}
	resources.local("all-bg", "resources/challenge/images/pc_bg.46428ec0.png")
	records := []any{}
	for _, kind := range []challengeKind{memoryOfChaos, apocalypticShadow, pureFiction} {
		data := result.Data
		if kind != memoryOfChaos {
			other, err := context.Query(kind.operation, context.Input)
			if err != nil {
				return gamekit.Image{}, false
			}
			data = other.Data
		}
		record, ok := challengeRecordData(context, kind, data, resources)
		if !ok {
			return gamekit.Image{}, false
		}
		// Only the Apocalyptic Shadow column shows the period's bosses.
		if kind != apocalypticShadow {
			record["bosses"] = []any{}
		}
		records = append(records, record)
	}
	return gamekit.Image{Template: "challenge-all", Data: map[string]any{"uid": result.Role.UID, "period": challengePeriod(context), "records": records}, Resources: resources.list}, true
}

func challengePeriod(context gamekit.ImageContext) string {
	if gamekit.Int(context.Input["schedule_type"]) == 2 {
		return "上期"
	}
	return "本期"
}

// challengeRecordData shapes one record for the template: the period with its
// NEW or EXPIRING mark, the stars, deepest floor and battles, the bosses of
// the period, then every floor not skipped with its stars or score and each
// node's time, team and buff.
func challengeRecordData(context gamekit.ImageContext, kind challengeKind, data map[string]any, resources *challengeResources) (map[string]any, bool) {
	current := gamekit.Int(context.Input["schedule_type"]) != 2
	// Pure Fiction and Apocalyptic Shadow list both periods; upstream keeps
	// the one the command asked for.
	groups, _ := data["groups"].([]any)
	var group map[string]any
	if len(groups) > 0 {
		group, _ = groups[0].(map[string]any)
	}
	if len(groups) > 1 {
		want := "New"
		if !current {
			want = "End"
		}
		for _, raw := range groups {
			if item, _ := raw.(map[string]any); gamekit.Text(item["status"]) == want {
				group = item
				break
			}
		}
	}
	begin, end := data["begin_time"], data["end_time"]
	if kind != memoryOfChaos || begin == nil {
		begin, end = group["begin_time"], group["end_time"]
	}
	from, okFrom := challengeTime(begin)
	to, okTo := challengeTime(end)
	if !okFrom || !okTo {
		return nil, false
	}

	for _, item := range challengeArtwork {
		resources.local(item[0], item[1])
	}
	for index, prefix := range []string{"bg-", "top-bg-", "floor-bg-"} {
		resources.local(prefix+kind.class, "resources/challenge/images/"+kind.art[index]+".png")
	}

	// Upstream shows the mark by the refresh schedule, whichever period
	// was asked; a past period is neither new nor expiring.
	badge := ""
	if now := context.Now.In(chinaTime); current && now.Before(from.AddDate(0, 0, 14)) {
		badge = "new"
	} else if current && now.After(to.AddDate(0, 0, -14)) {
		badge = "expiring"
	}
	bosses := []any{}
	if upper, _ := group["upper_boss"].(map[string]any); upper != nil {
		for index, key := range []string{"upper_boss", "lower_boss", "tierce_boss"} {
			boss, _ := group[key].(map[string]any)
			if boss == nil {
				continue
			}
			bosses = append(bosses, map[string]any{"label": "节点" + strconv.Itoa(index+1), "name": gamekit.Text(boss["name_mi18n"]),
				"icon": resources.official(boss["icon"]), "third": index == 2})
		}
	}

	floors := []any{}
	list, _ := data["all_floor_detail"].([]any)
	for _, raw := range list {
		floor, _ := raw.(map[string]any)
		if fast, _ := floor["is_fast"].(bool); fast || floor == nil {
			continue
		}
		tierce, _ := floor["is_tierce"].(bool)
		stars := []any{}
		normal, extra := gamekit.Int(floor["star_num"]), gamekit.Int(floor["extra_star_num"])
		if tierce {
			normal -= extra
		}
		for range max(normal, 0) {
			stars = append(stars, "rogue-star")
		}
		if tierce && extra == 1 {
			stars = append(stars, "abyss-star")
		}
		nodes := []any{}
		score := 0
		scored := false
		for index, key := range []string{"node_1", "node_2", "node_3"} {
			node, _ := floor[key].(map[string]any)
			if node == nil && index > 0 {
				continue
			}
			nodeScore := ""
			if value := node["score"]; value != nil {
				nodeScore = gamekit.Text(value)
				// Upstream totals the nodes when the first has a score, and
				// counts the third only on floors that have one.
				if kind.scored && (index == 0 || scored && (index == 1 || tierce)) {
					score += gamekit.Int(value)
					scored = true
				}
			}
			_, boss := node["boss_defeated"]
			defeated, _ := node["boss_defeated"].(bool)
			nodes = append(nodes, map[string]any{"label": "节点" + strconv.Itoa(index+1), "third": index == 2, "time": challengeClock(node),
				"boss": boss, "defeated": defeated, "score": nodeScore, "avatars": resources.team(node), "buff": resources.buff(node)})
		}
		_, hasRound := floor["round_num"]
		total := ""
		if scored {
			total = strconv.Itoa(score)
		}
		floors = append(floors, map[string]any{"name": gamekit.Text(floor["name"]), "has_round": hasRound, "round": gamekit.Text(floor["round_num"]),
			"stars": stars, "score": total, "nodes": nodes})
	}

	return map[string]any{
		"kind": kind.class, "title": kind.title, "badge": badge,
		"begin": from.Format("2006.01.02"), "end": to.Format("2006.01.02"),
		"star_num": gamekit.Int(data["star_num"]), "extra_star_num": gamekit.Int(data["extra_star_num"]),
		"max_floor": gamekit.Text(data["max_floor"]), "battle_num": gamekit.Text(data["battle_num"]),
		"bosses": bosses, "floors": floors,
	}, true
}

// challengeTime reads the official {year, month, day, hour, minute} time.
func challengeTime(value any) (time.Time, bool) {
	fields, _ := value.(map[string]any)
	year := gamekit.Int(fields["year"])
	if year == 0 {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(gamekit.Int(fields["month"])), gamekit.Int(fields["day"]), gamekit.Int(fields["hour"]), gamekit.Int(fields["minute"]), 0, 0, chinaTime), true
}

// challengeClock formats when a node or record was cleared; quick clears
// carry no time.
func challengeClock(record map[string]any) string {
	if at, ok := challengeTime(record["challenge_time"]); ok {
		return at.Format("2006.01.02 15:04")
	}
	return ""
}

// challengeResources collects a challenge image's resources, each file once:
// StarRail-plugin art, element icons, and the official images the response
// links to, cached on demand.
type challengeResources struct {
	context gamekit.ImageContext
	list    []rayleabot.RenderImageResource
	seen    map[string]string
}

func (r *challengeResources) once(key string, add func() (rayleabot.RenderImageResource, bool)) string {
	if id, ok := r.seen[key]; ok {
		return id
	}
	if r.seen == nil {
		r.seen = map[string]string{}
	}
	id := ""
	if resource, ok := add(); ok {
		r.list = append(r.list, resource)
		id = resource.ID
	}
	r.seen[key] = id
	return id
}

func (r *challengeResources) local(id, name string) string {
	return r.once(name, func() (rayleabot.RenderImageResource, bool) {
		return r.context.ArtworkResource(id, "starrail-plugin", name)
	})
}

func (r *challengeResources) official(url any) string {
	id := "official-" + strconv.Itoa(len(r.seen))
	return r.once(gamekit.Text(url), func() (rayleabot.RenderImageResource, bool) {
		return r.context.FetchURLResource(id, "mihoyo", gamekit.Text(url))
	})
}

// team lists a node's or record's characters as upstream's avatar cards show
// them; the element files are lower case.
func (r *challengeResources) team(record map[string]any) []any {
	list := []any{}
	avatars, _ := record["avatars"].([]any)
	for _, raw := range avatars {
		avatar, _ := raw.(map[string]any)
		element := strings.ToLower(gamekit.Text(avatar["element"]))
		list = append(list, map[string]any{"rarity": gamekit.Int(avatar["rarity"]), "rank": gamekit.Int(avatar["rank"]), "level": gamekit.Text(avatar["level"]),
			"element": r.local("element-"+element, "resources/common/element/"+element+".png"), "icon": r.official(avatar["icon"])})
	}
	return list
}

// buff is a node's or record's buff with its icon, or nil.
func (r *challengeResources) buff(record map[string]any) any {
	buff, _ := record["buff"].(map[string]any)
	if buff == nil {
		return nil
	}
	return map[string]any{"icon": r.official(buff["icon"]), "name": gamekit.Text(buff["name_mi18n"]), "desc": gamekit.Text(buff["desc_mi18n"])}
}
