package app

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/pluginmeta"
)

// The shipped manifests follow upstream wording; these cases pin the words
// whose patterns overlap, so a reordered manifest cannot steal a command.
func TestShippedManifestsResolveUpstreamWording(t *testing.T) {
	cases := map[string][]struct {
		word, id string
		args     []string
	}{
		"starrail": {
			{"希儿面板", "api-panel", []string{"希儿"}},
			{"希儿面版", "character", []string{"希儿"}},
			{"面板", "api-panel-list", nil},
			{"面板列表", "api-panel-list", nil},
			{"角色面板", "panel-list", nil},
			{"角色", "profile", nil},
			{"卡片100000001", "profile", []string{"100000001"}},
			{"探索", "profile", nil},
			{"开启到期活动推送", "subscribe", []string{"到期"}},
			{"推送公告", "content-push", nil},
			{"星琼任务", "monthly-task", nil},
			{"更新面板", "panel-refresh", nil},
			{"全部面板更新", "panel-refresh", nil},
			{"面板更新", "api-panel-refresh", nil},
			{"面板更新100000001", "api-panel-refresh", []string{"100000001"}},
			{"mys更新面板", "panel-refresh-account", nil},
			{"遗器列表", "artifact-list", nil},
			{"希儿换满命换满行迹", "talent-wiki", []string{"希儿换满命换满"}},
			{"希儿换满命", "panel-change", nil},
			{"圣遗物列表100000001", "artifact-list", []string{"100000001"}},
			{"希儿遗器", "score", []string{"希儿"}},
			{"希儿伤害", "build", []string{"希儿"}},
			{"希儿伤害2", "build", []string{"希儿", "2"}},
			{"上期忘却之庭", "challenge", nil},
			{"上期模拟宇宙", "rogue", nil},
			{"差分宇宙", "divergent", nil},
			{"常规演算二", "divergent-normal", nil},
			{"上周差分宇宙记录", "divergent-week", nil},
			{"周期演算", "divergent-cycle", nil},
			{"货币", "currency_wars", nil},
			{"币战回顾", "currency-wars-archive", nil},
			{"货币战争记录十", "currency-wars-archive", nil},
			{"星琼统计", "monthly-history", nil},
			{"星琼", "ledger", nil},
			{"星琼七月", "ledger", []string{"七"}},
			{"星琼7", "ledger", []string{"7"}},
			{"原石", "monthly", nil},
			{"星琼记录", "pay-log", []string{"星琼"}},
			{"希儿行迹", "talent-wiki", []string{"希儿"}},
			{"ark希儿排行", "cloud-custom-rank", nil},
			{"ark符玄圣遗物排名", "cloud-custom-rank", nil},
			{"ark符玄遗器排行", "cloud-custom-rank", nil},
			{"ark自定义排行帮助", "cloud-custom-rank-help", nil},
			{"ark获取面板3", "cloud-custom-rank-panel", []string{"3"}},
			{"ARK获取面板20", "cloud-custom-rank-panel", []string{"20"}},
			{"ark获取面板", "cloud-custom-rank-panel", nil},
			{"arktoken用量", "cloud-usage", nil},
			{"ARKTOKEN用量", "cloud-usage", nil},
			// ark's getRank takes whatever follows 角色排名, spaced or not.
			{"角色排名希儿100000001", "cloud-character-rank", []string{"希儿100000001"}},
			{"角色排名", "cloud-character-rank", nil},
			{"ark绑定星铁uid", "cloud-verify-code", nil},
			{"ark验证星铁uid", "cloud-verify", nil},
			{"希儿遗器排名", "rank", nil},
			{"最强希儿", "rank-top", nil},
			{"今年星琼统计", "monthly-history", nil},
			{"十连光锥", "simulation", nil},
			{"更新跃迁记录", "gacha-background", nil},
			{"帮助", "help", nil},
			{"插件更新日志", "version", nil},
			{"树脂", "note", nil},
			{"刷新行迹", "talent-refresh", nil},
			{"更新图像", "artwork", nil},
			{"克拉拉养成75", "growth", []string{"克拉拉", "75"}},
			{"抽卡链接", "gacha-link", nil},
			{"希儿参考面板", "ref-panel", []string{"希儿"}},
			{"参考面板帮助", "ref-panel-help", nil},
			{"攻略", "path-guides", nil},
			{"希儿攻略", "guides", []string{"希儿"}},
			{"深渊攻略", "abyss-guide", nil},
			{"强度榜", "tier-list", nil},
			{"面板帮助", "panel-help", nil},
			{"预估", "estimate", nil},
			{"星琼盘点", "estimate", nil},
			{"公告2", "news", []string{"2"}},
			{"喵喵别名星铁设置", "alias-set", nil},
			{"设置希儿别名", "alias-add", []string{"希儿"}},
			{"删除别名鸭鸭", "alias-remove", []string{"鸭鸭"}},
			{"喵喵别名删除", "alias-remove", nil},
			{"希儿昵称", "aliases", []string{"希儿"}},
			{"喵喵别名帮助", "alias-help", nil},
			{"上传希儿照片", "photo-upload", []string{"希儿"}},
			{"上传希儿面板图", "panel-image-upload", []string{"希儿"}},
			{"删除希儿面板图1", "panel-image-remove", []string{"希儿", "1"}},
			{"希儿面板图列表", "panel-image-list", []string{"希儿"}},
			{"希儿", "character-card", nil},
			// miao's CharWiki takes a character's 照片, 写真, 图片 or 图像;
			// the word alone is not its.
			{"希儿照片", "photo", []string{"希儿"}},
			{"希儿写真", "photo", []string{"希儿"}},
			{"照片", "character-card", nil},
			// A UID glued to the word, as upstream's rules accept it, is the
			// same leading argument as a spaced one. StarRail-plugin's
			// challenge rules have no end anchor and read the first number in
			// the message as the UID; only a UID counts.
			{"深渊100000001", "abyss", []string{"100000001"}},
			{"上期简易深渊100000001", "abyss", []string{"100000001"}},
			{"深渊12", "character-card", nil},
			{"最新深渊100000001", "abyss-latest", []string{"100000001"}},
			{"忘却之庭100000001", "challenge", []string{"100000001"}},
			{"上期混沌100000001", "challenge", []string{"100000001"}},
			{"虚构叙事100000001", "story", []string{"100000001"}},
			{"末日100000001", "boss", []string{"100000001"}},
			{"往期异相仲裁100000001", "peak", []string{"100000001"}},
			// queryChallenge looks for 上期, 简易 and, for Anomaly Arbitration,
			// 往期 anywhere in the message, so they also follow the name, in
			// either order; the words stay in the command word. Only Anomaly
			// Arbitration's rule takes a leading 往期.
			{"深渊上期", "abyss", nil},
			{"深渊简易", "abyss", nil},
			{"上期深渊简易", "abyss", nil},
			{"深渊简易上期100000001", "abyss", []string{"100000001"}},
			{"最新简易深渊", "abyss-latest", nil},
			{"最新深渊简易上期", "abyss-latest", nil},
			{"忘却之庭上期", "challenge", nil},
			{"简易混沌本期100000001", "challenge", []string{"100000001"}},
			{"虚构叙事上期简易", "story", nil},
			{"末日幻影简易100000001", "boss", []string{"100000001"}},
			{"异相仲裁往期", "peak", nil},
			{"简易仲裁上期100000001", "peak", []string{"100000001"}},
			{"往期忘却之庭", "character-card", nil},
			{"往期虚构叙事", "character-card", nil},
			{"往期末日幻影", "character-card", nil},
			{"深渊往期", "character-card", nil},
			// Its Simulated and Divergent Universe rules read the UID the same
			// way; the run's number stays in the word. The rogue command looks
			// for 上期 anywhere too, and its rule takes no 往期.
			{"模拟宇宙100000001", "rogue", []string{"100000001"}},
			{"上期宇宙100000001", "rogue", []string{"100000001"}},
			{"模拟宇宙上期", "rogue", nil},
			{"宇宙本期100000001", "rogue", []string{"100000001"}},
			{"往期宇宙", "character-card", nil},
			{"宇宙简易", "character-card", nil},
			{"蝗灾100000001", "swarm", []string{"100000001"}},
			{"寰宇蝗灾100000001", "swarm", []string{"100000001"}},
			{"差分100000001", "divergent", []string{"100000001"}},
			{"差分宇宙100000001", "divergent", []string{"100000001"}},
			{"常规演算二100000001", "divergent-normal", []string{"100000001"}},
			{"上周差分宇宙记录三100000001", "divergent-week", []string{"100000001"}},
			{"本期演算100000001", "divergent-week", []string{"100000001"}},
			// miao's profileDetail takes a UID after 面板, 详情, 遗器 and
			// the like; after 伤害 the index group holds it, and nine digits
			// or more are a UID, as in miao.
			{"希儿面板100000001", "api-panel", []string{"希儿", "100000001"}},
			{"希儿面版100000001", "character", []string{"希儿", "100000001"}},
			{"希儿详情100000001", "character", []string{"希儿", "100000001"}},
			{"希儿遗器100000001", "score", []string{"希儿", "100000001"}},
			{"希儿圣遗物100000001", "score", []string{"希儿", "100000001"}},
			{"希儿伤害100000001", "build", []string{"希儿", "100000001"}},
			// StarRail-plugin's plmb rule ends at 列表, so miao's list
			// answers a UID after the word.
			{"面板100000001", "panel-list", []string{"100000001"}},
			{"面板列表100000001", "panel-list", []string{"100000001"}},
			// Atlas answers these before the commands they reach. On
			// Miao-Yunzai StarRail-plugin's rogue rule, having no end anchor,
			// takes 宇宙布洛妮娅 first; the rogue command ends with its words.
			{"宇宙布洛妮娅", "character-card", nil},
			{"于夜色中图鉴", "catalog", []string{"于夜色中"}},
			{"希儿培养", "materials", []string{"希儿"}},
			// Atlas's index lists answer these before the fallback card.
			{"光锥", "character-card", nil},
			{"五星光锥", "character-card", nil},
			{"遗器", "character-card", nil},
			// Atlas's guide for role module takes 攻略0 after the guide.
			{"希儿攻略0", "guides", []string{"希儿"}},
			// StarRail-plugin's panel rule takes 设置面板 and 切换面板 with
			// anything after them ahead of 面板换装; the bare words are
			// MiaoOrStarrail's.
			{"切换面板API2", "api-switch", nil},
			{"设置面板api1", "api-switch", nil},
			{"切换面板3", "api-switch", nil},
			{"切换面板", "api-switch", nil},
			{"API列表", "api-list", nil},
			{"api列表", "api-list", nil},
		},
	}
	for game, list := range cases {
		manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
		if err != nil {
			t.Fatal(err)
		}
		set, err := newCommandSet(manifest)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range list {
			id, args, ok := set.resolve(tc.word, nil)
			if len(args) == 0 {
				args = nil
			}
			if !ok || id != tc.id || !reflect.DeepEqual(args, tc.args) {
				t.Errorf("%s %s resolved to %q %v %v, want %s %v", game, tc.word, id, args, ok, tc.id, tc.args)
			}
		}
	}
}

// MiaoOrStarrail hands the words ending in 面板 to miao; StarRail-plugin's
// panel rule answers a UID after 面板, glued or spaced, with its own page.
// Other text after 面板 is not a UID, and the 面板图 words stay miao's.
func TestPanelWordsReachStarRailPluginWithAUID(t *testing.T) {
	manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	set, err := newCommandSet(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		word string
		args []string
		page bool
	}{
		{"希儿面板100000001", nil, true},
		{"希儿面板", []string{"100000001"}, true},
		{"希儿面板", nil, false},
		{"希儿面板", []string{"12"}, false},
		{"希儿详情100000001", nil, false},
	} {
		id, args, _ := set.resolve(tc.word, tc.args)
		if page := id == "api-panel" && len(args) == 2 && apiUID.MatchString(args[1]); page != tc.page {
			t.Errorf("%s %v resolved to %s %v", tc.word, tc.args, id, args)
		}
	}
	for _, word := range []string{"希儿面板图列表", "上传希儿面板图", "希儿面板更新"} {
		if id, _, _ := set.resolve(word, nil); id == "api-panel" {
			t.Errorf("%s reached the panel service", word)
		}
	}
}

// Warp words follow Miao-Yunzai's precedence, which the manifest order
// repeats: StarRail-plugin's rule (priority -114514, no end anchor), miao's
// rules (50, with its gachaStat switch forced on), then Yunzai's gcLog (300),
// so each word reaches one page and every page is reachable.
func TestGachaWordsFollowUpstreamPrecedence(t *testing.T) {
	manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	set, err := newCommandSet(manifest)
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string][]string{
		"gacha": {"记录", "分析", "统计", "跃迁记录", "跃迁统计", "抽卡记录", "抽卡分析", "抽卡统计", "角色记录", "角色分析", "角色统计",
			"光锥记录", "武器分析", "常驻记录", "新手记录", "新手统计", "角色跃迁记录", "光锥抽卡统计", "角色记录100000001"},
		"gacha-detail": {"喵喵抽卡记录", "喵喵光锥记录", "喵喵常驻祈愿", "喵喵集录记录", "抽卡祈愿", "抽奖记录", "角色祈愿", "光锥祈愿",
			"常驻祈愿", "集录记录", "up记录", "up池记录", "角色池记录", "光锥池分析", "常驻池记录", "角色光锥记录", "up角色祈愿"},
		"gacha-stat": {"喵喵抽卡统计", "喵喵角色统计", "喵喵全部统计", "抽奖统计", "集录统计", "up统计", "全部统计", "版本统计", "角色池统计",
			"光锥池统计", "常驻池统计", "抽卡池统计", "全部池统计", "up全部统计"},
		"gacha-log": {"全部记录", "全部祈愿", "全部分析", "全部池记录", "全部角色记录", "全部光锥联动记录", "角色联动记录", "角色联动池祈愿",
			"光锥联动记录", "武器联动分析", "新手祈愿", "新手池记录", "角色池池记录", "集录池池记录"},
		"gacha-versions": {"新手池统计", "新手池池统计", "角色池池统计", "光锥池池统计", "up池池统计"},
		// Upstream's rule without an end anchor takes 记录帮助 too; the help
		// Yunzai's replies point to stays reachable.
		"gacha-help": {"记录帮助"},
	}
	for want, words := range pages {
		for _, word := range words {
			if id, _, _ := set.resolve(word, nil); id != want {
				t.Errorf("%s resolved to %q, want %s", word, id, want)
			}
		}
	}
	// Yunzai reads Star Rail's missing 集录 pool for it and draws nothing.
	if id, _, _ := set.resolve("集录池池统计", nil); strings.HasPrefix(id, "gacha") {
		t.Errorf("集录池池统计 resolved to %s", id)
	}
}

// As on the host, a fallback command takes a word only when no ordinary
// command does, wherever the manifest declares it.
func TestFallbackCommandsResolveAfterOrdinaryOnes(t *testing.T) {
	manifest, err := pluginmeta.Read([]byte(`{"id":"raylea.test","version":"1.0.0","commands":[
		{"id":"card","trigger":{"type":"pattern","pattern":"^.+$","fallback":true}},
		{"id":"panel","trigger":{"type":"pattern","pattern":"^(?P<character>.+?)面板$"}}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	set, err := newCommandSet(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for word, want := range map[string]string{"希儿面板": "panel", "希儿": "card"} {
		if id, _, ok := set.resolve(word, nil); !ok || id != want {
			t.Errorf("%s resolved to %q %v, want %s", word, id, ok, want)
		}
	}
}

// A hint or static picture answers a command by ID, and the commands ahead
// of Atlas are named by ID, so each must name one the manifest has.
func TestShippedHintsNameManifestCommands(t *testing.T) {
	manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	var data Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &data); err != nil {
		t.Fatal(err)
	}
	ids := slices.AppendSeq(slices.Collect(maps.Keys(data.Hints)), maps.Keys(data.Pictures.Static))
	for _, id := range append(ids, aheadOfAtlas...) {
		if !slices.ContainsFunc(manifest.Commands, func(command pluginmeta.Command) bool { return command.ID == id }) {
			t.Errorf("answer %s names no command", id)
		}
	}
}
