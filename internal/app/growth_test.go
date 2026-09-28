package app

import "testing"

func TestGrowthParsesOfficialCalculatorShapes(t *testing.T) {
	for _, tc := range []struct {
		game           string
		detail         string
		expectedSkills int
	}{{"starrail", `{"avatar":{"item_name":"测试","cur_level":50,"max_level":80},"equipment":{"item_id":"123","item_name":"测试光锥","cur_level":40,"max_level":80},"skills":[{"point_id":"101","cur_level":3,"max_level":10},{"point_id":"102","cur_level":0,"max_level":1}]}`, 2}} {
		p, err := prepareGrowth("1001", decoded(t, tc.detail), Catalog{})
		if err != nil || p.Current != 50 || p.Target != 50 || p.WeaponCurrent != 40 || len(p.Skills) != tc.expectedSkills {
			t.Fatal(p, err)
		}
	}
	if _, err := prepareGrowth("1001", decoded(t, `{"avatar":{"name":"test"}}`), Catalog{}); err == nil {
		t.Fatal("missing current level invented")
	}
}
func TestGrowthMaterialAggregationDoesNotDoubleCountItemBreakdown(t *testing.T) {
	data := decoded(t, `{"avatar_consume":[{"item_id":"1","item_name":"信用点","num":100}],"skill_consume":[{"item_id":"1","item_name":"信用点","num":20}],"coin_id":"1"}`)
	v := growthView(Game{ID: "starrail"}, GrowthPlan{}, QueryResult{Data: data})
	if len(v.Rows) != 1 || v.Rows[0].Value != "120" {
		t.Fatal(v)
	}
}

func TestGrowthNamesCalculatorSkillsFromTheCatalog(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[{"id":"1006","name":"银狼","kind":"character","talents":[{"name":"系统警告","type":"普攻","description":[]},{"name":"是否允许更改？","type":"战技","description":[]},{"name":"账号已封禁","type":"终结技","description":[]},{"name":"等待程序响应…","type":"天赋","description":[]}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	detail := decoded(t, `{"avatar":{"item_name":"银狼","cur_level":80,"max_level":80},"skills":[
		{"point_id":"1006002","cur_level":10,"max_level":10,"item_url":"https://example/SkillIcon_1006_BP.png"},
		{"point_id":"1006004","cur_level":10,"max_level":10,"item_url":"https://example/1006004.png"},
		{"point_id":"1006102","cur_level":1,"max_level":1},
		{"point_id":"1006203","cur_level":1,"max_level":1,"point_type":1},
		{"point_id":"1006420","cur_level":1,"max_level":1,"point_type":3}]}`)
	plan, err := prepareGrowth("1006", detail, catalog)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"是否允许更改？", "等待程序响应…", "额外能力 2", "属性加成 3", "额外能力 1006420"}
	for i, skill := range plan.Skills {
		if skill.Name != want[i] {
			t.Fatalf("skill %d named %q, want %q", i, skill.Name, want[i])
		}
	}
}
