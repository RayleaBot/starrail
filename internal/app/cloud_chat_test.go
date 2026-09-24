package app

import (
	"math"
	"strings"
	"testing"
)

// rankSpecificAnswer is ark's answer to rank/specific with percent 0: the
// relic distribution, then the damage one.
const rankSpecificAnswer = `[{"retcode":100,"data":{"scores":["295.20","275.90","264.90","250.80","240.20","218.90","187.70","109.90","70.40","25.20"],"total":10994,"name":"终结技伤害","top1":"338.20"}},
	{"retcode":100,"data":{"scores":["95589.83","77644.48","70002.64","62100.72","56369.78","47075.55","30730.31","9460.62","4672.43","996.34"],"total":11471,"name":"终结技伤害","top1":"116762.08"}}]`

// 排名统计 reads ark's two distributions as the panel page reads them, with
// no panel placed on them.
func TestReadRankStatsFollowsPanelPage(t *testing.T) {
	entry := Entry{ID: "1102", Name: "希儿", Element: "量子"}
	image, view, err := readRankStats(entry, cloudList(t, rankSpecificAnswer))
	if err != nil || image.DamageTitle != "终结技伤害" || image.Totals != [2]string{"11471", "10994"} {
		t.Fatalf("image = %+v %v", image, err)
	}
	// Without an image the reply lists both distributions.
	for _, line := range []string{"希儿排名统计", "排名趋势 · 终结技伤害", "伤害排名\n统计样本：11471 个UID\nTOP 1%：95589.83", "遗器排名\n统计样本：10994 个UID\nTOP 1%：295.2", "TOP 99%：25.2"} {
		if !strings.Contains(view.Text(), line) {
			t.Fatalf("text lacks %q:\n%s", line, view.Text())
		}
	}
	if damage := image.Damage; damage.Top != 100 || math.Abs(damage.Scores[0]-81.87) > 0.01 || damage.Percent != -100 {
		t.Fatalf("damage = %+v", damage)
	}
	if artis := image.Artis; artis.Top != 338.2 || artis.Scores[9] != 25.2 || artis.Percent != -100 {
		t.Fatalf("artis = %+v", artis)
	}
	// A distribution ark did not send reads 暂无数据.
	answer := cloudList(t, rankSpecificAnswer).([]any)
	if _, view, err := readRankStats(entry, []any{cloudObject(t, `{"retcode":102}`), answer[1]}); err != nil || !strings.Contains(view.Text(), "遗器排名\n暂无数据") {
		t.Fatalf("text = %q %v", view.Text(), err)
	}
	// ark answers a character it does not rank with a failed object, and
	// nothing else can be drawn.
	for raw, code := range map[string]string{`{"retcode":102}`: "plugin.game_cloud_rejected", `{"retcode":100,"data":{"scores":[1,2,3,4,5,6,7,8,9,10]}}`: "plugin.game_cloud_invalid", `[{"retcode":102},{"retcode":102}]`: "plugin.game_cloud_invalid"} {
		if _, _, err := readRankStats(entry, cloudList(t, raw)); PublicError(err).Code != code {
			t.Errorf("%s: %v", raw, err)
		}
	}
}

func TestArkUsageTextFollowsArk(t *testing.T) {
	// Anonymous quota: no permission or multiplier, no advanced quota.
	anonymous := cloudObject(t, `{"retcode":0,"data":{"auth":{"mode":"ip","ip":"203.0.113.9"},"quota":{"rank":{"minute":{"limit":300,"remaining":300},"hour":{"limit":3000,"remaining":2999},"day":{"limit":10000,"remaining":9988}},"custom":{"normal":{"limit":60,"remaining":56},"advanced":null}}}}`)
	want := "权限类型：普通\n全部请求剩余额度：300/2999/9988\n自定义排名请求额度倍率：1x\n自定义排名普通请求剩余额度：56\n自定义排名高级请求剩余额度：-"
	if got := arkUsageText(anonymous); got != want {
		t.Fatalf("anonymous usage:\n%s", got)
	}
	token := cloudObject(t, `{"retcode":0,"data":{"auth":{"permission":1,"limit_normal":2},"quota":{"rank":{"minute":{"remaining":10}},"custom":{"normal":{"remaining":100},"advanced":{"remaining":20}}}}}`)
	want = "权限类型：高级\n全部请求剩余额度：10/-/-\n自定义排名请求额度倍率：2x\n自定义排名普通请求剩余额度：100\n自定义排名高级请求剩余额度：20"
	if got := arkUsageText(token); got != want {
		t.Fatalf("token usage:\n%s", got)
	}
	for raw, want := range map[string]string{`{"retcode":401,"message":"invalid token"}`: "invalid token", `{"retcode":0}`: "查询失败"} {
		if got := arkUsageText(cloudObject(t, raw)); got != want {
			t.Errorf("%s: %q", raw, got)
		}
	}
}

func TestArkPlayerPanelsReadArkPlayerData(t *testing.T) {
	a := pluginApp(t)
	player := cloudObject(t, `{"uid":"100000001","avatars":{"1102":{"id":1102,"elem":"quantum","level":80,"promote":6,"cons":0,"talent":{"a":6,"e":10,"t":10,"q":10},"trees":["1102101"],
		"weapon":{"id":23001,"level":80,"promote":6,"affix":1},"artis":{"1":{"level":15,"star":5,"id":61011,"mainId":1,"attrIds":["5,2,2"]}}}}}`)
	panels := a.arkPlayerPanels(t.Context(), "100000001", player)
	if len(panels) != 1 || panels[0].ID != "1102" || panels[0].Source != "share" || panels[0].Weapon == nil || panels[0].Weapon.Name != "于夜色中" {
		t.Fatalf("panels = %+v", panels)
	}
	// Player data of another UID is not kept.
	if panels := a.arkPlayerPanels(t.Context(), "100000002", player); len(panels) != 0 {
		t.Fatalf("kept another UID's panels: %+v", panels)
	}
}

func TestPanelDataRefusalFollowsArkPermission(t *testing.T) {
	const account, master = "为确保数据安全，目前仅允许绑定CK用户导入/导出自己UID的面板数据，请联系Bot主人导入/导出...", "为确保数据安全，目前仅允许主人导入/导出自己UID的面板数据，请联系Bot主人导入/导出..."
	cases := []struct {
		level              int
		account, superUser bool
		want               string
	}{
		{0, false, false, ""},
		{1, true, false, ""}, {1, false, true, ""}, {1, false, false, account},
		{2, true, false, master}, {2, false, true, ""},
		{3, true, true, "当前功能已被禁用..."},
	}
	for _, tc := range cases {
		if got := panelDataRefusal(tc.level, tc.account, tc.superUser); got != tc.want {
			t.Errorf("level %d account %v super %v: %q", tc.level, tc.account, tc.superUser, got)
		}
	}
	if hasAccount(Accounts{Items: []Account{{Roles: nil}}}) || !hasAccount(Accounts{Items: []Account{{}, {Roles: []Role{{UID: "100000001"}}}}}) {
		t.Fatal("account")
	}
}
