package images

import (
	"encoding/json"
	"html"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// wikiArtwork maps the images the talent page's converted stylesheets name
// to miao's paths.
var wikiArtwork = [][2]string{
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"wiki-imgs-card-bg", "resources/wiki/imgs/card-bg.png"},
}

// wikiTalentKeys are the talents miao's page lists for a Star Rail character,
// and wikiShortKeys the ones whose table runs Lv1 to Lv7 instead of Lv6 to
// Lv12; miao fills their descriptions at level 5 instead of 8.
var (
	wikiTalentKeys = []string{"a", "e", "q", "t", "z", "xe", "me", "mt"}
	wikiShortKeys  = []string{"a", "a2", "me", "me2", "mt", "mt1", "mt2"}
	wikiValue      = regexp.MustCompile(`\$(\d+)\[(i|f1|f2)](%?)`)
)

// wikiCharacter is the part of miao's Star Rail data.json the page reads.
type wikiCharacter struct {
	Name, Desc, Elem string
	BaseAttr         map[string]float64
	TalentCons       map[string]any
	Talent           map[string]struct {
		Name, Desc string
		Tables     map[string]struct {
			Name   string
			IsSame bool
			Values []float64
		}
	}
	Cons     map[string]struct{ Name, Desc string }
	TreeData map[string]struct {
		Type, Name, Desc string
		Idx              int
	}
}

// Entry draws miao's talent page for a Star Rail character: 天赋 and 行迹
// list the abilities and traces, 星魂 the Eidolons. The page reads miao's
// character data, which comes with the downloaded miao artwork.
func Entry(context app.ImageContext, page app.EntryImage) (app.Image, bool) {
	if page.Command != "talent-wiki" || page.Entry.Kind != "character" || context.Artwork == nil {
		return app.Image{}, false
	}
	raw, err := context.Artwork.Open("miao-plugin", "resources/meta-sr/character/"+page.Entry.Name+"/data.json")
	if err != nil {
		return app.Image{}, false
	}
	var character wikiCharacter
	if json.Unmarshal(raw, &character) != nil || len(character.Talent) == 0 {
		return app.Image{}, false
	}
	cons := strings.Contains(page.Word, "星魂") || strings.Contains(page.Word, "命")
	return characterTalent(context, character, cons), true
}

// characterTalent draws miao's wiki/character-talent in its Star Rail mode:
// the card, name and description with the level-80 base stats, then every
// ability with its description filled at level 8 (5 for basic attacks and the
// memosprite's) and its level table, and the traces, or the six Eidolons.
func characterTalent(context app.ImageContext, character wikiCharacter, cons bool) app.Image {
	resources := &app.ImageResources{Context: context}
	for _, item := range wikiArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	base := "resources/meta-sr/character/" + character.Name + "/imgs/"
	image := func(id, name string) string { return resources.Artwork(id, "miao-plugin", base+name+".webp") }
	line := []any{}
	for _, stat := range [][2]string{{"hp", "基础生命"}, {"atk", "基础攻击"}, {"def", "基础防御"}, {"speed", "速度"}} {
		line = append(line, map[string]any{"num": miaoComma(character.BaseAttr[stat[0]], 1), "label": stat[1]})
	}
	face := image("face", "face-q")
	if face == "" {
		face = image("face", "face")
	}
	// Every Star Rail character shares miao's common card.
	card := resources.Artwork("card", "miao-plugin", "resources/meta-sr/character/common/imgs/card.webp")
	// miao's elemLayout classes the page by element.
	elem := map[string]string{"火": "pyro", "冰": "cryo", "风": "anemo", "雷": "electro", "量子": "quantum", "虚数": "geo", "物理": "sr"}[character.Elem]
	if elem == "" {
		elem = "hydro"
	}
	data := map[string]any{"elem": elem, "name": character.Name, "desc": character.Desc, "line": line, "cons": cons, "face": face, "card": card}
	if cons {
		list := []any{}
		for index := 1; index <= 6; index++ {
			key := strconv.Itoa(index)
			eidolon, ok := character.Cons[key]
			if !ok {
				continue
			}
			// Eidolons 3 and 5 raise an ability and show its icon.
			icon := "cons-" + key
			if talent, _ := character.TalentCons[key].(string); talent != "" {
				icon = "talent-" + talent
			}
			list = append(list, wikiDetail(eidolon.Name, strings.Split(eidolon.Desc, "<br>"), nil, image(icon, icon), 0, 0))
		}
		data["constellations"] = list
		return app.Image{Template: "character-talent", Data: data, Resources: resources.List}
	}
	talents := []any{}
	for _, key := range wikiTalentKeys {
		if _, ok := character.Talent[key]; !ok {
			continue
		}
		parts := []any{}
		for _, part := range []string{key, key + "1", key + "2"} {
			talent, ok := character.Talent[part]
			if !ok {
				continue
			}
			short := slices.Contains(wikiShortKeys, part)
			level, first, last := 8, 6, 12
			if short {
				level, first, last = 5, 1, 7
			}
			desc, tables := wikiFill(talent.Desc, talent.Tables, level)
			parts = append(parts, wikiDetail(talent.Name, desc, tables, image("talent-"+part, "talent-"+part), first, last))
		}
		_, merged := character.Talent[key+"2"]
		talents = append(talents, map[string]any{"merged": merged, "parts": parts})
	}
	// Traces run in the order of their IDs, as miao iterates them.
	ids := []string{}
	for id := range character.TreeData {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int {
		if len(a) != len(b) {
			return len(a) - len(b)
		}
		return strings.Compare(a, b)
	})
	passives := []any{}
	for _, id := range ids {
		trace := character.TreeData[id]
		if trace.Type != "skill" {
			continue
		}
		passives = append(passives, wikiDetail(trace.Name, strings.Split(trace.Desc, "<br>"), nil, image("tree-"+strconv.Itoa(trace.Idx), "tree-"+strconv.Itoa(trace.Idx)), 0, 0))
	}
	data["talents"], data["passives"] = talents, passives
	return app.Image{Template: "character-talent", Data: data, Resources: resources.List}
}

type wikiTable struct {
	name   string
	same   bool
	values []string
}

// wikiFill is miao's getDesc: each $N[i|f1|f2] placeholder becomes the
// table's value at the level, as a percentage when followed by %, marked
// with the level; the tables are formatted the same way at every level.
func wikiFill(desc string, tables map[string]struct {
	Name   string
	IsSame bool
	Values []float64
}, level int) ([]string, []wikiTable) {
	percent := map[string]bool{}
	for _, match := range wikiValue.FindAllStringSubmatch(desc, -1) {
		table, ok := tables[match[1]]
		if !ok || len(table.Values) < level || table.Values[level-1] == 0 {
			continue
		}
		value := table.Values[level-1]
		text := miaoComma(value, 0)
		if match[3] == "%" {
			percent[match[1]] = true
			digits := 1
			if match[2] == "f2" {
				digits = 2
			}
			text = jsFixed(value*100, digits) + "%"
		}
		desc = strings.ReplaceAll(desc, match[0], text+" (lv"+strconv.Itoa(level)+")")
	}
	keys := []string{}
	for key := range tables {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int { x, _ := strconv.Atoi(a); y, _ := strconv.Atoi(b); return x - y })
	out := []wikiTable{}
	for _, key := range keys {
		table := tables[key]
		values := []string{}
		for _, value := range table.Values {
			if percent[key] {
				values = append(values, jsFixed(value*100, 1)+"%")
			} else {
				values = append(values, miaoComma(value, 0))
			}
		}
		out = append(out, wikiTable{name: table.Name, same: table.IsSame, values: values})
	}
	return strings.Split(desc, "<br>"), out
}

// wikiDetail is miao's talent-detail block: the icon, name and description,
// the values every level shares, and the table from level first to last.
func wikiDetail(name string, desc []string, tables []wikiTable, icon string, first, last int) map[string]any {
	var text strings.Builder
	for index, line := range desc {
		text.WriteString(miaoRichText(line))
		heading := (strings.HasPrefix(line, "<h3>") && strings.HasSuffix(line, "</h3>")) || (strings.HasPrefix(line, "<i>") && strings.HasSuffix(line, "</i>"))
		if index < len(desc)-1 && !heading {
			text.WriteString("<br>")
		}
	}
	shared, rows, levels := []any{}, []any{}, []string{}
	for _, table := range tables {
		if table.same {
			value := ""
			if len(table.values) > 0 {
				value = table.values[0]
			}
			shared = append(shared, map[string]any{"name": table.name, "value": value})
			continue
		}
		rows = append(rows, map[string]any{"name": table.name, "values": table.values[min(first-1, len(table.values)):min(last, len(table.values))]})
	}
	for level := first; level <= last && first > 0; level++ {
		levels = append(levels, "Lv"+strconv.Itoa(level))
	}
	return map[string]any{"icon": icon, "name": name, "desc": text.String(), "tables": len(tables) > 0, "shared": shared, "rows": rows, "levels": levels}
}

// jsFixed is JavaScript's toFixed: the exact value rounded half up, a
// negative value as its magnitude after a minus sign.
func jsFixed(value float64, digits int) string {
	if value < 0 {
		return "-" + jsFixed(-value, digits)
	}
	scaled := new(big.Float).SetPrec(256).SetFloat64(value)
	scaled.Mul(scaled, new(big.Float).SetPrec(256).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)))
	scaled.Add(scaled, new(big.Float).SetPrec(256).SetFloat64(0.5))
	whole, _ := scaled.Int(nil)
	text := whole.String()
	if digits == 0 {
		return text
	}
	for len(text) <= digits {
		text = "0" + text
	}
	return text[:len(text)-digits] + "." + text[len(text)-digits:]
}

// miaoComma is miao's Format.comma: toFixed decimals with thousands commas.
func miaoComma(value float64, digits int) string {
	fixed, _ := strconv.ParseFloat(jsFixed(value, digits), 64)
	integer, decimal, _ := strings.Cut(strconv.FormatFloat(fixed, 'f', -1, 64), ".")
	for index := len(integer) - 3; index > 0; index -= 3 {
		integer = integer[:index] + "," + integer[index:]
	}
	if digits > 0 {
		if decimal == "" {
			decimal = strings.Repeat("0", digits)
		}
		return integer + "." + decimal
	}
	return integer
}

var (
	miaoTag   = regexp.MustCompile(`<[^>]*>`)
	miaoColor = regexp.MustCompile(`^<span style="color:\s*(#[0-9A-Fa-f]{3,8});?">$`)
)

// miaoRichText is a line of miao's description HTML, which upstream prints
// as is, keeping only its headings, line breaks, italics, bold, plain spans,
// no-break runs and colours: everything else is escaped or dropped.
func miaoRichText(text string) string {
	var out strings.Builder
	last := 0
	for _, bounds := range miaoTag.FindAllStringIndex(text, -1) {
		out.WriteString(html.EscapeString(text[last:bounds[0]]))
		last = bounds[1]
		tag := text[bounds[0]:bounds[1]]
		switch lower := strings.ToLower(tag); lower {
		case "<br>", "<br/>", "<br />":
			out.WriteString("<br>")
		case "<h3>", "</h3>", "<i>", "</i>", "<b>", "</b>", "<span>", "</span>", "<nobr>", "</nobr>":
			out.WriteString(lower)
		default:
			if match := miaoColor.FindStringSubmatch(tag); match != nil {
				out.WriteString(`<span style="color:` + match[1] + `">`)
			}
		}
	}
	out.WriteString(html.EscapeString(text[last:]))
	return out.String()
}
