package images

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

// rankArtwork maps the images named in the converted stylesheets (miao's
// common and character/rank-profile-list) to their paths; element
// backgrounds are added for the page's element.
var rankArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"character-imgs-crown-sr", "resources/character/imgs/crown-sr.webp"},
	{"character-imgs-mark-icon", "resources/character/imgs/mark-icon.png"},
	{"character-imgs-mark-icon2", "resources/character/imgs/mark-icon2.png"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-artifact-icon", "resources/common/item/artifact-icon.webp"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
}

// rankTalents are the talents miao's rank list shows by path: memosprite
// skills for Remembrance, the Elation skill for Elation.
var rankTalents = map[string][]string{"记忆": {"a", "e", "t", "q", "me", "mt"}, "欢愉": {"a", "e", "t", "q", "xe"}}

// Rank draws a group ranking the way miao's character/rank-profile-list
// does in its Star Rail mode: the title and ranking notes, then a row per
// entry with its place (or, listing every character's best, the character),
// the member's avatar, Eidolon, name and UID, the talents, light cone, relic
// sets with their score and grade, and the default detail's damage. miao
// widens the page for Remembrance and Elation characters, whose rows show
// more talents.
func Rank(context app.ImageContext, rank app.RankImage) (app.Image, bool) {
	if context.Game.Calc == nil || len(rank.Entries) == 0 {
		return app.Image{}, false
	}
	metadata := context.Game.Calc.Metadata()
	records, weapons := map[string]reference.Character{}, map[string]reference.Weapon{}
	for _, record := range metadata.Characters {
		records[record.ID] = record
	}
	for _, weapon := range metadata.Weapons {
		weapons[weapon.ID] = weapon
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range rankArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	miao := func(name string) string {
		return resources.Artwork("miao-"+strconv.Itoa(len(resources.List)), "miao-plugin", name)
	}
	elem := "hydro"
	if record, ok := records[rank.Character.ID]; ok && elementClasses[record.Element] != "" {
		elem = elementClasses[record.Element]
	}
	resources.Artwork("common-bg-bg-"+elem, "miao-plugin", "resources/common/bg/bg-"+elem+".webp")
	rows, wide := []any{}, false
	for index, entry := range rank.Entries {
		record, panel := records[entry.CharacterID], *entry.Panel
		base := "resources/meta-sr/character/" + record.Name + "/imgs/"
		star := app.Int(record.Data["star"])
		if star != 4 {
			star = 5
		}
		name := record.Name
		// miao's sName: the abbreviation for names of four or more characters.
		if utf8.RuneCountInString(name) >= 4 {
			name = abbreviation(context.Catalog, record)
		}
		face := miao(base + "face.webp")
		// Upstream's rows class the talents by the character's element name,
		// which no Star Rail style matches.
		row := map[string]any{"place": index + 1, "uid": entry.UID, "star": star, "face": face, "cons": panel.Rank, "name": name, "elem": record.Element}
		if entry.Avatar != "" {
			row["avatar"] = resources.Remote("avatar-"+strconv.Itoa(index), entry.Avatar)
		}
		keys := rankTalents[record.WeaponType]
		if keys == nil {
			keys = []string{"a", "e", "t", "q"}
		}
		wide = wide || len(keys) > 4
		levels := app.PanelTalents("starrail", panel, record)
		talents := []any{}
		for _, key := range keys {
			level := levels[key]
			talents = append(talents, map[string]any{"level": level.Level, "plus": level.Level > level.Original, "crown": level.Original >= 10, "icon": miao(base + "talent-" + key + ".webp")})
		}
		row["talents"] = talents
		if weapon := panel.Weapon; weapon != nil {
			cone := weapons[weapon.ID]
			label := weapon.Name
			if catalogEntry, ok := context.Catalog.Get(weapon.ID); ok && utf8.RuneCountInString(label) > 4 && catalogEntry.Abbr != "" {
				label = catalogEntry.Abbr
			}
			row["weapon"] = map[string]any{"icon": miao("resources/meta-sr/weapon/" + cone.Type + "/" + cone.Name + "/icon.webp"), "star": app.Int(weapon.Rarity),
				"name": label, "affix": weapon.Refinement, "badge": weapon.Refinement + 1, "level": weapon.Level}
		}
		// miao shows each set worn as two or four pieces by its first piece.
		counts, order := map[string]int{}, []string{}
		for _, piece := range panel.Equipment {
			if piece.SetName == "" {
				continue
			}
			if counts[piece.SetName] == 0 {
				order = append(order, piece.SetName)
			}
			counts[piece.SetName]++
		}
		sets := []any{}
		for _, set := range order {
			if counts[set] < 2 || len(sets) == 2 {
				continue
			}
			icon := miao("resources/meta-sr/artifact/" + set + "/arti-1.webp")
			if icon == "" {
				icon = miao("resources/meta-sr/artifact/" + set + "/arti-5.webp")
			}
			sets = append(sets, icon)
		}
		grade := entry.Grade
		if grade == "" {
			grade = "D"
		}
		row["sets"], row["set_count"], row["set_name"] = sets, len(sets), app.RankSetName(context.Catalog, panel)
		row["grade"], row["mark"] = grade, miaoComma(entry.Score, 1)
		if entry.Damage != nil {
			value := entry.Damage.Text
			if value == "" {
				value = miaoComma(entry.Damage.Value, 1)
			}
			row["damage"] = map[string]any{"title": app.RankDamageTitle(entry.Damage.Title), "value": value}
		}
		rows = append(rows, row)
	}
	title := "最强排行"
	if rank.Mode == "mark" {
		title = "最高分排行"
	}
	if rank.Character.ID != "" {
		name := rank.Character.Name
		// miao titles both Trailblazers 开拓者.
		for _, prefix := range []string{"穹·", "星·"} {
			if path, ok := strings.CutPrefix(name, prefix); ok {
				name = "开拓者·" + path
			}
		}
		title = name + map[string]string{"mark": "遗器评分"}[rank.Mode] + "排行"
	}
	since := time.UnixMilli(rank.SinceMS).In(chinaTime).Format("01-02 15:04")
	data := map[string]any{"elem": elem, "title": context.Game.Prefix + title, "mode": rank.Mode, "max": rank.Character.ID == "", "wide": wide, "since": since, "hash": context.Game.Prefix, "rows": rows}
	return app.Image{Template: "rank", Data: data, Resources: resources.List}, true
}
