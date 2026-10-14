package app

import (
	"context"
	"net/url"
	"slices"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Yunzai's 角色素材 (material.js) for Star Rail: 符玄素材, 符玄突破, 符玄培养
// send the character's material picture from 米游社 collections;
// 更新 reads the collections again.

// Yunzai's first collections, 小橙子阿 (1998643, stopped at V3.8) and
// 紫喵Azunya (2146693, stopped at V3.2), no longer update. Prefer the active
// 让我摸个鱼吧Moyu 养成材料合集 (2279356). In 1998643, only 角色攻略 posts
// through V2.5 put a wide material table third; later ones put a tall
// light-cone guide there. All three collections use image_list.
var materialCollections = []struct {
	id      string
	image   int
	subject string
	wide    bool
}{{"2279356", 0, "", false}, {"2146693", 0, "", false}, {"1998643", 2, "角色攻略", true}}

var materialOSS = "?x-oss-process=image//resize,s_1000/quality,q_80/auto-orient,0/interlace,1/format,jpg"

// materialPicture is the address of a character's material picture, "" when
// no collection has a post naming it.
func (c PublicContentClient) materialPicture(ctx context.Context, target guideTarget, others []guideTarget) (string, error) {
	for _, collection := range materialCollections {
		page, err := c.collection(ctx, collection.id)
		if err != nil {
			return "", err
		}
		for _, raw := range asList(page["posts"]) {
			item := asObject(raw)
			subject := asText(asObject(item["post"])["subject"])
			if !target.names(subject) || !strings.Contains(subject, collection.subject) || slices.ContainsFunc(others, func(other guideTarget) bool { return other.names(subject) }) {
				continue
			}
			images := asList(item["image_list"])
			if collection.image >= len(images) {
				continue
			}
			image := asObject(images[collection.image])
			if collection.wide {
				width, _ := strconv.ParseFloat(asText(image["width"]), 64)
				height, _ := strconv.ParseFloat(asText(image["height"]), 64)
				if width <= height {
					continue
				}
			}
			found := asText(image["url"])
			link, err := url.Parse(found)
			if err != nil || link.Scheme != "https" || !(strings.HasSuffix(link.Hostname(), ".miyoushe.com") || strings.HasSuffix(link.Hostname(), ".mihoyo.com")) {
				return "", gameError("public_invalid", "素材图片地址不在允许范围内。")
			}
			return found + materialOSS, nil
		}
	}
	return "", nil
}

func (a *App) otherCharacterTargets(target guideTarget) []guideTarget {
	var others []guideTarget
	for _, entry := range a.Catalog.Entries {
		if entry.Kind != "character" {
			continue
		}
		other := a.guideTargetFor(entry)
		if other.Word != target.Word || other.Trailblazer != target.Trailblazer {
			others = append(others, other)
		}
	}
	return others
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
		target := a.guideTargetFor(entry)
		if link, err = a.Content.materialPicture(ctx, target, a.otherCharacterTargets(target)); err != nil {
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
