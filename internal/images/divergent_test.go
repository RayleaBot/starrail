package images

import (
	"encoding/json"
	"strings"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
)

func decodeRecord(t *testing.T, text string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestDivergentFollowsStarRailPlugin(t *testing.T) {
	result := func(weekly string) gamekit.QueryResult {
		return gamekit.QueryResult{Role: gamekit.Role{UID: "100000001"}, Data: decodeRecord(t, `{"basic":{"season_level":20,"weekly_score":1200,"weekly_score_max":3000,
			"possibility_gallery_finished":3,"possibility_gallery_total":9,"season_task_finished":4,"season_task_total":10,"finished_weekly":true,
			"normal_record_brief":{"title":"星阶","common_info_v2":{"level":5,"exist_data":true}},"weekly_record_brief":`+weekly+`},
			"normal_detail":{"records":[{"finish_time":{"year":2026,"month":9,"day":5,"hour":8,"minute":3,"second":9},"common_info_v2":{"level":4}}]}}`)}
	}
	// Upstream names the weekly difficulty by its ID and keeps the official
	// title only when there is no ID.
	for weekly, want := range map[string]string{
		`{"title":"官方","common_info_v2":{"id":3}}`: "III",
		`{"title":"官方","common_info_v2":{"id":0}}`: "官方",
		`{"title":"官方","common_info_v2":{"id":9}}`: "",
	} {
		image, ok := Divergent(gamekit.ImageContext{}, result(weekly))
		if !ok || image.Data["weekly_title"] != want {
			t.Errorf("%s: weekly_title = %v, want %q", weekly, image.Data["weekly_title"], want)
		}
	}
	image, _ := Divergent(gamekit.ImageContext{}, result(`{}`))
	latest := image.Data["latest"].(map[string]any)
	if latest["time"] != "2026.09.05 08:03:09" || latest["level"] != "4" || image.Data["gallery"] != "3/9" || image.Data["weekly_finished"] != true {
		t.Errorf("image = %v", image.Data)
	}
}

func TestDivergentRunsFollowTheCommandWord(t *testing.T) {
	data := decodeRecord(t, `{"basic":{},
		"normal_detail":{"records":[{"common_info_v2":{"level":1}},{"common_info_v2":{"level":2}}]},
		"cur_week_detail":{"weekly_name":"本周","weekly_buff_desc":["●效果<br>换行"],"records":[{"common_info_v2":{"level":3}}]},
		"last_week_detail":{"weekly_name":"上周","records":[{"common_info_v2":{"level":4},
			"buffs":[{"base_type":{"name":"「巡猎」"},"items":[{},{}]},{"base_type":{"name":"「毁灭」"},"items":[{}]}]}]}}`)
	result := gamekit.QueryResult{Data: data}
	image, ok := DivergentNormal(gamekit.ImageContext{Word: "常规演算二"}, result)
	if !ok || image.Data["level"] != "2" || image.Data["run"] != "2 / 2" || image.Data["weekly"] != nil {
		t.Errorf("常规演算二 = %v", image.Data)
	}
	// Upstream answers in text when the run does not exist.
	if _, ok := DivergentNormal(gamekit.ImageContext{Word: "常规演算三"}, result); ok {
		t.Error("常规演算三 drew a missing run")
	}
	image, _ = DivergentWeek(gamekit.ImageContext{Word: "本期演算"}, result)
	weekly := image.Data["weekly"].(map[string]any)
	if image.Data["title"] != "差分宇宙·周期演算 (本期)" || weekly["effects"].([]any)[0] != "效果<br>换行" {
		t.Errorf("本期演算 = %v", image.Data)
	}
	image, _ = DivergentWeek(gamekit.ImageContext{Word: "上周差分记录"}, result)
	if image.Data["title"] != "差分宇宙·周期演算 (上期)" || image.Data["level"] != "4" || image.Data["buff_count"] != 3 {
		t.Errorf("上周差分记录 = %v", image.Data)
	}
}

func TestOfficialRichTextKeepsOnlyFormatting(t *testing.T) {
	for text, want := range map[string]string{
		`<color=#f29e38ff>暴击</color>提高<br/>10%`:                       `<span style="color: #f29e38ff">暴击</span>提高<br>10%`,
		`<span style="color: #FFD780;">伤害</span>`:                     `<span style="color: #FFD780">伤害</span>`,
		`<img src=x onerror=alert(1)><script>x</script>a&b <unbreak>`: `xa&amp;b `,
		`<span style="color: red" onclick="x">文字</span>`:              `文字</span>`,
	} {
		if got := officialRichText(text); got != want {
			t.Errorf("officialRichText(%q) = %q, want %q", text, got, want)
		}
	}
}
