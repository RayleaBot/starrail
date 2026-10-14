package images

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/artwork"
)

func TestCharacterWikiUsesBaseStatsAndOrderedMaterials(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"character/真珠/data.json":  `{"name":"真珠","star":5,"elem":"冰","weapon":"欢愉","sp":180,"allegiance":"星际和平公司","cncv":"苏婉","jpcv":"安野希世乃","desc":"描述","baseAttr":{"hp":1203.05,"atk":465.7,"def":727.65,"speed":99,"aggro":100},"talent-materials":{"海妖残鳍":65,"梦现管锥":"48/64/51","《绒绒号》典藏版合集":"14/53/104","伪觉者的期许":9,"命运的足迹":6}}`,
		"material/data.json":      `{"1":{"name":"海妖残鳍","type":"boss","star":4},"2":{"name":"梦现管锥","type":"normal","star":3},"3":{"name":"《绒绒号》典藏版合集","type":"talent","star":4},"4":{"name":"伪觉者的期许","type":"weekly","star":4},"5":{"name":"命运的足迹","type":"virtual","star":5}}`,
		"material/boss/海妖残鳍.webp": "webp",
	}
	for name, content := range files {
		file := filepath.Join(root, "miao-plugin/resources/meta-sr", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	context := app.ImageContext{Artwork: &artwork.Store{Root: root}, Game: app.Game{Prefix: "*"}}
	image, ok := Entry(context, app.EntryImage{Command: "catalog", Entry: app.Entry{ID: "1503", Name: "真珠", Kind: "character"}})
	if !ok || image.Template != "character-wiki" {
		t.Fatal(image)
	}
	titles := []string{}
	for _, raw := range image.Data["attr"].([]any) {
		titles = append(titles, raw.(map[string]any)["title"].(string))
	}
	if !slices.Equal(titles, []string{"基础生命", "基础攻击", "基础防御", "速度", "嘲讽", "能量上限"}) {
		t.Fatal(titles)
	}
	if got := image.Data["attr"].([]any)[5].(map[string]any)["value"]; got != "180" {
		t.Fatal(got)
	}
	withoutEnergy, ok := characterWiki(context, wikiCharacter{}, app.Entry{Name: "真珠"})
	if !ok || len(withoutEnergy.Data["attr"].([]any)) != 5 {
		t.Fatal(withoutEnergy.Data)
	}
	materials := image.Data["materials"].([]any)
	names := []string{}
	for _, raw := range materials {
		names = append(names, raw.(map[string]any)["label"].(string))
	}
	if !slices.Equal(names, []string{"海妖残鳍", "梦现管锥", "《绒绒号》典藏版合集", "伪觉者的期许", "命运的足迹"}) || materials[1].(map[string]any)["num"] != "48/64/51" || materials[0].(map[string]any)["num"] != "65" || materials[0].(map[string]any)["icon"] == "" {
		t.Fatal(materials)
	}
	if _, ok := Entry(context, app.EntryImage{Command: "catalog", Entry: app.Entry{Kind: "weapon", Name: "真珠"}}); ok {
		t.Fatal("weapon drew character wiki")
	}
}

// The expected strings are what JavaScript's toFixed and miao's Format print.
func TestMiaoFormatsFollowJavaScript(t *testing.T) {
	for _, tc := range []struct {
		value  float64
		digits int
		want   string
	}{{106.25, 1, "106.3"}, {178.75, 1, "178.8"}, {1.005, 2, "1.00"}, {0.5, 0, "1"}, {931.392, 1, "931.4"}, {-100, 1, "-100.0"}, {-0.04, 1, "-0.0"}} {
		if got := app.JSFixed(tc.value, tc.digits); got != tc.want {
			t.Errorf("app.JSFixed(%v, %d) = %s, want %s", tc.value, tc.digits, got, tc.want)
		}
	}
	if got := miaoComma(115, 1); got != "115.0" {
		t.Errorf("miaoComma(115) = %s", got)
	}
	if got := miaoComma(12345.6, 0); got != "12,346" {
		t.Errorf("miaoComma(12345.6) = %s", got)
	}
}

func TestWikiFillFollowsMiaoGetDesc(t *testing.T) {
	tables := map[string]struct {
		Name   string
		IsSame bool
		Values []float64
	}{"1": {Name: "技能伤害", Values: []float64{1.1, 1.21, 1.32, 1.43, 1.54, 1.65, 1.7875, 1.925, 2.0625}}, "2": {Name: "回合数", Values: []float64{2, 2, 2, 2, 2, 2, 2, 2, 2}}}
	desc, out := wikiFill("造成<nobr>$1[i]%</nobr>攻击力的伤害，持续<nobr>$2[i]</nobr>回合。<br>第二行", tables, 8)
	if len(desc) != 2 || desc[0] != "造成<nobr>192.5% (lv8)</nobr>攻击力的伤害，持续<nobr>2 (lv8)</nobr>回合。" {
		t.Errorf("desc = %q", desc)
	}
	if out[0].values[6] != "178.8%" || out[1].values[0] != "2" {
		t.Errorf("tables = %v", out)
	}
}
