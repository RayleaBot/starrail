package images

import (
	"fmt"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

// helpIcons gives each command the icon of the StarRail-plugin help entry
// that lists it (config/system/help_system.js); commands upstream does not
// list show none, as upstream hides an entry's missing icon.
var helpIcons = map[string]int{
	"select": 78, "showcase": 63, "character": 66, "monthly": 98, "monthly-history": 98, "note": 95, "profile": 75,
	"gacha": 97, "gacha-summary": 97, "gacha-background": 97,
	"rogue": 107, "divergent": 107, "currency_wars": 107, "swarm": 107,
	"challenge": 110, "story": 110, "boss": 110, "peak": 110, "abyss": 110, "abyss-latest": 110,
	"guides": 106, "simulation": 100, "codes": 96, "artwork": 85,
}

// helpColumns is StarRail-plugin's column count.
const helpColumns = 3

// helpArtwork maps the images named in the converted stylesheets
// (StarRail-plugin's common and help/index, with the default theme) and its
// fonts to their paths.
var helpArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"tttgbnumber", "resources/common/font/tttgbnumber.ttf"},
	{"YS", "resources/common/font/YS.ttf"},
	{"help-icon", "resources/help/icon.png"},
	{"help-theme-bg", "resources/help/theme/default/bg.jpg"},
	{"help-theme-main", "resources/help/theme/default/main.png"},
}

// Help draws the help menu the way StarRail-plugin's help/index does: the
// title bar, then each group as a table of three columns, each command with
// its icon, usage and description, on its default theme.
func Help(context gamekit.ImageContext, help gamekit.HelpImage) (gamekit.Image, bool) {
	if len(help.Groups) == 0 {
		return gamekit.Image{}, false
	}
	resources := &gamekit.ImageResources{Context: context}
	for _, item := range helpArtwork {
		resources.Artwork(item[0], "starrail-plugin", item[1])
	}
	groups := []any{}
	for _, group := range help.Groups {
		rows, row := []any{}, []any{}
		for _, command := range group.Commands {
			cell := map[string]any{"title": command.Usage, "desc": command.Description}
			// The sprite has ten 50px icons a row.
			if icon := helpIcons[command.ID]; icon > 0 {
				x, y := (icon-1)%10, (icon-1)/10
				cell["position"] = fmt.Sprintf("-%dpx -%dpx", x*50, y*50)
			}
			row = append(row, cell)
			if len(row) == helpColumns {
				rows, row = append(rows, row), []any{}
			}
		}
		if len(row) > 0 {
			for len(row) < helpColumns {
				row = append(row, nil)
			}
			rows = append(rows, row)
		}
		groups = append(groups, map[string]any{"title": group.Title, "rows": rows})
	}
	return gamekit.Image{Template: "help", Data: map[string]any{"title": help.Title, "subtitle": help.Subtitle, "groups": groups}, Resources: resources.List}, true
}
