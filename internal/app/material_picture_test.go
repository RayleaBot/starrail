package app

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// Each collection keeps the picture at its own place; a post without it
// sends the search on to the next collection, as upstream's getImg does.
func TestMaterialPictureReadsYunzaiCollectionsInTurn(t *testing.T) {
	image := func(name string) string { return `"https://upload-bbs.miyoushe.com/` + name + `.jpg"` }
	pages := map[string]string{
		"1998643": `{"posts":[{"post":{"subject":"角色攻略 符玄","images":[` + image("a") + `,` + image("b") + `]}},{"post":{"subject":"角色攻略 景元","images":[` + image("c") + `,` + image("d") + `,` + image("e") + `]}}]}`,
		"2146693": `{"posts":[{"post":{"subject":"符玄丨材料一图流","images":[` + image("f") + `]}}]}`,
		"2279356": `{"posts":[{"post":{"subject":"『白露』材料全收集","images":[` + image("g") + `]},"image_list":[{"url":` + image("h") + `}]}]}`,
	}
	c := PublicContentClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		body := `{"retcode":0,"data":` + pages[r.URL.Query().Get("collection_id")] + `}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	for name, want := range map[string]string{"景元": "e", "符玄": "f", "白露": "h", "希儿": ""} {
		got, err := c.materialPicture(t.Context(), name)
		if want != "" {
			want = "https://upload-bbs.miyoushe.com/" + want + ".jpg" + materialOSS
		}
		if err != nil || got != want {
			t.Errorf("%s = %q %v, want %q", name, got, err, want)
		}
	}
}
