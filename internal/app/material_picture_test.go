package app

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// Prefer the active collection, skip multi-character covers and tall light-cone
// guides, and distinguish a character from a longer name that contains it.
func TestMaterialPictureFindsTheCharacterPost(t *testing.T) {
	a := App{Catalog: Catalog{Entries: []Entry{
		{ID: "1205", Name: "刃", Kind: "character"},
		{ID: "1507", Name: "千冶•刃", Kind: "character"},
		{ID: "1201", Name: "青雀", Kind: "character"},
		{ID: "1108", Name: "桑博", Kind: "character"},
		{ID: "1202", Name: "停云", Kind: "character"},
		{ID: "1102", Name: "希儿", Kind: "character"},
	}}}
	image := func(name string, width, height int) map[string]any {
		return map[string]any{"url": "https://upload-bbs.miyoushe.com/" + name + ".jpg", "width": width, "height": height}
	}
	post := func(subject string, images ...map[string]any) map[string]any {
		return map[string]any{"post": map[string]any{"subject": subject}, "image_list": images}
	}
	pages := map[string][]map[string]any{
		"2279356": {
			post("『青雀/桑博/停云』材料全收集", image("cover", 2560, 1440)),
			post("【V4.2攻略】千冶·刃前瞻丨行迹属性/加点/材料", image("a", 1080, 1440)),
		},
		"2146693": {
			post("【V1.2前瞻】刃培养材料一图流", image("b", 1080, 1440)),
		},
		"1998643": {
			post("【V3.3攻略】角色攻略 青雀 机制/养成/就业解析", image("cover", 1080, 1440), image("guide", 1080, 1440), image("light-cone", 1242, 1600)),
			post("【V1.0攻略】关于青雀丨配队解析", image("cover", 1080, 1440), image("guide", 1080, 1440), image("banner", 1875, 411)),
			post("【V1.0攻略】角色攻略 青雀（机制/养成/就业解析）", image("cover", 1080, 1440), image("guide", 1080, 1440), image("c", 4026, 2774)),
		},
	}
	var collections []string
	c := PublicContentClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		id := r.URL.Query().Get("collection_id")
		collections = append(collections, id)
		body, err := json.Marshal(map[string]any{"retcode": 0, "data": map[string]any{"posts": pages[id]}})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})}
	for _, tc := range []struct {
		name, image string
		collections []string
	}{
		{"千冶•刃", "a", []string{"2279356"}},
		{"刃", "b", []string{"2279356", "2146693"}},
		{"青雀", "c", []string{"2279356", "2146693", "1998643"}},
		{"希儿", "", []string{"2279356", "2146693", "1998643"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			collections = nil
			entry, ok := a.Catalog.Resolve(tc.name, "character", nil)
			if !ok {
				t.Fatal("character missing")
			}
			target := a.guideTargetFor(entry)
			got, err := c.materialPicture(t.Context(), target, a.otherCharacterTargets(target))
			want := ""
			if tc.image != "" {
				want = "https://upload-bbs.miyoushe.com/" + tc.image + ".jpg" + materialOSS
			}
			if err != nil || got != want {
				t.Errorf("picture = %q %v, want %q", got, err, want)
			}
			if !reflect.DeepEqual(collections, tc.collections) {
				t.Errorf("collections = %v, want %v", collections, tc.collections)
			}
		})
	}
}
