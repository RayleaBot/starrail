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
	if got := guidePostImage("希儿", post); got != "https://upload-bbs.miyoushe.com/b.png" {
		t.Fatal(got)
	}
	if got := guidePostImage("银狼", post); got != "" {
		t.Fatal("another character's post matched")
	}
	// A Trailblazer path matches both words.
	post = guidePost(t, `{"post":{"subject":"开拓者（存护）一图流"},"image_list":[{"url":"https://upload-bbs.miyoushe.com/a.png","height":"900"}]}`)
	if got := guidePostImage("开拓者·存护", post); got != "https://upload-bbs.miyoushe.com/a.png" {
		t.Fatal(got)
	}
	if got := guidePostImage("开拓者·毁灭", post); got != "" {
		t.Fatal("another path matched")
	}
}
