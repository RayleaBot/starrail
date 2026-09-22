package images

import (
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

// rosterEntry is one owned character as miao's avatar card shows it, with
// what miao sorts rosters by.
type rosterEntry struct {
	id                                  string
	card                                map[string]any
	panel                               *app.CharacterPanel
	level, star, aeq, cons              int
	weaponLevel, weaponStar, refinement int
}

// buildRoster reads the requester's characters and their details (one detail
// query without IDs answers for all of them) as miao's avatar cards, in
// miao's order: level, rarity, original talents, eidolons, light cone level,
// rarity and superimposition, highest first.
func buildRoster(context app.ImageContext, resources *app.ImageResources, list []any) []rosterEntry {
	records, weapons := map[string]reference.Character{}, map[string]reference.Weapon{}
	if context.Game.Calc != nil {
		metadata := context.Game.Calc.Metadata()
		for _, record := range metadata.Characters {
			records[record.ID] = record
		}
		for _, weapon := range metadata.Weapons {
			weapons[weapon.ID] = weapon
		}
	}
	panels := map[string]app.CharacterPanel{}
	if context.Query != nil {
		if result, err := context.Query("starrail.character", nil); err == nil {
			for _, panel := range app.NormalizePanels("starrail", result, context.Catalog) {
				panels[panel.ID] = panel
			}
		}
	}
	miao := func(name string) string {
		return resources.Artwork("miao-"+strconv.Itoa(len(resources.List)), "miao-plugin", name)
	}
	entries := []rosterEntry{}
	for _, raw := range list {
		avatar, _ := raw.(map[string]any)
		id := app.Text(avatar["id"])
		record := records[id]
		star := app.Int(record.Data["star"])
		if star != 4 {
			star = 5
		}
		card := map[string]any{"known": record.Name != "", "name": record.Name, "abbr": abbreviation(context.Catalog, record), "elem": elementClasses[record.Element], "star": star,
			"face": miao("resources/meta-sr/character/" + record.Name + "/imgs/face.webp"), "type": "mini",
			"level": app.Int(avatar["level"]), "cons": app.Int(avatar["rank"]), "artis": []any{}}
		item := rosterEntry{id: id, card: card, star: star, aeq: 4}
		if panel, ok := panels[id]; ok {
			item.panel = &panel
			card["level"], card["cons"] = panel.Level, panel.Rank
			levels := app.PanelTalents("starrail", panel, record)
			talents := []any{}
			item.aeq = 0
			for _, key := range []string{"a", "e", "q", "t"} {
				level := levels[key]
				item.aeq += level.Original
				talents = append(talents, map[string]any{"key": key, "level": level.Level, "original": level.Original, "plus": level.Level > level.Original, "crown": level.Original == 10})
			}
			if levels["a"].Level > 0 {
				card["talents"] = talents
			} else {
				item.aeq = 4
			}
			if weapon := panel.Weapon; weapon != nil {
				entry := weapons[weapon.ID]
				name := weapon.Name
				if catalogEntry, ok := context.Catalog.Get(weapon.ID); ok {
					name = catalogEntry.Name
					// miao shortens names longer than four characters.
					if utf8.RuneCountInString(name) > 4 && catalogEntry.Abbr != "" {
						name = catalogEntry.Abbr
					}
				}
				badge := weapon.Refinement
				if badge > 4 {
					badge++
				}
				path := "resources/meta-sr/weapon/" + entry.Type + "/" + entry.Name + "/"
				card["weapon"] = map[string]any{"icon": miao(path + "icon.webp"), "small": miao(path + "icon-s.webp"), "star": app.Int(weapon.Rarity),
					"affix": weapon.Refinement, "badge": badge, "level": weapon.Level, "name": name}
				item.weaponLevel, item.weaponStar, item.refinement = weapon.Level, app.Int(weapon.Rarity), weapon.Refinement
			}
			// miao shows each set worn as two or four pieces, by its head (or
			// its planar sphere for ornaments).
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
			for _, name := range order {
				if counts[name] < 2 {
					continue
				}
				icon := miao("resources/meta-sr/artifact/" + name + "/arti-1.webp")
				if icon == "" {
					icon = miao("resources/meta-sr/artifact/" + name + "/arti-5.webp")
				}
				sets = append(sets, icon)
			}
			card["artis"] = sets
		}
		item.level, item.cons = app.Int(card["level"]), app.Int(card["cons"])
		entries = append(entries, item)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		for _, pair := range [][2]int{{a.level, b.level}, {a.star, b.star}, {a.aeq, b.aeq}, {a.cons, b.cons}, {a.weaponLevel, b.weaponLevel}, {a.weaponStar, b.weaponStar}, {a.refinement, b.refinement}} {
			if pair[0] != pair[1] {
				return pair[0] > pair[1]
			}
		}
		return false
	})
	return entries
}
