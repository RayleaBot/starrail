package app

import (
	"testing"
)

func guidePost(t *testing.T, raw string) map[string]any {
	t.Helper()
	return cloudObject(t, raw)
}

func TestGuidePostImageTakesTheTallest(t *testing.T) {
	post := guidePost(t, `{"post":{"subject":"希儿一图流"},"image_list":[{"url":"https://upload-bbs.miyoushe.com/a.png","height":"600"},{"url":"https://upload-bbs.miyoushe.com/b.png","height":"900"},{"url":"https://upload-bbs.miyoushe.com/c.png","height":"900"}]}`)
	if got := guidePostImage(guideTarget{Word: "希儿"}, post); got != "https://upload-bbs.miyoushe.com/b.png" {
		t.Fatal(got)
	}
	if got := guidePostImage(guideTarget{Word: "银狼"}, post); got != "" {
		t.Fatal("another character's post matched")
	}
	// A Trailblazer path matches both words.
	post = guidePost(t, `{"post":{"subject":"开拓者（存护）一图流"},"image_list":[{"url":"https://upload-bbs.miyoushe.com/a.png","height":"900"}]}`)
	if got := guidePostImage(guideTarget{Word: "存护", Trailblazer: true}, post); got != "https://upload-bbs.miyoushe.com/a.png" {
		t.Fatal(got)
	}
	if got := guidePostImage(guideTarget{Word: "毁灭", Trailblazer: true}, post); got != "" {
		t.Fatal("another path matched")
	}
}

func TestGuideTargetsTellCharactersApart(t *testing.T) {
	a := App{Catalog: Catalog{Entries: []Entry{
		{ID: "1304", Name: "砂金", Kind: "character"},
		{ID: "1501", Name: "砂金•戏浪", Kind: "character"},
		{ID: "1013", Name: "黑塔", Kind: "character"},
		{ID: "1401", Name: "大黑塔", Kind: "character"},
		{ID: "1006", Name: "银狼", Kind: "character"},
		{ID: "1502", Name: "银狼LV.999", Kind: "character"},
		{ID: "1102", Name: "希儿", Kind: "character"},
		{ID: "2102", Name: "希儿Pro", Kind: "character"},
		{ID: "8007", Name: "穹·记忆", Kind: "character"},
	}}}
	targets := map[string]guideTarget{}
	for _, entry := range a.Catalog.Entries {
		targets[entry.Name] = a.guideTargetFor(entry)
	}
	if target := targets["希儿Pro"]; target.Word != "希儿" {
		t.Fatalf("enhanced character target = %+v", target)
	}
	for _, tc := range []struct {
		name, subject string
		want          bool
	}{
		{"砂金", "【V4.5攻略】「砂金·戏浪」丨全方位·角色解析攻略", false},
		{"砂金", "「砂金」全方位·一图流", true},
		{"砂金•戏浪", "夏日限定款丨砂金戏浪攻略", true},
		{"黑塔", "「大黑塔」丨全方位", false},
		{"银狼", "银狼lv999攻略", false},
		{"银狼LV.999", "银狼lv999攻略", true},
		{"希儿Pro", "「希儿」全方位·一图流", true},
		{"穹·记忆", "『记忆主』攻略", true},
		{"穹·记忆", "开拓者-记忆丨全方位", true},
		{"穹·记忆", "开拓者-毁灭", false},
	} {
		if got := targets[tc.name].names(tc.subject); got != tc.want {
			t.Errorf("%s matches %q = %v, want %v", tc.name, tc.subject, got, tc.want)
		}
	}
}
