package app

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuidesFixedCollectionFullOriginalsAndVersionedDefault(t *testing.T) {
	s := GuideSettings{Path: filepath.Join(t.TempDir(), "guides.json")}
	if _, err := s.Manage("guides.configure", map[string]any{"default_source": "2", "revision": 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Manage("guides.configure", map[string]any{"default_source": "3", "revision": 0}); err == nil {
		t.Fatal("stale settings overwritten")
	}
	client := PublicContentClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "bbs-api.mihoyo.com" || r.URL.Query().Get("collection_id") != "1998643" || r.Header.Get("Cookie") != "" {
			t.Fatal("wrong destination")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":{"posts":[{"post":{"post_id":"1","game_id":6,"subject":"希儿攻略","content":"<p>合成正文</p>","images":["https://upload-bbs.miyoushe.com/a.png"]}},{"post":{"post_id":"2","game_id":6,"subject":"其他"}}]}}`))}, nil
	})}
	out, err := client.guides(context.Background(), ContentQuery{Source: "2", Query: "希儿"})
	if err != nil {
		t.Fatal(err)
	}
	posts := out["items"].([]PublicPost)
	if len(posts) != 1 || len(posts[0].Parts) != 1 || len(posts[0].Images) != 1 || out["next_offset"] != -1 {
		t.Fatal(out)
	}
}
