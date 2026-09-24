package app

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func decoded(t *testing.T, source string) map[string]any {
	t.Helper()
	var result map[string]any
	decoder := json.NewDecoder(strings.NewReader(source))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestMonthlyAndChallengeViewsUseBusinessFields(t *testing.T) {
	for _, tc := range []struct {
		game, op, data string
		want           []string
	}{
		{"starrail", "currency_wars", `{"grid_fight_brief":{"weekly_score_cur":"80","weekly_score_max":"100","division":{"name":"黄金"}},"grid_fight_archive_list":[{"brief":{"remain_hp":"42","archive_rank":"1"},"lineup":{"front_roles":[{"name":"测试角色"}]}}]}`, []string{"每周积分：80 / 100", "名称：黄金", "前排：测试角色", "剩余生命：42"}},
		{"starrail", "peak", `{"challenge_peak_best_record_brief":{"boss_stars":3,"mob_stars":9},"challenge_peak_records":[{"boss_record":{"round_num":2,"avatars":[{"name":"测试角色"}]}}]}`, []string{"首领星数：3", "关卡星数：9", "轮数：2", "队伍：测试角色"}},
	} {
		t.Run(tc.game+tc.op, func(t *testing.T) {
			view := BusinessView(Game{ID: tc.game}, Operation{Name: tc.game + "." + tc.op, Label: "测试"}, QueryResult{Data: decoded(t, tc.data)}, Catalog{})
			for _, text := range tc.want {
				if !strings.Contains(view.Text(), text) {
					t.Fatalf("missing business output %q", text)
				}
			}
			if strings.Contains(view.Text(), "avatar_list") || strings.Contains(view.Text(), "grid_fight_brief") {
				t.Fatal("raw protocol key shown to user")
			}
		})
	}
}
func TestGachaSummaryNeverClaimsCompleteHistory(t *testing.T) {
	view := BusinessView(Game{ID: "starrail"}, Operation{Name: "starrail.gacha_summary"}, QueryResult{Data: decoded(t, `{"gacha_type":"11","pool":{"cards":[{"name":"池一","total_count":90,"up_count":1}]},"five_star":{"list":[{"gacha_count":10,"item":null},{"gacha_count":80,"item":{"name":"测试五星"}}],"has_more":true}}`)}, Catalog{})
	for _, text := range []string{"总抽数 90", "当前未出五星抽数：10", "测试五星：80 抽", "不是完整逐抽历史", "首批"} {
		if !strings.Contains(view.Text(), text) {
			t.Fatalf("missing boundary %q", text)
		}
	}
}
func TestExtendedCommandInputs(t *testing.T) {
	app := &App{}
	for _, tc := range []struct{ mode, valid, invalid, key string }{{"year_month", "202609", "202613", "month"}, {"starrail_pool", "21", "100", "gacha_type"}} {
		input, uid, err := app.commandInput(Operation{Input: tc.mode}, []string{tc.valid, "100000001"}, nil)
		if err != nil || uid != "100000001" || input[tc.key] == nil {
			t.Fatal("valid selector failed")
		}
		if _, _, err := app.commandInput(Operation{Input: tc.mode}, []string{tc.invalid}, nil); err == nil {
			t.Fatal("invalid selector accepted")
		}
	}
	// 面板, 遗器, 伤害 and 养成 read their character by name, then the UID.
	if input, uid, err := pluginApp(t).commandInput(Operation{Input: "characters"}, []string{"希儿", "100000001"}, nil); err != nil || uid != "100000001" || !reflect.DeepEqual(input["character_ids"], []any{"1102"}) {
		t.Errorf("a character = %v %q %v", input, uid, err)
	}
	// A UID alone, as in 月历100000001, is the UID.
	if input, uid, err := app.commandInput(Operation{Input: "year_month"}, []string{"100000001"}, nil); err != nil || uid != "100000001" || input["month"] != nil {
		t.Errorf("a UID alone = %v %q %v", input, uid, err)
	}
}
