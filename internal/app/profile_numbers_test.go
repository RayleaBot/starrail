package app

import "testing"

func TestProfileNumbersRestoreRelicPrecisionAndLightCone(t *testing.T) {
	g := testGame(t)
	panel := CharacterPanel{ID: "1302", Element: "物理", Level: 80, Source: "mihoyo", RankKnown: true, EquipmentKnown: true,
		Weapon: &PanelEquipment{ID: "23018", Level: 80, Refinement: 1},
		Stats: []PanelStat{{ID: "2", Value: "1319", Base: "1319"}, {ID: "4", Value: "108", Base: "103"},
			{ID: "5", Value: "14.0%", Base: "14.0%", Added: "0.0%"}},
		Equipment: []PanelEquipment{{Slot: 1, Level: 15, SetName: "幽锁深牢的系囚", Complete: true,
			Main: []PanelStat{{ID: "27", Key: "hpPlus", Value: "705"}},
			Sub:  []PanelStat{{ID: "51", Key: "speed", Value: "5"}, {ID: "52", Key: "cpct", Value: "9.0%"}}}},
		Relics: &MiaoRelics{Complete: true, Pieces: map[string]map[string]any{"1": {
			"star": 5, "level": 15, "mainId": "1", "attrIds": []any{"7,2,3", "8,3,4"},
		}}},
	}
	record, err := findReferenceCharacter(g.Calc, panel)
	if err != nil {
		t.Fatal(err)
	}
	n := g.profileNumbers(t.Context(), panel, record)
	for key, want := range map[string]PanelStat{
		"atk":   {Value: "1,319.5", Base: "1,319.5", Added: "0.0"},
		"speed": {Value: "107.9", Base: "103.0", Added: "4.9"},
		"cpct":  {Value: "14.1%", Base: "5.0%", Added: "9.1%"},
		"joy":   {Value: "0.0%", Base: "0.0%", Added: "0.0%"},
	} {
		if got := n.Stats[key]; got != want {
			t.Errorf("%s = %+v, want %+v", key, got, want)
		}
	}
	if len(n.Weapon) != 3 || n.Weapon[0].Value != "1,058.4" || n.Weapon[1].Value != "582.1" || n.Weapon[2].Value != "463.0" {
		t.Errorf("light cone = %+v", n.Weapon)
	}

	// At level 20 the character's ascension is ambiguous without base stats.
	panel.Level, panel.Stats = 20, nil
	partial := g.profileNumbers(t.Context(), panel, record)
	if len(partial.Stats) != 0 || len(partial.Weapon) != 3 || partial.Weapon[1].Value != "582.1" {
		t.Errorf("partial panel = %+v", partial)
	}
}
