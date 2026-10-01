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
	answer := cloudList(t, rankSpecificAnswer).([]any)
	image, view, ok := readRankStats(entry, answer)
	if !ok || image.DamageTitle != "终结技伤害" || image.Totals != [2]string{"11471", "10994"} {
		t.Fatalf("image = %+v", image)
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
	if _, view, ok := readRankStats(entry, []any{cloudObject(t, `{"retcode":102}`), answer[1]}); !ok || !strings.Contains(view.Text(), "遗器排名\n暂无数据") {
		t.Fatalf("text = %q", view.Text())
	}
	// Nothing can be drawn when neither distribution reads.
	if _, _, ok := readRankStats(entry, cloudList(t, `[{"retcode":102},{"retcode":102}]`).([]any)); ok {
		t.Fatal("drew failed distributions")
	}
}

// 角色排名, 总排名 and the other user commands read ark's answer as
// ark-plugin does: retcode 100 is success, anything else is dealError's
// ERROR_MAP text or 未知错误, as is a failed request.
func TestArkUserRepliesFollowArk(t *testing.T) {
	failed := gameError("cloud_unavailable", "云服务暂不可用或请求已取消。")
	rank := `{"retcode":100,"rank":12,"score":98.765}`
	for _, tc := range []struct {
		answer string
		err    error
		want   string
	}{
		{rank, nil, "uid:100000001的希儿全服伤害排名为 12，伤害评分: 98.77"},
		// toFixed rounds an exact half up.
		{`{"retcode":100,"rank":3,"score":0.125}`, nil, "uid:100000001的希儿全服伤害排名为 3，伤害评分: 0.13"},
		{`{"retcode":102}`, nil, "未查询到角色信息"},
		{`{"retcode":201}`, nil, "请求超过速率限制，请5分钟后重试"},
		{`{"retcode":0}`, nil, "未知错误"},
		{rank, failed, "未知错误"},
	} {
		if got := characterRankReply("*", "100000001", "希儿", cloudObject(t, tc.answer), tc.err); got != tc.want {
			t.Errorf("%s %v: %q", tc.answer, tc.err, got)
		}
	}
	all := cloudObject(t, `{"retcode":100,"rank":[{"retcode":100,"rank":5,"score":88.1},{"retcode":102},{"retcode":100,"rank":40,"score":70}]}`)
	want := "uid:100000001的星铁全服排名数据:\n希儿全服伤害排名为5，伤害评分: 88.10\n符玄全服伤害排名为40，伤害评分: 70.00\n"
	if got := totalRankReply("*", "100000001", []string{"希儿", "银狼", "符玄"}, all, nil); got != want {
		t.Errorf("total rank:\n%s", got)
	}
	if got := totalRankReply("*", "100000001", nil, nil, failed); got != "未知错误" {
		t.Errorf("failed total rank: %q", got)
	}
	// ark's refusal of a UID nobody verified names this plugin's command.
	if got := arkDealError("*", cloudObject(t, `{"retcode":304}`), nil); got != "该uid未验证号主，请通过 *ark验证星铁uid 验证uid" {
		t.Errorf("304: %q", got)
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
	for raw, want := range map[string]string{`{"retcode":401,"message":"invalid token"}`: "云服务授权不可用，请联系管理员检查。", `{"retcode":0}`: "云服务请求失败，请稍后重试。"} {
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

// ark绑定星铁uid and ark验证星铁uid reply as ark-plugin does: the code and
// its Star Rail steps, 验证成功, the ERROR_MAP text of ark's retcode, and 未知错误
// for a retcode it does not map or a request that fails.
func TestArkVerifyReplyFollowsArk(t *testing.T) {
	code := arkVerifyReply("*", "verify/code", cloudObject(t, `{"retcode":100,"data":{"verifyCode":"ab12cd"}}`), nil)
	if want := "验证码: ab12cd\n使用方式：\n星铁：手机——右上角三点——漫游签证——设置签名——填入验证码，5-10分钟后输入 *ark验证星铁uid\n验证码有效期24小时，验证通过后自动与QQ绑定，在其他Bot上无需再次绑定"; code != want {
		t.Fatalf("code reply:\n%s", code)
	}
	cases := []struct {
		route, answer string
		err           error
		want          string
	}{
		{"verify", `{"retcode":100}`, nil, "验证成功"},
		{"verify", `{"retcode":302}`, nil, "验证失败，个人签名不匹配，请五分钟后重试"},
		{"verify", `{"retcode":304}`, nil, "该uid未验证号主，请通过 *ark验证星铁uid 验证uid"},
		{"verify/code", `{"retcode":-1}`, nil, "插件版本过低，请更新插件"},
		{"verify", `{"retcode":999}`, nil, "未知错误"},
		{"verify", `{"retcode":100}`, gameError("cloud_unavailable", "云服务暂不可用或请求已取消。"), "未知错误"},
	}
	for _, tc := range cases {
		if got := arkVerifyReply("*", tc.route, cloudObject(t, tc.answer), tc.err); got != tc.want {
			t.Errorf("%s %s: %q", tc.route, tc.answer, got)
		}
	}
}
