package images

import (
	"strconv"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// simulationImages is where StarRail-plugin keeps its 十连 page's pictures.
const simulationImages = "resources/gachasimulation/resources/images/"

// simulationElements are StarRail-plugin's damage-type icon names by the
// catalog's element names.
var simulationElements = map[string]string{"火": "Fire", "冰": "Ice", "虚数": "Imaginary", "物理": "Physical", "量子": "Quantum", "雷": "Thunder", "风": "Wind"}

// simulationPaths are miao's light cone directories with StarRail-plugin's
// path icon for each; the two newer paths use its BgPaths pictures.
var simulationPaths = [][2]string{{"毁灭", "毁灭"}, {"巡猎", "巡猎"}, {"智识", "智识"}, {"同谐", "同谐"}, {"虚无", "虚无"}, {"存护", "存护"}, {"丰饶", "丰饶"},
	{"记忆", "BgPathsMemory"}, {"欢愉", "BgPathsJoy"}}

// GachaSimulation draws 十连 the way StarRail-plugin's gachasimulation page
// does: the draws in order on three tilted rows of three, four and three
// cards, each coloured by rarity, characters with their element and
// light cones with their path, and five-stars with the pulls they took.
// StarRail-plugin's pictures cover only its own old pool; a character or light
// cone it lacks shows miao's splash or icon instead.
func GachaSimulation(context app.ImageContext, simulation app.SimulationImage) (app.Image, bool) {
	if context.Artwork == nil || len(simulation.Draws) == 0 {
		return app.Image{}, false
	}
	resources := &app.ImageResources{Context: context}
	resources.Artwork("SR", "starrail-plugin", "resources/common/font/SR.ttf")
	picture := func(id, name string) string { return resources.Artwork(id, "starrail-plugin", simulationImages+name) }
	cards := []any{}
	for index, draw := range simulation.Draws {
		rank := "three"
		switch draw.Rarity {
		case 5:
			rank = "five"
		case 4:
			rank = "four"
		}
		card := map[string]any{"rank": rank, "character": draw.Item == "character", "stars": make([]int, draw.Rarity)}
		if draw.Rarity == 5 {
			count := strconv.Itoa(draw.Interval) + "抽"
			if draw.Guaranteed {
				count = "大保底 " + count
			}
			card["count"] = count
		}
		id := "draw-" + strconv.Itoa(index)
		entry, _ := context.Catalog.Resolve(draw.Name, draw.Item, nil)
		if entry.Name == "" {
			entry.Name = draw.Name
		}
		if draw.Item == "character" {
			card["image"] = picture(id, "char_image/"+entry.ID+".png")
			if card["image"] == "" {
				card["image"] = resources.Artwork(id, "miao-plugin", "resources/meta-sr/character/"+entry.Name+"/imgs/splash.webp")
			}
			if element := simulationElements[entry.Element]; element != "" {
				card["element"] = picture("damage-"+element, "damage/IconAttribute"+element+".png")
			}
		} else {
			card["image"] = picture(id, "weapon/"+entry.ID+".png")
			for number, path := range simulationPaths {
				icon := "resources/meta-sr/weapon/" + path[0] + "/" + entry.Name + "/icon.webp"
				if _, ok := context.Artwork.File("miao-plugin", icon); !ok {
					continue
				}
				if card["image"] == "" {
					card["image"] = resources.Artwork(id, "miao-plugin", icon)
				}
				card["path"] = picture("path-"+strconv.Itoa(number), "weaponlife/"+path[1]+".png")
				break
			}
		}
		cards = append(cards, card)
	}
	// StarRail-plugin splits the ten into rows of three, four and three.
	rows := []any{}
	for index, bounds := range [][2]int{{0, 3}, {3, 7}, {7, 10}} {
		if bounds[0] >= len(cards) {
			break
		}
		rows = append(rows, map[string]any{"class": "line" + strconv.Itoa(index+1), "cards": cards[bounds[0]:min(bounds[1], len(cards))]})
	}
	data := map[string]any{"rows": rows, "icon": picture("icon", "DrawcardIcon.png"), "background": picture("background", "GachaImg_Common_Star.png"),
		"line": picture("line", "bkg-line.png"), "sparkle": picture("sparkle", "CS_Gacha_BG_Stars3.png"), "star": picture("star", "StarBig.png")}
	return app.Image{Template: "gacha-simulation", Data: data, Resources: resources.List}, true
}
