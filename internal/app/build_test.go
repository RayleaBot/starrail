package app

import (
	"testing"

	"github.com/RayleaBot/starrail/internal/reference"
)

func TestBuildPanelKeepsTotalTalentLevelsAndServantSkills(t *testing.T) {
	data := decoded(t, `{"avatar_list":[{"id":1001,"level":80,"rank":6,"element":"Fire","equip":null,"relics":[],"ornaments":[],"properties":[{"property_type":1,"base":"1000","final":"2000"},{"property_type":2,"base":"800","final":"1600"},{"property_type":3,"base":"500","final":"700"},{"property_type":5,"final":"50%"},{"property_type":6,"final":"100%"},{"property_type":14,"final":"40%"}],"skills":[{"point_id":100101,"remake":"普攻","level":7,"extra_level":1,"point_type":2,"is_activated":true},{"point_id":1001101,"point_type":3,"level":1,"is_activated":true}],"servant_detail":{"servant_skills":[{"remake":"忆灵技","level":11,"point_id":100109,"point_type":2,"is_activated":true}]}}]}`)
	panels := NormalizePanels(QueryResult{Data: data}, Catalog{})
	record := reference.Character{Data: map[string]any{"talent": map[string]any{"a": map[string]any{"name": "a"}, "me": map[string]any{"name": "me"}}}}
	profile, err := buildProfile(panels[0], record)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Talents["a"] != 7 || profile.Talents["me"] != 11 {
		t.Fatal("official total level was incremented or servant omitted")
	}
	if len(profile.Trees) != 1 || profile.Trees[0] != "1001101" {
		t.Fatal("trace selection changed")
	}
	if profile.Weapon.ID != "" || profile.Weapon.Level != 0 {
		t.Fatal("unequipped state invented a weapon")
	}
	panels[0].WeaponKnown = false
	if _, err = buildProfile(panels[0], record); err == nil {
		t.Fatal("unknown weapon state treated as unequipped")
	}
}
func TestEquipmentSetMappingUsesKnownCatalogEntries(t *testing.T) {
	data := decoded(t, `{"avatar_list":[{"id":1001,"relics":[{"id":61301,"pos":1,"name":"测试装备","main_property":{"property_type":32,"value":"100"},"properties":[]}]}]}`)
	panels := NormalizePanels(QueryResult{Data: data}, Catalog{ArtifactSets: map[string]string{"61301": "参考套装"}})
	if panels[0].Equipment[0].SetName != "参考套装" {
		t.Fatal("official equipment id did not resolve")
	}
	panels = NormalizePanels(QueryResult{Data: data}, Catalog{})
	if panels[0].Equipment[0].SetName != "" {
		t.Fatal("unknown set was guessed")
	}
}

// A relic below +3 previews the substat it will gain; miao's MysPanelHSR
// leaves it out, and so do the panel, its score and its damage.
func TestOfficialPanelsLeaveOutPreviewedSubstats(t *testing.T) {
	data := decoded(t, `{"avatar_list":[{"id":1001,"relics":[{"id":61301,"pos":1,"level":0,"rarity":5,"main_property":{"property_type":27,"value":"112","is_preview":false},
		"properties":[{"property_type":56,"times":1,"value":"4.3%","is_preview":false},{"property_type":31,"times":1,"value":"19","is_preview":true}]}]}]}`)
	panels := NormalizePanels(QueryResult{Data: data}, Catalog{})
	if subs := panels[0].Equipment[0].Sub; len(subs) != 1 || subs[0].ID != "56" {
		t.Fatalf("read = %+v", subs)
	}
}

func TestEnhancedRulesKeepOfficialQueryIdentity(t *testing.T) {
	panel := CharacterPanel{ID: "1005", Element: "Lightning", Name: "卡芙卡", Skills: []PanelSkill{{ID: "11005001"}}}
	if referenceCharacterID(panel) != "2005" {
		t.Fatal("enhanced skill prefix not selected")
	}
	record, err := findBuildCharacter(calcEngine(t), panel)
	if err != nil || record.ID != "2005" {
		t.Fatal("wrong reference rule", record.ID, err)
	}
	result := BuildResult{}
	applyBuildIdentity(&result, panel, record)
	if result.CharacterID != "1005" || result.Variant != "enhanced" {
		t.Fatal("reference identity replaced official API identity")
	}
	panel.Skills[0].ID = "1005001"
	if referenceCharacterID(panel) != "1005" {
		t.Fatal("ordinary skill became enhanced")
	}
	panel.ID = "9999"
	panel.Skills[0].ID = "19999001"
	if referenceCharacterID(panel) != "9999" {
		t.Fatal("unknown transformation guessed")
	}
}
