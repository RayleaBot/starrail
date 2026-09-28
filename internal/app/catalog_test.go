package app

import "testing"

func TestResolvePrefersTheExactName(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[
		{"id":"1002","name":"丹恒","kind":"character"},
		{"id":"1213","name":"丹恒•饮月","aliases":["饮月"],"kind":"character"},
		{"id":"8006","name":"星·同谐","kind":"character"},
		{"id":"8010","name":"星·同谐","kind":"character"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string]string{"丹恒": "1002", "饮月": "1213", "丹恒•": "1213", "8010": "8010", "龙尊": "1213"} {
		aliases := map[string]string{"龙尊": "1213"}
		entry, ok := catalog.Resolve(query, "character", aliases)
		if !ok || entry.ID != want {
			t.Fatalf("%s resolved to %q %v, want %s", query, entry.ID, ok, want)
		}
	}
	// Two entries share the name exactly; only the ID can pick one.
	if _, ok := catalog.Resolve("星·同谐", "character", nil); ok {
		t.Fatal("an ambiguous name resolved")
	}
}

func TestEntryViewListsStatsSkillsAndEidolons(t *testing.T) {
	entry := Entry{ID: "1006", Name: "银狼", Kind: "character", Rarity: 5, Element: "量子", Weapon: "虚无",
		Stats:   map[string]any{"hp": 484.7, "atk": 296.2, "cpct": 5.0, "level": 80.0},
		Talents: []Talent{{Name: "系统警告", Type: "普攻", Tag: "单攻", Level: 6, Description: []string{"造成 100% 伤害。"}}, {Name: "强制结束进程", Type: "秘技", Description: []string{"立即攻击。"}}},
		Cons:    []Talent{{Name: "社会工程", Description: []string{"恢复能量。"}}}}
	view := EntryView(Game{Name: "星铁"}, entry)
	titles := []string{}
	for _, section := range view.Sections {
		titles = append(titles, section.Title)
	}
	want := []string{"基础属性（80 级）", "普攻 · 系统警告（单攻） · 6 级数值", "秘技 · 强制结束进程", "星魂 1 · 社会工程"}
	if len(titles) != len(want) {
		t.Fatalf("sections %v, want %v", titles, want)
	}
	for i := range want {
		if titles[i] != want[i] {
			t.Fatalf("section %d is %q, want %q", i, titles[i], want[i])
		}
	}
	if rows := view.Sections[0].Rows; len(rows) != 3 || rows[0] != (Row{Label: "生命值", Value: "484.7"}) || rows[2] != (Row{Label: "暴击率", Value: "5%"}) {
		t.Fatalf("stat rows %v", rows)
	}
	if view.Rows[2] != (Row{Label: "命途", Value: "虚无"}) {
		t.Fatalf("rows %v", view.Rows)
	}
}

func TestSummariesLeaveOutTheTexts(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[
		{"id":"1006","name":"银狼","aliases":["小狼"],"abbr":"狼","kind":"character","rarity":5,"element":"量子","weapon":"虚无","description":"骇客","talents":[{"name":"系统警告","description":["伤害"]}]},
		{"id":"21000","name":"一场术后对话","kind":"weapon","rarity":4,"weapon":"丰饶"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	list := catalog.Summaries()
	if len(list) != 2 {
		t.Fatalf("summaries %v", list)
	}
	first := list[0]
	if first.ID != "1006" || first.Name != "银狼" || first.Abbr != "狼" || first.Weapon != "虚无" || len(first.Aliases) != 1 || first.Rarity != 5 || list[1].Weapon != "丰饶" {
		t.Fatalf("summaries %+v", list)
	}
}
