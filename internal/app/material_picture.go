package app

import (
	"context"
	"net/url"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Yunzai's 角色素材 (material.js) for Star Rail: 符玄素材, 符玄突破, 符玄培养
// send the character's material picture from 小橙子啊's 米游社 collection,
// else from two others; 更新 reads the collections again.

// materialCollections are Yunzai's srCollection_id in the order it tries
// them, with the index of the picture each keeps in a post: the first two
// in the post's own images, the last in the listed images.
var materialCollections = []struct {
	id    string
	image int
	post  bool
}{{"1998643", 2, true}, {"2146693", 0, true}, {"2279356", 0, false}}

var materialOSS = "?x-oss-process=image//resize,s_1000/quality,q_80/auto-orient,0/interlace,1/format,jpg"

// materialPicture is the address of a character's material picture, "" when
// no collection has a post naming it.
func (c PublicContentClient) materialPicture(ctx context.Context, name string) (string, error) {
	for _, collection := range materialCollections {
		page, err := c.collection(ctx, collection.id)
		if err != nil {
			return "", err
		}
		for _, raw := range asList(page["posts"]) {
			item := asObject(raw)
			post := asObject(item["post"])
			if !strings.Contains(asText(post["subject"]), name) {
				continue
			}
			images := asList(item["image_list"])
			if collection.post {
				images = asList(post["images"])
			}
			if collection.image >= len(images) {
				break
			}
			found := asText(images[collection.image])
			if !collection.post {
				found = asText(asObject(images[collection.image])["url"])
			}
			link, err := url.Parse(found)
			if err != nil || link.Scheme != "https" || !(strings.HasSuffix(link.Hostname(), ".miyoushe.com") || strings.HasSuffix(link.Hostname(), ".mihoyo.com")) {
				return "", gameError("public_invalid", "素材图片地址不在允许范围内。")
			}
			return found + materialOSS, nil
		}
	}
	return "", nil
}

// materialCommand sends a character's material picture; a name that is no
// character is left to the reference material search. Upstream's filter
// leaves 培养 in the name, so 符玄培养 never finds a character there; the
// trigger strips it here as it does 素材.
func (a *App) materialCommand(ctx context.Context, event *rayleabot.EventContext, args []string) (bool, error) {
	raw := strings.Join(args, "")
	refresh := strings.Contains(raw, "更新")
	entry, ok := a.Catalog.Resolve(strings.ReplaceAll(raw, "更新", ""), "character", a.aliasMap(event))
	if !ok {
		return false, nil
	}
	// Upstream lists 8001 and 8003, the Trailblazer's paths when it was
	// written; every path is the Trailblazer.
	if len(entry.ID) == 4 && strings.HasPrefix(entry.ID, "80") {
		return true, event.SendText("暂无主角素材")
	}
	key := a.Game.ID + "/material/" + entry.Name
	link, cached := a.guides.get(key)
	if !cached || refresh {
		var err error
		if link, err = a.Content.materialPicture(ctx, entry.Name); err != nil {
			return true, event.SendText("暂无素材数据，请稍后再试")
		}
		if link == "" {
			// Yunzai answers nothing when no collection names the character.
			return true, event.Result(map[string]any{"handled": false})
		}
		a.guides.put(key, link)
	}
	return true, event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image(link))
}
