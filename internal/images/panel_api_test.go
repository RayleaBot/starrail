package images_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/artwork"
	"github.com/RayleaBot/starrail/internal/images"
)

// The panel writes numbers as new_panel.html does: totals floored, bases
// and percentages floored at one decimal through their decimal text
// (MathPro.floor10), scores with toFixed, relic stats from 1 up as whole
// numbers.
func TestAPIPanelWritesNumbersAsStarRailPlugin(t *testing.T) {
	critical := 1234.9
	image := app.APIPanelImage{UID: "100000001", API: "avocado.wiki", Character: app.APICharacter{AvatarID: 1102, Name: "希儿", Level: 80, Rank: 2, DamageType: "Quantum",
		Properties: map[string]float64{"hpBase": 931.392, "hpDelta": 2.3, "criticalChance": 0.673, "spRatio": 0.194, "quantumAdded": 0.388},
		Equipment:  app.APIEquipment{ID: 23001, Name: "于夜色中", Level: 80, Rank: 1, Rarity: "CombatPowerLightconeRarity5", HP: 1058.4},
		Relics: []app.APIRelic{{Name: "快枪手的野穗毡帽", Level: 15, MaxLevel: 15, MainValue: 0.432, Score: 21.25,
			Affixes: []app.APIAffix{{Name: "速度", Weight: 1, Value: 4.6, Count: 3}, {Name: "暴击率", Weight: 0.5, Value: 0.0648, Count: 1}, {Name: "效果命中", Value: 0.03}}}}},
		Damages: []app.APIDamage{{Title: "普攻"}}}
	image.Damages[0].Value.Critical = &critical
	drawn, ok := images.APIPanel(app.ImageContext{}, image)
	if !ok {
		t.Fatal("panel")
	}
	attributes := drawn.Data["attributes"].([]map[string]any)
	// 2.3 is 22.999… tenths in binary; upstream's decimal shift keeps 2.3.
	hp := attributes[0]
	if hp["value"] != "933" || hp["base"] != "931.3" || hp["delta"] != "2.3" {
		t.Errorf("hp = %v", hp)
	}
	if attributes[4]["value"] != "67.3%" || attributes[9]["value"] != "119.4%" || attributes[10]["name"] != "属性伤害提高" || attributes[10]["value"] != "38.8%" {
		t.Errorf("attributes = %v", attributes)
	}
	if drawn.Data["equipment"].(map[string]any)["hp"] != "1058" {
		t.Errorf("equipment = %v", drawn.Data["equipment"])
	}
	relic := drawn.Data["relics"].([]any)[0].(map[string]any)
	if relic["rank"] != "5" || relic["score"] != "21.3" || relic["main_value"] != "43.2%" {
		t.Errorf("relic = %v", relic)
	}
	var affixes [][3]any
	for _, raw := range relic["affixes"].([]any) {
		affix := raw.(map[string]any)
		affixes = append(affixes, [3]any{affix["weight"], affix["rolls"], affix["value"]})
	}
	if !reflect.DeepEqual(affixes, [][3]any{{"weight-heavy", ">>", "4"}, {"weight-normal", "", "6.4%"}, {"", "", "3%"}}) {
		t.Errorf("affixes = %v", affixes)
	}
	damage := drawn.Data["damages"].([]any)[0].(map[string]any)
	if drawn.Data["damaged"] != true || damage["critical"] != "1234" || damage["expect"] != "NaN" {
		t.Errorf("damage = %v", damage)
	}
	// Without the damage module's table the page says it has none.
	image.Damages = nil
	if drawn, _ = images.APIPanel(app.ImageContext{}, image); drawn.Data["damaged"] != false {
		t.Error("a character without damage showed the table")
	}
}

// The skill tree is drawn in the canvas's units, twice the background's
// size: a skill with its level, a locked minor trace, and none for a trace
// the layout does not place. Icons that cannot be loaded say so.
func TestAPIPanelDrawsTheSkillTree(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"resources/panel/data/character.json":            `{"1102":{"path":"巡猎"}}`,
		"resources/panel/data/skillTree.json":            `{"巡猎":{"Point01":[10,20],"Point09":[30,40]}}`,
		"resources/panel/resources/skill_tree/Rogue.svg": `<svg width="100" height="50" viewBox="0 0 100 50" xmlns="http://www.w3.org/2000/svg"></svg>`,
		"resources/panel/resources/skill_tree/lock.png":  "png",
	} {
		path := filepath.Join(root, "starrail-plugin", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	character := app.APICharacter{AvatarID: 1102, Behaviors: []app.APIBehavior{
		{Anchor: "Point01", Level: 6, MaxLevel: 6, Icon: "/icon/skill/1102_basic_atk.png"}, {Anchor: "Point09"}, {Anchor: "Point19", Level: 1}}}
	drawn, _ := images.APIPanel(app.ImageContext{Artwork: &artwork.Store{Root: root}}, app.APIPanelImage{Character: character})
	tree := drawn.Data["tree"].(map[string]any)
	nodes := tree["nodes"].([]any)
	if drawn.Data["path"] != "巡猎" || tree["width"] != 200.0 || tree["height"] != 100.0 || tree["lock"] == "" || len(nodes) != 2 {
		t.Fatalf("tree = %v", tree)
	}
	skill, minor := nodes[0].(map[string]any), nodes[1].(map[string]any)
	if skill["level"] != "6/6" || skill["cx"] != 79.0 || skill["text_y"] != 218.0 || skill["locked"] != false || skill["icon"] != "" || skill["squeeze"] != false {
		t.Errorf("skill = %v", skill)
	}
	if minor["level"] != nil || minor["locked"] != true || minor["r"] != 51.0 || minor["lock_x"] != 112.0 || minor["squeeze"] != true {
		t.Errorf("minor trace = %v", minor)
	}
}

// A relic draws the icon upstream's relic data names, and miao's picture of
// the piece for a set upstream's data lacks, by the relic's set and slot.
func TestAPIPanelRelicsFallBackToMiao(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"starrail-plugin/resources/panel/data/relics.json":          "{\"61011\":{\"icon\":\"101_0.png\"}}",
		"starrail-plugin/resources/panel/resources/relic/101_0.png": "png",
		"miao-plugin/resources/meta-sr/artifact/谧宁拾骨地/arti-5.webp":  "webp",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	context := app.ImageContext{Artwork: &artwork.Store{Root: root}, Catalog: app.Catalog{ArtifactSets: map[string]string{"61011": "云无留迹的过客", "63195": "谧宁拾骨地", "61291": "闪耀功勋的魔法少女"}}}
	drawn, _ := images.APIPanel(context, app.APIPanelImage{Character: app.APICharacter{AvatarID: 1413, Relics: []app.APIRelic{{ID: 61011}, {ID: 63195}, {ID: 61291}}}})
	paths := map[string]string{"": ""}
	for _, resource := range drawn.Resources {
		paths[resource.ID] = filepath.ToSlash(resource.Path)
	}
	relics := drawn.Data["relics"].([]any)
	got := []string{}
	for _, relic := range relics {
		got = append(got, paths[relic.(map[string]any)["icon"].(string)])
	}
	want := []string{"assets/starrail-plugin/resources/panel/resources/relic/101_0.png", "assets/miao-plugin/resources/meta-sr/artifact/谧宁拾骨地/arti-5.webp", ""}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("relic icons = %v, want %v", got, want)
	}
}
