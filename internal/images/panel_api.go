package images

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// APICard draws 面板列表 and 面板更新 the way StarRail-plugin's new_card page
// does: each kept character's avatar, name and eidolons, those the query
// returned marked after 面板更新, with the UID, the service, the update time
// and the sender's name. StarRail-plugin's avatars cover only its older
// characters; one it lacks shows miao's face, marked face so the page crops it
// to the round frame.
func APICard(context app.ImageContext, image app.APICardImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	resources.Artwork("SR", "starrail-plugin", "resources/common/font/SR.ttf")
	characters := []any{}
	for _, character := range image.Characters {
		id := strconv.Itoa(character.AvatarID)
		card := map[string]any{"avatar": resources.Artwork("avatar-"+id, "starrail-plugin", "resources/panel/resources/avatar/"+id+".png"),
			"name": character.Name, "rank": character.Rank, "updated": image.Update && character.IsNew}
		if card["avatar"] == "" {
			if face := miaoFace(resources, id); face != "" {
				card["avatar"], card["face"] = face, true
			}
		}
		characters = append(characters, card)
	}
	kind := ""
	if image.Update {
		kind = "update"
	}
	return app.Image{Template: "api-card", Data: map[string]any{
		"uid": image.UID, "api": image.API, "time": image.Time, "user_name": image.UserName, "type": kind, "characters": characters,
	}, Resources: resources.List}, true
}

// apiPanelArtwork maps the images and font the panel page's stylesheet names
// to StarRail-plugin's paths.
var apiPanelArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"api-panel-bkg", "resources/common/cont/bkg.png"},
	{"api-panel-line", "resources/panel/resources/dividing-line.png"},
}

// apiPanelValues are the page's value icons, under panel/resources/icon/value.
const apiPanelValues = "resources/panel/resources/icon/value/"

// APIPanel draws <角色>面板<UID> the way StarRail-plugin's new_panel page
// does: the picture, name, element, path, level and eidolons; the eleven
// stats; the light cone and skills; the relics with their scores and
// weighted substats; the skill tree; and the damage table.
func APIPanel(context app.ImageContext, image app.APIPanelImage) (app.Image, bool) {
	character := image.Character
	resources := &app.ImageResources{Context: context}
	for _, item := range apiPanelArtwork {
		resources.Artwork(item[0], "starrail-plugin", item[1])
	}
	// A file's resource ID is its directory's and its own name, as
	// value-IconMaxHP.png.
	local := func(name string) string {
		directory, file := path.Split(name)
		return resources.Artwork(path.Base(directory)+"-"+file, "starrail-plugin", name)
	}
	var paths map[string]struct {
		Path string `json:"path"`
	}
	starRailData(context, "resources/panel/data/character.json", &paths)
	lifePath := paths[strconv.Itoa(character.AvatarID)].Path
	element := character.DamageType
	// A total the service leaves out reads as 0, where upstream prints NaN.
	value := func(key string) float64 { return character.Properties[key] }
	added := func(icon, name, base, delta string) map[string]any {
		return map[string]any{"icon": local(apiPanelValues + icon + ".png"), "name": name, "value": jsNumber(math.Floor(value(base) + value(delta))),
			"base": jsNumber(floor10(value(base), -1)), "delta": jsNumber(floor10(value(delta), -1)), "added": true}
	}
	attributes := []map[string]any{
		added("IconMaxHP", "生命值", "hpBase", "hpDelta"), added("IconAttack", "攻击力", "attackBase", "attackDelta"),
		added("IconDefence", "防御力", "defenseBase", "defenseDelta"), added("IconSpeed", "速度", "speedBase", "speedAdd"),
	}
	percent := func(icon, name string, fraction float64) map[string]any {
		return map[string]any{"icon": local(apiPanelValues + icon + ".png"), "name": name, "value": jsNumber(floor10(fraction*100, -1)) + "%"}
	}
	attributes = append(attributes, percent("IconCriticalChance", "暴击率", value("criticalChance")), percent("IconCriticalDamage", "暴击伤害", value("criticalDamage")),
		percent("IconStatusResistance", "效果抵抗", value("statusResistance")), percent("IconStatusProbability", "效果命中", value("statusProbability")),
		percent("IconBreakUp", "击破特攻", value("breakDamageAdded")), percent("IconEnergyLimit", "能量回复", 1+value("spRatio")))
	if lifePath == "丰饶" {
		attributes = append(attributes, percent("IconHealRatio", "治疗量加成", value("healRatio")))
	} else {
		attributes = append(attributes, percent("Icon"+element+"AddedRatio", "属性伤害提高", value(strings.ToLower(element)+"Added")))
	}
	data := map[string]any{
		"picture": resources.Artwork("api-panel-picture", "starrail-plugin", image.Picture), "name": character.Name,
		"element": local("resources/common/element/" + strings.ToLower(element) + ".png"), "path": lifePath,
		"level": character.Level, "rank": character.Rank, "attributes": attributes,
		"skills": apiPanelSkills(local, image.Skills), "uid": image.UID, "api": image.API,
		"relics": apiPanelRelics(context, local, character.Relics), "tree": apiPanelTree(context, resources, lifePath, character.Behaviors),
		"damaged": image.Damages != nil, "damages": apiPanelDamages(image.Damages),
	}
	if equipment := character.Equipment; equipment.ID > 0 {
		id := strconv.Itoa(equipment.ID)
		// Upstream names the rarity by the last seven letters of the light
		// cone's, as Rarity5.
		rarity := equipment.Rarity[max(0, len(equipment.Rarity)-7):]
		data["equipment"] = map[string]any{"image": local("resources/panel/resources/weapon/" + id + ".png"), "name": equipment.Name, "level": equipment.Level, "rank": equipment.Rank,
			"star": local("resources/panel/resources/star/LightCore_" + rarity + ".png"), "hp_icon": local(apiPanelValues + "IconMaxHP.png"), "atk_icon": local(apiPanelValues + "IconAttack.png"),
			"def_icon": local(apiPanelValues + "IconDefence.png"), "hp": jsNumber(math.Floor(equipment.HP)), "atk": jsNumber(math.Floor(equipment.Attack)), "def": jsNumber(math.Floor(equipment.Defence))}
	}
	return app.Image{Template: "api-panel", Data: data, Resources: resources.List}, true
}

func apiPanelSkills(local func(string) string, skills []app.APISkill) []any {
	list := []any{}
	for _, skill := range skills {
		list = append(list, map[string]any{"icon": local("resources/panel/resources/skill/" + skill.File), "level": skill.Level, "type": skill.Type})
	}
	return list
}

// apiPanelRelics are the relic cards: the icon upstream's relic data names,
// the rarity as a third of the top level, the service's score, and each
// substat by the weight the service gives it, with a mark for each roll after
// the first.
func apiPanelRelics(context app.ImageContext, local func(string) string, relics []app.APIRelic) []any {
	var icons map[string]struct {
		Icon string `json:"icon"`
	}
	starRailData(context, "resources/panel/data/relics.json", &icons)
	list := []any{}
	for _, relic := range relics {
		rarity := jsNumber(float64(relic.MaxLevel) / 3)
		affixes := []any{}
		for _, affix := range relic.Affixes {
			weight := ""
			switch {
			case affix.Weight >= 1:
				weight = "weight-heavy"
			case affix.Weight > 0:
				weight = "weight-normal"
			}
			affixes = append(affixes, map[string]any{"weight": weight, "name": affix.Name, "rolls": strings.Repeat(">", max(0, affix.Count-1)), "value": apiRelicValue(affix.Value)})
		}
		score := "0"
		if relic.Score != 0 {
			score = app.JSFixed(relic.Score, 1)
		}
		icon := ""
		if file := icons[strconv.Itoa(relic.ID)].Icon; file != "" {
			icon = local("resources/panel/resources/relic/" + file)
		}
		list = append(list, map[string]any{"rank": rarity, "icon": icon, "level": relic.Level, "star": local("resources/panel/resources/star/LightCore_Rarity" + rarity + ".png"),
			"name": relic.Name, "score": score, "main_name": relic.MainName, "main_value": apiRelicValue(relic.MainValue), "affixes": affixes})
	}
	return list
}

// apiRelicValue writes a relic stat as upstream: from 1 up a flat number,
// below it a percentage.
func apiRelicValue(value float64) string {
	if value >= 1 {
		return jsNumber(math.Floor(value))
	}
	return jsNumber(floor10(value*100, -1)) + "%"
}

func apiPanelDamages(damages []app.APIDamage) []any {
	list := []any{}
	for _, damage := range damages {
		list = append(list, map[string]any{"title": damage.Title, "critical": apiDamageValue(damage.Value.Critical), "expect": apiDamageValue(damage.Value.Expect)})
	}
	return list
}

func apiDamageValue(value *float64) string {
	if value == nil {
		return "NaN"
	}
	return jsNumber(math.Floor(*value))
}

// apiTreeBackgrounds are the skill tree backgrounds by path (skillTreeImg);
// upstream has none for Remembrance and Elation.
var apiTreeBackgrounds = map[string]string{"存护": "Knight", "智识": "Mage", "丰饶": "Priest", "巡猎": "Rogue", "毁灭": "Warrior", "同谐": "Shaman", "虚无": "Warlock"}

// apiTreeIcons is where the page loads the trace icons the service names.
const apiTreeIcons = "https://avocado.wiki/"

var apiSVGSize = regexp.MustCompile(`^<svg[^>]*\swidth="([0-9.]+)"[^>]*\sheight="([0-9.]+)"`)

// apiPanelTree is the skill tree upstream draws on a canvas twice the size of
// the path's background, in the canvas's units: each trace at its place in
// upstream's layout, recoloured, the skills with their levels, locked traces
// darkened with a lock, and 加载失败 where an icon cannot be loaded. It is nil
// for a path without a layout or background, for which upstream fails to
// draw the whole panel.
func apiPanelTree(context app.ImageContext, resources *app.ImageResources, lifePath string, behaviors []app.APIBehavior) map[string]any {
	var layouts map[string]map[string][2]float64
	starRailData(context, "resources/panel/data/skillTree.json", &layouts)
	name, drawn := apiTreeBackgrounds[lifePath]
	if layouts[lifePath] == nil || !drawn || context.Artwork == nil {
		return nil
	}
	svg, err := context.Artwork.Open("starrail-plugin", "resources/panel/resources/skill_tree/"+name+".svg")
	size := apiSVGSize.FindSubmatch(svg)
	if err != nil || size == nil {
		return nil
	}
	width, _ := strconv.ParseFloat(string(size[1]), 64)
	height, _ := strconv.ParseFloat(string(size[2]), 64)
	width, height = width*2, height*2
	urls := []any{}
	for _, behavior := range behaviors {
		if behavior.Icon != "" {
			urls = append(urls, apiTreeIcons+strings.TrimPrefix(behavior.Icon, "/"))
		}
	}
	resources.Prefetch("panel-api", urls...)
	nodes := []any{}
	for _, behavior := range behaviors {
		place, placed := layouts[lifePath][behavior.Anchor]
		number, _ := strconv.Atoi(strings.TrimPrefix(behavior.Anchor, "Point"))
		// The skills, the major and the minor traces; upstream gives other
		// anchors no size, and its layouts place none.
		size, skill := 0.0, false
		switch {
		case !placed:
			continue
		case number >= 1 && number <= 5:
			size, skill = 138, true
		case number >= 6 && number <= 8:
			size = 144
		case number >= 9 && number <= 18:
			size = 102
		default:
			continue
		}
		// The canvas squeezes 加载失败, four 30px characters, into a narrower
		// node.
		x, y := place[0], place[1]
		node := map[string]any{"cx": x + size/2, "cy": y + size/2, "r": size / 2, "skill": skill, "locked": behavior.Level == 0,
			"lock_cx": x + size - 10, "lock_cy": y + 10, "lock_x": x + size - 20, "lock_y": y, "icon_at": x + 5, "icon_top": y + 5, "icon_size": size - 10,
			"text_y": y + size + 60, "squeeze": size < 120, "size": size}
		if skill && behavior.Level > 0 {
			node["level"] = strconv.Itoa(behavior.Level)
			if behavior.MaxLevel > 0 {
				node["level"] = node["level"].(string) + "/" + strconv.Itoa(behavior.MaxLevel)
			}
		}
		if behavior.Icon != "" {
			node["icon"] = resources.URL("panel-api", apiTreeIcons+strings.TrimPrefix(behavior.Icon, "/"))
		}
		nodes = append(nodes, node)
	}
	return map[string]any{"width": width, "height": height, "nodes": nodes,
		"background": `<image href="data:image/svg+xml;base64,` + base64.StdEncoding.EncodeToString(svg) + `" width="` + jsNumber(width) + `" height="` + jsNumber(height) + `" preserveAspectRatio="none"/>`,
		"lock":       resources.Artwork("api-panel-lock", "starrail-plugin", "resources/panel/resources/skill_tree/lock.png")}
}

// starRailData reads a JSON file of the starrail-plugin source into out,
// leaving it empty when the file cannot be read.
func starRailData(context app.ImageContext, name string, out any) {
	if context.Artwork == nil {
		return
	}
	if raw, err := context.Artwork.Open("starrail-plugin", name); err == nil {
		_ = json.Unmarshal(raw, out)
	}
}

// floor10 is StarRail-plugin's MathPro.floor10: the value floored at the
// given power of ten, shifted through its decimal text as upstream does, so
// that binary rounding does not cost a digit.
func floor10(value float64, exp int) float64 {
	if exp == 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return math.Floor(value)
	}
	shift := func(value float64, by int) float64 {
		mantissa, power, _ := strings.Cut(strconv.FormatFloat(value, 'e', -1, 64), "e")
		exponent, _ := strconv.Atoi(power)
		shifted, _ := strconv.ParseFloat(mantissa+"e"+strconv.Itoa(exponent+by), 64)
		return shifted
	}
	return shift(math.Floor(shift(value, -exp)), exp)
}

// jsNumber writes a number as JavaScript prints it.
func jsNumber(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	case value == 0:
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
