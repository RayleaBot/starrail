package app

import (
	"encoding/json"
	"maps"
	"reflect"
	"slices"
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
			{"希儿面板", "character", []string{"希儿"}},
			{"面板", "panel-list", nil},
			{"开启到期活动推送", "subscribe", []string{"到期"}},
			{"推送公告", "content-push", nil},
			{"星琼任务", "monthly-task", nil},
			{"更新面板", "panel-refresh", nil},
			{"mys更新面板", "panel-refresh-account", nil},
			{"遗器列表", "artifact-list", nil},
			{"希儿换满命换满行迹", "talent-wiki", []string{"希儿换满命换满"}},
			{"希儿换满命", "panel-change", nil},
			{"圣遗物列表100000001", "artifact-list", []string{"100000001"}},
			{"希儿遗器", "score", []string{"希儿"}},
			{"希儿伤害", "build", []string{"希儿"}},
			{"希儿伤害2", "build", []string{"希儿", "2"}},
			{"上期忘却之庭", "challenge", []string{"上期"}},
			{"上期模拟宇宙", "rogue", []string{"上期"}},
			{"差分宇宙", "divergent", nil},
			{"常规演算二", "divergent-normal", nil},
			{"上周差分宇宙记录", "divergent-week", nil},
			{"周期演算", "divergent-cycle", nil},
			{"星琼统计", "monthly-history", nil},
			{"希儿行迹", "talent-wiki", []string{"希儿"}},
			{"希儿遗器排名", "rank", nil},
			{"最强希儿", "rank-top", nil},
			{"今年星琼统计", "monthly-history", nil},
			{"跃迁记录", "gacha", nil},
			{"十连光锥", "simulation", nil},
			{"更新跃迁记录", "gacha-background", nil},
			{"帮助", "help", nil},
			{"刷新行迹", "talent-refresh", nil},
			{"角色分析", "gacha", nil},
			{"光锥记录", "gacha", nil},
			{"抽卡统计", "gacha", nil},
			{"新手统计", "gacha", nil},
			{"up统计", "gacha-versions", nil},
			{"角色池统计", "gacha-versions", nil},
			{"喵喵光锥记录", "gacha-detail", nil},
			{"喵喵全部统计", "gacha-stat", nil},
			{"全部统计", "gacha-stat", nil},
			{"角色联动记录", "gacha", nil},
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

// miao's pages take the 喵喵 words, and with its setting the plain words
// StarRail-plugin's rule, which runs first upstream, leaves.
func TestMiaoGachaWordsFollowUpstreamPrecedence(t *testing.T) {
	manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	set, err := newCommandSet(manifest)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		word    string
		off, on bool
	}{
		{"喵喵抽卡记录", true, true},
		{"喵喵常驻统计", true, true},
		{"抽卡记录", false, false},
		{"抽卡统计", false, false},
		{"抽奖记录", false, true},
		{"角色池记录", false, true},
		{"up统计", false, true},
		{"全部统计", false, true},
		{"新手池统计", false, false},
		{"角色联动记录", false, false},
	}
	for _, tc := range cases {
		if off, on := set.miaoGacha(tc.word, false), set.miaoGacha(tc.word, true); off != tc.off || on != tc.on {
			t.Errorf("%s draws miao's page %v without and %v with the setting, want %v and %v", tc.word, off, on, tc.off, tc.on)
		}
	}
}

// A hint or static picture answers a command by ID, so each must name one
// the manifest has.
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
	for _, id := range ids {
		if !slices.ContainsFunc(manifest.Commands, func(command pluginmeta.Command) bool { return command.ID == id }) {
			t.Errorf("answer %s names no command", id)
		}
	}
}
