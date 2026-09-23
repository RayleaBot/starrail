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

// rankLayout is the talents miao's rank list shows for the ranked
// character's path, and the page's width among widths for Remembrance,
// Elation and the others: memosprite skills for Remembrance, the Elation
// skill for Elation; other paths, and the list of every character's best,
// show four.
func rankLayout(path string, widths [3]int) ([]string, int) {
	switch path {
	case "记忆":
		return []string{"a", "e", "t", "q", "me", "mt"}, widths[0]
	case "欢愉":
		return []string{"a", "e", "t", "q", "xe"}, widths[1]
	}
	return []string{"a", "e", "t", "q"}, widths[2]
}

// Rank draws a group ranking the way miao's character/rank-profile-list
// does in its Star Rail mode: the title and ranking notes, then a row per
// entry with its place (or, listing every character's best, the character),
// the member's avatar, Eidolon, name and UID, the talents, light cone, relic
// sets with their score and grade, and the default detail's damage.
func Rank(context app.ImageContext, rank app.RankImage) (app.Image, bool) {
	if context.Game.Calc == nil || len(rank.Entries) == 0 {
		return app.Image{}, false
	}
	page := newRankPage(context)
	ranked := page.records[rank.Character.ID]
	elem := "hydro"
	if elementClasses[ranked.Element] != "" {
		elem = elementClasses[ranked.Element]
	}
	page.resources.Artwork("common-bg-bg-"+elem, "miao-plugin", "resources/common/bg/bg-"+elem+".webp")
	keys, width := rankLayout(ranked.WeaponType, [3]int{970, 970, 900})
	rows := []any{}
	for index, entry := range rank.Entries {
		record, panel := page.records[entry.CharacterID], *entry.Panel
		row := page.row(index+1, entry.UID, record, panel.Rank)
		if entry.Avatar != "" {
			row["avatar"] = page.resources.Remote("avatar-"+strconv.Itoa(index), entry.Avatar)
		}
		row["talents"] = page.talents(record, keys, app.PanelTalents(panel, record))
		if weapon := panel.Weapon; weapon != nil {
			row["weapon"] = page.weapon(page.weapons[weapon.ID], weapon.Level, weapon.Refinement)
		}
		// miao shows each set worn as two or four pieces.
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
			if counts[set] >= 2 && len(sets) < 2 {
				sets = append(sets, page.setIcon(set))
			}
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
	data := map[string]any{"elem": elem, "title": context.Game.Prefix + title, "mode": rank.Mode, "max": rank.Character.ID == "", "time": since, "hash": context.Game.Prefix, "rows": rows, "width": width}
	return app.Image{Template: "rank", Data: data, Resources: page.resources.List}, true
}

// CloudRank draws ark-plugin's custom ranking as ark fills miao's
// rank-profile-list for Star Rail: the character's face in each row with the
// UID ark shows, the Eidolon, talents, light cone, sets and score it returns
// and the ranked detail's average damage; the notes read 全服数据, ark's order
// and filters and the rows asked for. ark draws the page 930 pixels wide,
// 960 for Elation and 1000 for Remembrance characters.
func CloudRank(context app.ImageContext, rank app.CloudRankImage) (app.Image, bool) {
	if context.Game.Calc == nil || len(rank.Entries) == 0 {
		return app.Image{}, false
	}
	page := newRankPage(context)
	record, ok := page.records[rank.Character.ID]
	if !ok {
		return app.Image{}, false
	}
	elem := elementClasses[record.Element]
	page.resources.Artwork("common-bg-bg-"+elem, "miao-plugin", "resources/common/bg/bg-"+elem+".webp")
	keys, width := rankLayout(record.WeaponType, [3]int{1000, 960, 930})
	rows := []any{}
	for index, entry := range rank.Entries {
		row := page.row(index+1, entry.UID, record, entry.Cons)
		row["talents"] = page.talents(record, keys, entry.Talents)
		if cone, ok := page.weapons[entry.Weapon]; ok {
			row["weapon"] = page.weapon(cone, entry.WeaponLevel, entry.WeaponAffix)
		}
		sets := []any{}
		for _, name := range entry.Sets[:min(2, len(entry.Sets))] {
			sets = append(sets, page.setIcon(name))
		}
		row["sets"], row["set_count"], row["set_name"], row["grade"], row["mark"] = sets, len(sets), entry.SetName, customRankGrade(entry.Mark), miaoComma(entry.Mark, 1)
		if entry.Damage != nil {
			row["damage"] = map[string]any{"title": rank.DamageTitle, "value": miaoComma(*entry.Damage, 1)}
		}
		rows = append(rows, row)
	}
	title := context.Game.Prefix + record.Name + map[string]string{"mark": "圣遗物评分"}[rank.Mode] + "排行"
	data := map[string]any{"elem": elem, "title": title, "mode": rank.Mode, "max": false, "time": "全服数据", "hash": context.Game.Prefix, "rows": rows, "width": width,
		"limit": rank.Limit, "number": rank.Number}
	return app.Image{Template: "rank", Data: data, Resources: page.resources.List}, true
}

// customRankGrade is ark's grade of a custom ranking row: the average score
// of the six relics against miao's grade steps, MAX past the last.
func customRankGrade(mark float64) string {
	average := mark / 6
	for _, step := range []struct {
		grade string
		below float64
	}{{"D", 7}, {"C", 14}, {"B", 21}, {"A", 28}, {"S", 35}, {"SS", 42}, {"SSS", 49}, {"ACE", 56}, {"MAX", 70}} {
		if average < step.below {
			return step.grade
		}
	}
	return "MAX"
}

// rankPage is what a ranking page draws its rows from: the calculation
// records and light cones by ID, and the page's resources.
type rankPage struct {
	context   app.ImageContext
	resources *app.ImageResources
	records   map[string]reference.Character
	weapons   map[string]reference.Weapon
}

func newRankPage(context app.ImageContext) *rankPage {
	metadata := context.Game.Calc.Metadata()
	page := &rankPage{context: context, resources: &app.ImageResources{Context: context}, records: map[string]reference.Character{}, weapons: map[string]reference.Weapon{}}
	for _, record := range metadata.Characters {
		page.records[record.ID] = record
	}
	for _, weapon := range metadata.Weapons {
		page.weapons[weapon.ID] = weapon
	}
	for _, item := range rankArtwork {
		page.resources.Artwork(item[0], "miao-plugin", item[1])
	}
	return page
}

func (p *rankPage) miao(name string) string {
	return p.resources.Artwork("miao-"+strconv.Itoa(len(p.resources.List)), "miao-plugin", name)
}

// row starts a row as miao lays it out: the place and UID, the character's
// face and rarity, its Eidolon and name, abbreviated from four characters on
// as miao's sName.
func (p *rankPage) row(place int, uid string, record reference.Character, cons int) map[string]any {
	star := app.Int(record.Data["star"])
	if star != 4 {
		star = 5
	}
	name := record.Name
	if utf8.RuneCountInString(name) >= 4 {
		name = abbreviation(p.context.Catalog, record)
	}
	// Upstream's rows class the talents by the character's element name,
	// which no Star Rail style matches.
	return map[string]any{"place": place, "uid": uid, "star": star, "face": p.miao("resources/meta-sr/character/" + record.Name + "/imgs/face.webp"), "cons": cons, "name": name, "elem": record.Element}
}

// talents are a row's talent cells: each talent's icon with its level and
// miao's bonus and crown marks, no level for a talent without one.
func (p *rankPage) talents(record reference.Character, keys []string, levels map[string]app.PanelTalent) []any {
	cells := []any{}
	for _, key := range keys {
		cell := map[string]any{"icon": p.miao("resources/meta-sr/character/" + record.Name + "/imgs/talent-" + key + ".webp"), "level": ""}
		if level, ok := levels[key]; ok {
			cell["level"], cell["plus"], cell["crown"] = level.Level, level.Level > level.Original, level.Original >= 10
		}
		cells = append(cells, cell)
	}
	return cells
}

// weapon is the light cone cell: its icon and rarity, the name (its
// abbreviation past four characters), superimposition and level.
func (p *rankPage) weapon(cone reference.Weapon, level, affix int) map[string]any {
	label, star := cone.Name, 0
	if entry, ok := p.context.Catalog.Get(cone.ID); ok {
		star = entry.Rarity
		if utf8.RuneCountInString(label) > 4 && entry.Abbr != "" {
			label = entry.Abbr
		}
	}
	return map[string]any{"icon": p.miao("resources/meta-sr/weapon/" + cone.Type + "/" + cone.Name + "/icon.webp"), "star": star, "name": label, "affix": affix, "badge": affix + 1, "level": level}
}

// setIcon is miao's picture of a relic set: its first piece, the sphere for
// planar ornaments.
func (p *rankPage) setIcon(name string) string {
	if icon := p.miao("resources/meta-sr/artifact/" + name + "/arti-1.webp"); icon != "" {
		return icon
	}
	return p.miao("resources/meta-sr/artifact/" + name + "/arti-5.webp")
}
