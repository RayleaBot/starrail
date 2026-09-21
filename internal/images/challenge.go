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
	class, title   string
	bg, top, floor string
	// scored records total each floor's node scores.
	scored bool
}

var (
	// Memory of Chaos keeps upstream's older name in the title.
	challengeRecord = challengeKind{class: "challenge", title: "忘却之庭", bg: "bg_1.bc4384de", top: "top_bg_1.ef4dfad4", floor: "floor_1.aa901da5"}
	storyRecord     = challengeKind{class: "story", title: "虚构叙事", bg: "bg_2.8d088acd", top: "top_bg_2.4b303ec9", floor: "floor_2.5d63517c", scored: true}
	bossRecord      = challengeKind{class: "boss", title: "末日幻影", bg: "bg_3.f75b3067", top: "top_bg_3.32f46b6a", floor: "floor_3.73c272d4", scored: true}
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

// challengeImage draws a record the way StarRail-plugin's challenge command
// does: the period with its NEW or EXPIRING mark, the stars, deepest floor and
// battles, the bosses of the period, then every floor not skipped with its
// stars or score and each node's time, team and buff.
func challengeImage(kind challengeKind) gamekit.ImageBuilder {
	return func(context gamekit.ImageContext, result gamekit.QueryResult) (gamekit.Image, bool) {
		data := result.Data
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
		if kind.class != "challenge" || begin == nil {
			begin, end = group["begin_time"], group["end_time"]
		}
		from, okFrom := challengeTime(begin)
		to, okTo := challengeTime(end)
		if !okFrom || !okTo {
			return gamekit.Image{}, false
		}

		resources := []rayleabot.RenderImageResource{}
		local := func(id, name string) string {
			resource, ok := context.ArtworkResource(id, "starrail-plugin", name)
			if !ok {
				return ""
			}
			resources = append(resources, resource)
			return id
		}
		for _, item := range challengeArtwork {
			local(item[0], item[1])
		}
		for id, name := range map[string]string{"bg": kind.bg, "top-bg": kind.top, "floor-bg": kind.floor} {
			local(id, "resources/challenge/images/"+name+".png")
		}
		official := map[string]string{}
		officialImage := func(url string) string {
			if id, seen := official[url]; seen {
				return id
			}
			id := "official-" + strconv.Itoa(len(official))
			resource, ok := context.FetchURLResource(id, "mihoyo", url)
			if ok {
				resources = append(resources, resource)
			} else {
				id = ""
			}
			official[url] = id
			return id
		}
		elements := map[string]string{}
		element := func(name string) string {
			name = strings.ToLower(name)
			if id, seen := elements[name]; seen {
				return id
			}
			elements[name] = local("element-"+name, "resources/common/element/"+name+".png")
			return elements[name]
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
					"icon": officialImage(gamekit.Text(boss["icon"])), "third": index == 2})
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
				clock := ""
				if at, ok := challengeTime(node["challenge_time"]); ok {
					clock = at.Format("2006.01.02 15:04")
				}
				_, boss := node["boss_defeated"]
				defeated, _ := node["boss_defeated"].(bool)
				avatars := []any{}
				members, _ := node["avatars"].([]any)
				for _, rawAvatar := range members {
					avatar, _ := rawAvatar.(map[string]any)
					avatars = append(avatars, map[string]any{"rarity": gamekit.Int(avatar["rarity"]), "rank": gamekit.Int(avatar["rank"]),
						"element": element(gamekit.Text(avatar["element"])), "icon": officialImage(gamekit.Text(avatar["icon"])), "level": gamekit.Text(avatar["level"])})
				}
				var buff any
				if raw, _ := node["buff"].(map[string]any); raw != nil {
					buff = map[string]any{"icon": officialImage(gamekit.Text(raw["icon"])), "name": gamekit.Text(raw["name_mi18n"]), "desc": gamekit.Text(raw["desc_mi18n"])}
				}
				nodes = append(nodes, map[string]any{"label": "节点" + strconv.Itoa(index+1), "third": index == 2, "time": clock,
					"boss": boss, "defeated": defeated, "score": nodeScore, "avatars": avatars, "buff": buff})
			}
			_, hasRound := floor["round_num"]
			total := ""
			if scored {
				total = strconv.Itoa(score)
			}
			floors = append(floors, map[string]any{"name": gamekit.Text(floor["name"]), "has_round": hasRound, "round": gamekit.Text(floor["round_num"]),
				"stars": stars, "score": total, "nodes": nodes})
		}

		period := "本期"
		if !current {
			period = "上期"
		}
		return gamekit.Image{Template: "challenge", Data: map[string]any{
			"uid": result.Role.UID, "kind": kind.class, "title": kind.title, "period": period, "badge": badge,
			"begin": from.Format("2006.01.02"), "end": to.Format("2006.01.02"),
			"star_num": gamekit.Int(data["star_num"]), "extra_star_num": gamekit.Int(data["extra_star_num"]),
			"max_floor": gamekit.Text(data["max_floor"]), "battle_num": gamekit.Text(data["battle_num"]),
			"bosses": bosses, "floors": floors,
		}, Resources: resources}, true
	}
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
