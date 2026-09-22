package app

import (
	"strings"
	"testing"
)

func TestPanelsUseMetadataAndKeepOfficialValues(t *testing.T) {
	for _, tc := range []struct {
		game, data string
		want       []string
	}{
		{"starrail", `{"property_info":{"5":{"name":"暴击率"}},"avatar_list":[{"id":1001,"name":"测试角色","level":80,"rank":1,"properties":[{"property_type":5,"final":"76.4%"}],"equip":{"name":"测试光锥","level":80,"rank":2},"ornaments":[{"id":1,"pos":5,"name":"测试球","level":15,"rarity":5,"main_property":{"property_type":12,"value":"38.8%"},"properties":[]}],"skills":[{"point_id":"12345","level":6,"is_activated":true,"skill_stages":[{"name":"战技","desc":"说明"}]}]}]}`, []string{"暴击率：76.4%", "测试光锥", "5号位 · 测试球", "战技：等级 6"}},
	} {
		t.Run(tc.game, func(t *testing.T) {
			panels := NormalizePanels(QueryResult{Data: decoded(t, tc.data)}, Catalog{})
			if len(panels) != 1 || len(panels[0].Stats) == 0 {
				t.Fatal("panel fields lost")
			}
			text := PanelView(Game{ID: tc.game}, panels, "100000001").Text()
			for _, want := range tc.want {
				if !strings.Contains(text, want) {
					t.Fatalf("missing %q", want)
				}
			}
			if len(panels[0].Skills) > 0 && strings.Contains(panels[0].Skills[0].Description, "<b>") {
				t.Fatal("raw markup retained")
			}
		})
	}
}
func TestMissingPropertiesAreNotInvented(t *testing.T) {
	panels := NormalizePanels(QueryResult{Data: decoded(t, `{"avatar_list":[{"id":1001,"name":"空面板","properties":[{"property_type":5}]}]}`)}, Catalog{})
	if len(panels[0].Stats) != 0 || len(panels[0].Equipment) != 0 || panels[0].Weapon != nil {
		t.Fatal("missing panel data became zero-valued stats or equipment")
	}
	if queryOperationName("starrail.character", nil) != "starrail.characters" {
		t.Fatal("empty starrail selector fetched all detailed panels")
	}
}
