package app

import (
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/artwork"
	"github.com/RayleaBot/starrail/internal/pluginmeta"
)

func TestAtlasRulesTakeMessagesAsPickRule(t *testing.T) {
	starRail := func(condition int, pick ...string) AtlasRule {
		return AtlasRule{Condition: condition, Pick: pick, PrefixForReg: `(\*|#星铁)`}
	}
	for _, tc := range []struct {
		rule      AtlasRule
		msg, name string
	}{
		{starRail(3, "图鉴"), "#星铁希儿", "希儿"},
		{starRail(3, "图鉴"), "*希儿", "希儿"},
		{starRail(3, "图鉴"), "希儿图鉴", "希儿"},
		{starRail(3, "图鉴"), "#星铁希儿 图鉴", "希儿"},
		// Every prefix at the start goes.
		{starRail(3, "图鉴"), "#星铁*希儿", "希儿"},
		// Yunzai's # alone is not the Star Rail prefix.
		{starRail(3, "图鉴"), "#希儿", ""},
		{starRail(3, "图鉴"), "希儿", ""},
		{starRail(3, "图鉴"), "#星铁图鉴", ""},
		{starRail(5, "攻略0", "攻略", "指南"), "#星铁希儿攻略", "希儿"},
		{starRail(5, "攻略0", "攻略", "指南"), "希儿攻略", "希儿"},
		{starRail(5, "攻略0", "攻略", "指南"), "#星铁希儿攻略0", "希儿"},
		{starRail(5, "攻略0", "攻略", "指南"), "希儿指南", "希儿"},
		{starRail(5, "攻略0", "攻略", "指南"), "#星铁希儿", ""},
		{starRail(1, "图鉴"), "#星铁希儿图鉴", "希儿图鉴"},
		{starRail(1, "图鉴"), "希儿图鉴", ""},
		// Condition 2 leaves the prefix in the name, which then names nothing.
		{starRail(2, "攻略"), "#星铁希儿攻略", "#星铁希儿"},
		{starRail(4, "攻略"), "#星铁希儿攻略", "希儿"},
		{starRail(4, "攻略"), "希儿攻略", ""},
		{starRail(0, "图鉴"), "希儿", "希儿"},
		// The prefix counts only before a single line, as ^prefix.*$ reads it.
		{starRail(1, "图鉴"), "#星铁希儿\n天赋", ""},
		// Without a prefix expression the prefix is #, and without Pick words
		// the rule picks 图鉴.
		{AtlasRule{Condition: 1}, "#希儿", "希儿"},
		{AtlasRule{Condition: 2}, "希儿图鉴", "希儿"},
	} {
		rule, err := tc.rule.compile()
		if err != nil {
			t.Fatal(err)
		}
		if name, ok := rule.take(tc.msg); name != tc.name || ok != (tc.name != "") {
			t.Errorf("condition %d took %q as %q %v, want %q", tc.rule.Condition, tc.msg, name, ok, tc.name)
		}
	}
}

func TestShippedAtlasRulesCompile(t *testing.T) {
	var game Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &game); err != nil {
		t.Fatal(err)
	}
	for _, library := range game.Pictures.Atlas {
		if _, ok := library.Rules["config"]; !ok {
			t.Errorf("%s has no config rule", library.Source)
		}
		for name, rule := range library.Rules {
			if _, err := rule.compile(); err != nil || rule.Condition < 0 || rule.Condition > 5 {
				t.Errorf("%s rule %s: %v", library.Source, name, err)
			}
		}
	}
}

// atlasApp is an app with the shipped Atlas rules and a downloaded library,
// which keys its images by ID and names them in its othername files.
func atlasApp(t *testing.T) (*App, string) {
	t.Helper()
	var game Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &game); err != nil {
		t.Fatal(err)
	}
	catalog, err := ParseCatalog([]byte(`{"version":"test","entries":[
		{"id":"1102","name":"希儿","aliases":["蝴蝶"],"kind":"character"},
		{"id":"23001","name":"于夜色中","kind":"weapon"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeArtwork(t, root, "star-rail-atlas/path.json", `{
		"enemy":{"BRYC-SEG1":"/enemy/BRYC-SEG1.png"},
		"relic":{"101":"/relic/101.png","102":"/relic/102.yaml"},
		"guide for role":{"1102":"/guide for role/1102.png"},
		"material for role":{"1102":"/material for role/1102.png"},
		"lightcone":{"23001":"/lightcone/23001.png"},
		"role":{"1102":"/role/1102.png","8001":"/role/8001.png"}
	}`)
	for _, file := range []string{"enemy/BRYC-SEG1.png", "relic/101.png", "relic/102.yaml", "guide for role/1102.png", "material for role/1102.png", "lightcone/23001.png", "role/1102.png", "role/8001.png"} {
		writeArtwork(t, root, "star-rail-atlas/"+file, "image")
	}
	writeArtwork(t, root, "star-rail-atlas/othername/enemy.yaml", "BRYC-SEG1:\n  - 宇宙布洛妮娅\n")
	writeArtwork(t, root, "star-rail-atlas/othername/relic.yaml", "'101':\n  - 云无留迹的过客\n  - 过客套\n'102':\n  - 快枪手\n")
	writeArtwork(t, root, "star-rail-atlas/othername/guide for role.yaml", "'1102':\n  - 希儿\n  - 希尔\n")
	writeArtwork(t, root, "star-rail-atlas/othername/material for role.yaml", "'1102':\n  - 希儿\n")
	writeArtwork(t, root, "star-rail-atlas/othername/lightcone.yaml", "'23001':\n  - 于夜色中\n  - 希儿专武\n")
	writeArtwork(t, root, "star-rail-atlas/othername/role.yaml", "# roles\n'1102':\n  - 希儿\n  - 希尔\n'8001':\n  - 物理开拓者\n  - 8002\n")
	return &App{Artwork: &artwork.Store{Root: root, Sources: game.Artwork}, Game: game, Catalog: catalog}, root
}

// atlasReply is the picture Atlas answers a message with, as Yunzai reads
// it, "" when it leaves the message to others.
func atlasReply(a *App, msg string) string {
	run := atlasRun{app: a}
	if !run.atlas(a.miaoAccept(msg, func() map[string]string { return nil })) || len(run.answers) == 0 {
		return ""
	}
	return run.answers[0].picture.Path
}

// atlasAnswers are the replies Atlas sends for a sender's message, a picture
// as its path and a list joined, with whether Atlas ends the message.
func atlasAnswers(a *App, sender, msg string) ([]string, bool) {
	aliases := func() map[string]string { return nil }
	run := atlasRun{app: a, owner: Subject{ActorID: sender}}
	ended := run.atlas(a.miaoAccept(msg, aliases))
	replies := []string{}
	for _, answer := range run.answers {
		if answer.list != nil {
			replies = append(replies, strings.Join(answer.list, " "))
		} else {
			replies = append(replies, answer.picture.Path)
		}
	}
	return replies, ended
}

func TestAtlasAnswersMessagesByTheLibraryRules(t *testing.T) {
	a, _ := atlasApp(t)
	for msg, want := range map[string]string{
		// The library's aliases name the key, in path.json's module order.
		"#星铁希儿":     "role/1102.png",
		"#星铁希尔":     "role/1102.png",
		"*希儿":       "role/1102.png",
		"#星铁物理开拓者":  "role/8001.png",
		"#星铁希儿专武":   "lightcone/23001.png",
		"#星铁于夜色中图鉴": "lightcone/23001.png",
		"于夜色中图鉴":    "lightcone/23001.png",
		"#星铁过客套":    "relic/101.png",
		"#星铁宇宙布洛妮娅": "enemy/BRYC-SEG1.png",
		"#星铁希儿攻略":   "guide for role/1102.png",
		"希儿攻略":      "guide for role/1102.png",
		"#星铁希尔攻略0":  "guide for role/1102.png",
		"希儿指南":      "guide for role/1102.png",
		"#星铁希儿培养":   "material for role/1102.png",
		"希儿突破":      "material for role/1102.png",
		"#星铁不存在":    "",
		"#希儿":       "",
		"希儿":        "",
		"#星铁快枪手":    "",
		"#星铁希儿攻略2":  "",
		"#星铁希儿 攻略":  "guide for role/1102.png",
		// A name the library writes as a bare number is its text here.
		"#星铁8002": "role/8001.png",
		// miao's wiki check takes a character before a wiki word first.
		"#星铁希儿图鉴": "",
		"#星铁蝴蝶资料": "",
		"#星铁希儿天赋": "",
		"希儿图鉴":    "",
		// Its names are the catalog's, not the library's.
		"#星铁希尔图鉴": "role/1102.png",
	} {
		if got := atlasReply(a, msg); got != want {
			t.Errorf("%q answered with %q, want %q", msg, got, want)
		}
	}
}

func TestAtlasReadsTheLibraryAgainAfterADownload(t *testing.T) {
	a, root := atlasApp(t)
	if got := atlasReply(a, "#星铁希儿"); got != "role/1102.png" {
		t.Fatal(got)
	}
	writeArtwork(t, root, "star-rail-atlas/path.json", `{"role":{"1102":"/role/8001.png"}}`)
	if got := atlasReply(a, "#星铁希儿"); got != "role/1102.png" {
		t.Fatal("the library was read again before a download", got)
	}
	writeArtwork(t, root, "star-rail-atlas.json", `{}`)
	if got := atlasReply(a, "#星铁希儿"); got != "role/8001.png" {
		t.Fatal("the downloaded library was not read", got)
	}
}

// Atlas sees a command's message as Miao-Yunzai writes it, after the
// commands that follow StarRail-plugin's rules and before the others.
func TestAtlasAnswersBetweenStarRailPluginAndTheOtherCommands(t *testing.T) {
	manifest, err := pluginmeta.Read(pluginFile(t, "info.json"))
	if err != nil {
		t.Fatal(err)
	}
	set, err := newCommandSet(manifest)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := atlasApp(t)
	for word, want := range map[string]struct {
		command string
		ahead   bool
		picture string
	}{
		// A character's name reaches Atlas before the fallback card.
		"希儿":     {"character-card", false, "role/1102.png"},
		"希儿卡片":   {"character-card", false, ""},
		"于夜色中图鉴": {"catalog", false, "lightcone/23001.png"},
		"希儿图鉴":   {"catalog", false, ""},
		"希儿培养":   {"materials", false, "material for role/1102.png"},
		// StarRail-plugin's strategy answers first and passes the message
		// on, so Atlas answers after the guide.
		"希儿攻略": {"guides", true, "guide for role/1102.png"},
		// Its other rules end the messages they answer.
		"希儿面板": {"api-panel", true, ""},
		"宇宙":   {"rogue", true, ""},
		"角色":   {"profile", true, ""},
		"体力":   {"note", true, ""},
		// Its rogue rule has no end anchor, so on Miao-Yunzai it also takes
		// 宇宙布洛妮娅, one of the library's enemies; the rogue command ends
		// with its words, so Atlas answers.
		"宇宙布洛妮娅": {"character-card", false, "enemy/BRYC-SEG1.png"},
		// 收益曲线 passes the message on to Atlas, which finds nothing.
		"收益曲线": {"income-curve", false, ""},
	} {
		command, _, _ := set.resolve(word, nil)
		ahead := slices.Contains(aheadOfAtlas, command)
		event := &rayleabot.EventContext{Event: rayleabot.Event{Payload: map[string]any{"command": word}}}
		if command != want.command || ahead != want.ahead {
			t.Errorf("%s resolved to %s, ahead of Atlas %v", word, command, ahead)
		}
		if got := atlasReply(a, yunzaiMessage(event)); got != want.picture {
			t.Errorf("Atlas answered %s with %q, want %q", word, got, want.picture)
		}
	}
}

func TestAtlasIndexListsAndPicksByNumber(t *testing.T) {
	a, root := atlasApp(t)
	writeArtwork(t, root, "star-rail-atlas/index/lightcone.yaml", "光锥:\n  - 五星光锥\n  - 四星光锥\n五星光锥:\n  - 于夜色中\n  - 希儿专武\n  - 银河铁道之夜\n")
	writeArtwork(t, root, "star-rail-atlas/index/relic.yaml", "遗器:\n  - 云无留迹的过客\n")
	steps := []struct {
		sender, msg string
		replies     []string
		ended       bool
	}{
		// A key of the index answers with its list, led by the prefix the
		// module's rule writes, and the message goes on to the others.
		{"a", "#星铁五星光锥", []string{"*于夜色中 *希儿专武 *银河铁道之夜"}, false},
		// A number picks an entry, which answers and ends the message; the
		// pick counts twice, once more as the entry runs through Atlas, and
		// an answer resets the count.
		{"a", "1", []string{"lightcone/23001.png"}, true},
		{"b", "1", []string{}, false},
		{"a", "2", []string{"lightcone/23001.png"}, true},
		// An entry without a picture answers nothing.
		{"a", "3", []string{}, false},
		{"a", "你好", []string{}, false},
		// The fourth count without an answer drops the list.
		{"a", "1", []string{}, false},
		{"a", "1", []string{}, false},
		// An entry that is itself a key answers with its list and ends the
		// message; its count is left at two, so the next pick drops the list
		// as it answers.
		{"b", "*光锥", []string{"*五星光锥 *四星光锥"}, false},
		{"b", "1", []string{"*于夜色中 *希儿专武 *银河铁道之夜"}, true},
		{"b", "1", []string{"lightcone/23001.png"}, true},
		{"b", "1", []string{}, false},
		{"c", "#星铁遗器", []string{"*云无留迹的过客"}, false},
		{"c", "1", []string{"relic/101.png"}, true},
		// miao's check leaves Atlas its own command word, which still counts.
		{"d", "#星铁五星光锥", []string{"*于夜色中 *希儿专武 *银河铁道之夜"}, false},
		{"d", "#星铁希儿图鉴", []string{}, false},
		{"d", "#星铁希儿天赋", []string{}, false},
		{"d", "#星铁蝴蝶资料", []string{}, false},
		{"d", "1", []string{}, false},
	}
	for _, step := range steps {
		replies, ended := atlasAnswers(a, step.sender, step.msg)
		if !reflect.DeepEqual(replies, step.replies) || ended != step.ended {
			t.Fatalf("%s %q: %q %v, want %q %v", step.sender, step.msg, replies, ended, step.replies, step.ended)
		}
	}
}

func TestAtlasPagesFollowReplyMessageArray(t *testing.T) {
	lines := []string{"请直接发送数字序号或对应指令："}
	for index := range 200 {
		lines = append(lines, strconv.Itoa(index+1)+"、*五星光锥")
	}
	pages := atlasPages(lines)
	// Lines join until a message reaches 100 characters.
	if first := pages[0][0]; strings.Count(first, "\n") != 11 || !strings.HasPrefix(first, "请直接发送数字序号或对应指令：\n1、*五星光锥") {
		t.Fatalf("%q", first)
	}
	if len(pages) != 1 || pages[0][len(pages[0])-1] != "第1页，共1页" {
		t.Fatal(pages)
	}
	many := []string{}
	for range 2000 {
		many = append(many, strings.Repeat("长", 100))
	}
	pages = atlasPages(many)
	if len(pages) != 22 || len(pages[0]) != 96 || pages[0][95] != "第1页...未完待续" || pages[21][len(pages[21])-1] != "第22页，共22页" {
		t.Fatal(len(pages), len(pages[0]), pages[21][len(pages[21])-1])
	}
}
