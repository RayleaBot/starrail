package images

import (
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/app"
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
func challengeImage(kind challengeKind) app.ImageBuilder {
	return func(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
		resources := newChallengeResources(context)
		record, ok := challengeRecordData(context, kind, result.Data, resources)
		if !ok {
			return app.Image{}, false
		}
		record["uid"], record["period"] = result.Role.UID, challengePeriod(context)
		return app.Image{Template: "challenge", Data: record, Resources: resources.List}, true
	}
}

// latestChallengeImage draws 最新深渊 as the record that ran.
func latestChallengeImage(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	for _, kind := range challengeKinds {
		if kind.operation == result.Operation {
			return challengeImage(kind)(context, result)
		}
	}
	return app.Image{}, false
}

// allChallengesImage draws 深渊 the way upstream's index_all does: Memory of
// Chaos, Apocalyptic Shadow and Pure Fiction of the same period side by side.
// Upstream answers nothing when one of them fails; this reply falls back to
// Memory of Chaos in text.
func allChallengesImage(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	if context.Query == nil {
		return app.Image{}, false
	}
	resources := newChallengeResources(context)
	resources.local("all-bg", "resources/challenge/images/pc_bg.46428ec0.png")
	records := []any{}
	for _, kind := range []challengeKind{memoryOfChaos, apocalypticShadow, pureFiction} {
		data := result.Data
		if kind != memoryOfChaos {
			other, err := context.Query(kind.operation, context.Input)
			if err != nil {
				return app.Image{}, false
			}
			data = other.Data
		}
		record, ok := challengeRecordData(context, kind, data, resources)
		if !ok {
			return app.Image{}, false
		}
		// Only the Apocalyptic Shadow column shows the period's bosses.
		if kind != apocalypticShadow {
			record["bosses"] = []any{}
		}
		records = append(records, record)
	}
	return app.Image{Template: "challenge-all", Data: map[string]any{"uid": result.Role.UID, "period": challengePeriod(context), "records": records}, Resources: resources.List}, true
}

func challengePeriod(context app.ImageContext) string {
	if app.Int(context.Input["schedule_type"]) == 2 {
		return "上期"
	}
	return "本期"
}

// challengeRecordData shapes one record for the template: the period with its
// NEW or EXPIRING mark, the stars, deepest floor and battles, the bosses of
// the period, then every floor not skipped with its stars or score and each
// node's time, team and buff.
func challengeRecordData(context app.ImageContext, kind challengeKind, data map[string]any, resources *challengeResources) (map[string]any, bool) {
	current := app.Int(context.Input["schedule_type"]) != 2
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
			if item, _ := raw.(map[string]any); app.Text(item["status"]) == want {
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
			// Upstream draws the second node's box whenever the first has a
			// boss, and the third only when there is one.
			if boss == nil && index == 2 {
				continue
			}
			bosses = append(bosses, map[string]any{"label": "节点" + strconv.Itoa(index+1), "name": app.Text(boss["name_mi18n"]),
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
		normal, extra := app.Int(floor["star_num"]), app.Int(floor["extra_star_num"])
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
			// Upstream shows a node's score line whenever the record has the
			// key, empty when it is null.
			nodeScore := ""
			value, hasScore := node["score"]
			if value != nil {
				nodeScore = app.Text(value)
				// Upstream totals the nodes when the first has a score, and
				// counts the third only on floors that have one.
				if kind.scored && (index == 0 || scored && (index == 1 || tierce)) {
					score += app.Int(value)
					scored = true
				}
			}
			_, boss := node["boss_defeated"]
			defeated, _ := node["boss_defeated"].(bool)
			nodes = append(nodes, map[string]any{"label": "节点" + strconv.Itoa(index+1), "third": index == 2, "time": challengeClock(node),
				"boss": boss, "defeated": defeated, "score": nodeScore, "has_score": hasScore, "avatars": resources.team(node), "buff": resources.buff(node)})
		}
		_, hasRound := floor["round_num"]
		total := ""
		if scored {
			total = strconv.Itoa(score)
		}
		floors = append(floors, map[string]any{"name": app.Text(floor["name"]), "has_round": hasRound, "round": app.Text(floor["round_num"]),
			"stars": stars, "score": total, "nodes": nodes})
	}

	return map[string]any{
		"kind": kind.class, "title": kind.title, "badge": badge,
		"begin": from.Format("2006.01.02"), "end": to.Format("2006.01.02"),
		"star_num": app.Int(data["star_num"]), "extra_star_num": app.Int(data["extra_star_num"]),
		"max_floor": app.Text(data["max_floor"]), "battle_num": app.Text(data["battle_num"]),
		"bosses": bosses, "floors": floors,
	}, true
}

// challengeTime reads the official {year, month, day, hour, minute} time.
func challengeTime(value any) (time.Time, bool) {
	fields, _ := value.(map[string]any)
	year := app.Int(fields["year"])
	if year == 0 {
		return time.Time{}, false
	}
	return time.Date(year, time.Month(app.Int(fields["month"])), app.Int(fields["day"]), app.Int(fields["hour"]), app.Int(fields["minute"]), 0, 0, chinaTime), true
}

// challengeClock formats when a node or record was cleared; quick clears
// carry no time.
func challengeClock(record map[string]any) string {
	if at, ok := challengeTime(record["challenge_time"]); ok {
		return at.Format("2006.01.02 15:04")
	}
	return ""
}

// challengeResources collects a challenge image's resources: StarRail-plugin
// art, element icons, and the official images the response links to, cached
// on demand.
type challengeResources struct{ app.ImageResources }

func newChallengeResources(context app.ImageContext) *challengeResources {
	return &challengeResources{app.ImageResources{Context: context}}
}

func (r *challengeResources) local(id, name string) string {
	return r.Artwork(id, "starrail-plugin", name)
}

func (r *challengeResources) official(url any) string { return r.URL("mihoyo", url) }

// character sets a character card's picture on the pages that draw the icon
// the official answer links to, as StarRail-plugin's pages load it. When the
// icon cannot be loaded (the answer names none, or its download failed and
// waits out the store's pause) the page would show an empty card; the face
// miao's meta-sr keeps for the character's ID is drawn instead, the picture
// 面板列表 draws, marked face so the page fits it into the official icon's
// frame, as most of miao's faces are square.
func (r *challengeResources) character(card map[string]any, id, icon any) map[string]any {
	card["icon"] = r.official(icon)
	if card["icon"] != "" || r.Context.Game.Calc == nil {
		return card
	}
	for _, record := range r.Context.Game.Calc.Metadata().Characters {
		if record.ID == app.Text(id) {
			card["icon"] = r.Artwork("face-"+record.ID, "miao-plugin", "resources/meta-sr/character/"+record.Name+"/imgs/face.webp")
			card["face"] = card["icon"] != ""
			break
		}
	}
	return card
}

// team lists a node's or record's characters as upstream's avatar cards show
// them; the element files are lower case.
func (r *challengeResources) team(record map[string]any) []any {
	list := []any{}
	avatars, _ := record["avatars"].([]any)
	for _, raw := range avatars {
		avatar, _ := raw.(map[string]any)
		element := strings.ToLower(app.Text(avatar["element"]))
		list = append(list, r.character(map[string]any{"rarity": app.Int(avatar["rarity"]), "rank": app.Int(avatar["rank"]), "level": app.Text(avatar["level"]),
			"element": r.local("element-"+element, "resources/common/element/"+element+".png")}, avatar["id"], avatar["icon"]))
	}
	return list
}

// buff is a node's or record's buff with its icon, or nil.
func (r *challengeResources) buff(record map[string]any) any {
	buff, _ := record["buff"].(map[string]any)
	if buff == nil {
		return nil
	}
	return map[string]any{"icon": r.official(buff["icon"]), "name": app.Text(buff["name_mi18n"]), "desc": app.Text(buff["desc_mi18n"])}
}
