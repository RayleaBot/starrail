package app

import "testing"

func TestGrowthParsesOfficialCalculatorShapes(t *testing.T) {
	for _, tc := range []struct {
		game           string
		detail         string
		expectedSkills int
	}{{"starrail", `{"avatar":{"item_name":"测试","cur_level":50,"max_level":80},"equipment":{"item_id":"123","item_name":"测试光锥","cur_level":40,"max_level":80},"skills":[{"point_id":"101","cur_level":3,"max_level":10},{"point_id":"102","cur_level":0,"max_level":1}]}`, 2}} {
		p, err := prepareGrowth("1001", decoded(t, tc.detail))
		if err != nil || p.Current != 50 || p.Target != 50 || p.WeaponCurrent != 40 || len(p.Skills) != tc.expectedSkills {
			t.Fatal(p, err)
		}
	}
	if _, err := prepareGrowth("1001", decoded(t, `{"avatar":{"name":"test"}}`)); err == nil {
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
