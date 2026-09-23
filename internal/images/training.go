package images

import (
	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// commonItemArtwork is miao's shared item frames and fonts the training page
// uses.
var commonItemArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"common-item-artifact-icon", "resources/common/item/artifact-icon.webp"},
	{"common-item-bg1", "resources/common/item/bg1.png"},
	{"common-item-bg2", "resources/common/item/bg2.png"},
	{"common-item-bg3", "resources/common/item/bg3.png"},
	{"common-item-bg4", "resources/common/item/bg4.png"},
	{"common-item-bg5", "resources/common/item/bg5.png"},
	{"common-item-bg1-o", "resources/common/item/bg1-o.png"},
	{"common-item-bg2-o", "resources/common/item/bg2-o.png"},
	{"common-item-bg3-o", "resources/common/item/bg3-o.png"},
	{"common-item-bg4-o", "resources/common/item/bg4-o.png"},
	{"common-item-bg5-o", "resources/common/item/bg5-o.png"},
	{"common-item-crown-o", "resources/common/item/crown-o.png"},
	{"common-item-crown-sr-o", "resources/common/item/crown-sr-o.png"},
	{"common-item-fetter", "resources/common/item/fetter.png"},
	{"common-bg-bg-hydro", "resources/common/bg/bg-hydro.webp"},
	{"character-imgs-bg-02", "resources/character/imgs/bg-02.jpg"},
	{"character-imgs-main-02", "resources/character/imgs/main-02.png"},
}

// trainingLevels is miao's talent colour step for each original level; basic
// attacks, which stop at 6 in Star Rail, use their own steps.
var (
	trainingLevels       = []int{0, 1, 1, 1, 2, 2, 3, 3, 3, 4, 5}
	trainingAttackLevels = []int{0, 1, 1, 2, 3, 4, 5}
)

// Training draws 练度统计 the way miao's character/profile-stat does for Star
// Rail: every character in miao's order with level, eidolons, the four skills
// coloured by their original level, the light cone, and the relic sets with
// the pinned scoring's grade and score.
func Training(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	list, _ := result.Data["avatar_list"].([]any)
	if len(list) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	for _, item := range commonItemArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	rows := []any{}
	for _, item := range buildRoster(context, resources, list) {
		card := item.card
		talents := []any{}
		levels, _ := card["talents"].([]any)
		for index, key := range []string{"a", "e", "q", "t"} {
			level, original := any("-"), 1
			if index < len(levels) {
				talent := levels[index].(map[string]any)
				level, original = talent["level"], app.Int(talent["original"])
			}
			steps := trainingLevels
			if key == "a" {
				steps = trainingAttackLevels
			}
			talents = append(talents, map[string]any{"level": level, "class": steps[min(max(original, 0), len(steps)-1)], "plus": app.Int(level) > original})
		}
		row := map[string]any{"no": len(rows) + 1, "star": item.star, "face": card["face"], "name": card["abbr"], "level": item.level, "cons": item.cons,
			"talents": talents, "artis": card["artis"]}
		if weapon, _ := card["weapon"].(map[string]any); weapon != nil {
			row["weapon"] = map[string]any{"star": weapon["star"], "level": weapon["level"], "icon": weapon["small"], "affix": weapon["affix"],
				"badge": app.Int(weapon["affix"]) + 1, "name": weapon["name"]}
		}
		// Characters with relics get the pinned scoring's grade and score.
		if item.panel != nil && len(item.panel.Equipment) > 0 && context.Score != nil {
			if scored, err := context.Score(*item.panel); err == nil && scored.ScoreDetail != nil {
				row["mark"], row["grade"] = scored.ScoreDetail.Mark, scored.ScoreDetail.Grade
			}
		}
		rows = append(rows, row)
	}
	return app.Image{Template: "training", Data: map[string]any{
		"uid": result.Role.UID, "count": len(rows), "rows": rows, "updated": context.Now.In(chinaTime).Format("2006-01-02 15:04"),
	}, Resources: resources.List}, true
}
