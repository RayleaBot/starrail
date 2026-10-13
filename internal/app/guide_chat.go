package app

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Chat 攻略 sends the image StarRail-plugin's strategy command does: the
// tallest image of the first collection post naming the character.

var guideWord = regexp.MustCompile(`^(更新|刷新)?(.+?)攻略([0-9]+|all)?$`)

// guideOSS is the resize the upstream commands ask the image host for.
var guideOSS = "?x-oss-process=image//resize,s_1200/quality,q_90/auto-orient,0/interlace,1/format,jpg"

// guideSamples are the characters the upstream 攻略帮助 replies show.
var guideSamples = "希儿"

// guideFolding drops the marks posts vary between a name's parts.
var guideFolding = strings.NewReplacer("·", "", "•", "", "・", "", "-", "", ".", "", " ", "")

// guideFold writes a name or a post subject in lower case without those
// marks, so 砂金•戏浪 and 砂金·戏浪, or 银狼LV.999 and 银狼lv999, read alike.
func guideFold(text string) string { return strings.ToLower(guideFolding.Replace(text)) }

// guideTarget is how a post subject names a character: Word, its folded
// name, outside the longer names of other characters that contain it
// (砂金·戏浪 for 砂金, 大黑塔 for 黑塔); a Trailblazer's Word is its path,
// named with 开拓者 or as the path's 主 (记忆主).
type guideTarget struct {
	Word        string
	Longer      []string
	Trailblazer bool
}

func (a *App) guideTargetFor(entry Entry) guideTarget {
	if base, path, ok := strings.Cut(strings.ReplaceAll(entry.Name, "•", "·"), "·"); ok && (base == "穹" || base == "星" || base == "开拓者") {
		return guideTarget{Word: path, Trailblazer: true}
	}
	// A Pro entry, a character's enhanced version, is looked up under the character.
	if name, ok := strings.CutSuffix(entry.Name, "Pro"); ok {
		if i := slices.IndexFunc(a.Catalog.Entries, func(e Entry) bool { return e.Kind == "character" && e.Name == name }); i >= 0 {
			entry = a.Catalog.Entries[i]
		}
	}
	target := guideTarget{Word: guideFold(entry.Name)}
	// Upstream writes 托帕 alone.
	if entry.Name == "托帕&账账" {
		target.Word = "托帕"
	}
	for _, other := range a.Catalog.Entries {
		if other.Kind != "character" || other.ID == entry.ID {
			continue
		}
		for _, name := range append([]string{other.Name}, other.Aliases...) {
			if folded := guideFold(name); folded != target.Word && strings.Contains(folded, target.Word) {
				target.Longer = append(target.Longer, folded)
			}
		}
	}
	// A longer name goes before a shorter one inside it.
	slices.SortFunc(target.Longer, func(x, y string) int { return len(y) - len(x) })
	return target
}

func (t guideTarget) names(subject string) bool {
	subject = guideFold(subject)
	if t.Trailblazer {
		return strings.Contains(subject, "开拓者") && strings.Contains(subject, t.Word) || strings.Contains(subject, t.Word+"主")
	}
	for _, longer := range t.Longer {
		subject = strings.ReplaceAll(subject, longer, "")
	}
	return strings.Contains(subject, t.Word)
}

// guidePostImage is the image a post gives for a character's guide, or ""
// when the post is not the character's: its tallest image, the first of
// equal ones, as StarRail-plugin picks it. The target checks the subject's
// character name or Trailblazer path.
func guidePostImage(target guideTarget, item map[string]any) string {
	if !target.names(asText(asObject(item["post"])["subject"])) {
		return ""
	}
	images, _ := item["image_list"].([]any)
	best, height := "", -1.0
	for _, image := range images {
		if value, _ := strconv.ParseFloat(asText(asObject(image)["height"]), 64); value > height {
			best, height = asText(asObject(image)["url"]), value
		}
	}
	return best
}

// guidePicture reads a source's collections and returns the guide image of
// the first post naming the character, resized as upstream asks; "" when no
// post names it.
func (c PublicContentClient) guidePicture(ctx context.Context, source GuideSource, target guideTarget) (string, error) {
	pages := make([]map[string]any, len(source.Collections))
	failures := make([]error, len(source.Collections))
	var wait sync.WaitGroup
	for i, id := range source.Collections {
		wait.Add(1)
		go func() {
			defer wait.Done()
			pages[i], failures[i] = c.collection(ctx, id)
		}()
	}
	wait.Wait()
	if err := errors.Join(failures...); err != nil {
		return "", err
	}
	for _, page := range pages {
		posts, _ := page["posts"].([]any)
		for _, post := range posts {
			if found := guidePostImage(target, asObject(post)); found != "" {
				link, err := url.Parse(found)
				if err != nil || link.Scheme != "https" || !(strings.HasSuffix(link.Hostname(), ".miyoushe.com") || strings.HasSuffix(link.Hostname(), ".mihoyo.com")) {
					return "", gameError("public_invalid", "攻略图片地址不在允许范围内。")
				}
				return found + guideOSS, nil
			}
		}
	}
	return "", nil
}

// guideCache keeps the guide images found, as upstream keeps the downloaded
// files, until 更新 asks again or a day passes.
type guideCache struct {
	mu    sync.Mutex
	items map[string]guideCached
}

type guideCached struct {
	url string
	at  time.Time
}

func (c *guideCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	return item.url, ok && time.Since(item.at) < 24*time.Hour
}

func (c *guideCache) put(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil || len(c.items) >= 256 {
		c.items = map[string]guideCached{}
	}
	c.items[key] = guideCached{value, time.Now()}
}

// guideCommand answers 攻略帮助 and 设置默认攻略.
func (a *App) guideCommand(event *rayleabot.EventContext, command string, args []string) error {
	sources := guideSources
	config, err := a.GuideSettings.Manage("guides.settings", nil)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	settings := config["settings"].(GuideConfig)
	if command == "guide-help" {
		p, sample := a.Game.Prefix, guideSamples
		lines := []string{a.Game.Name + "攻略帮助：", p + sample + "攻略[来源序号]：发送该来源合集中的攻略图，不写序号用默认来源", p + sample + "攻略all：依次发送各来源", p + "更新" + sample + "攻略[来源序号]：重新查找", "合集里没有时可发送" + p + "米游社" + sample + "攻略搜索帖子"}
		if slices.Contains(event.SuperAdmins, event.Event.Actor.ID) {
			lines = append(lines, p+"设置默认攻略<来源序号>：修改默认来源")
		}
		lines = append(lines, "", "攻略来源：")
		for _, source := range sources {
			line := source.ID + ". " + source.Name
			if source.ID == settings.Default {
				line += "（默认）"
			}
			lines = append(lines, line)
		}
		return event.SendText(strings.Join(lines, "\n"))
	}
	if len(args) == 0 {
		return event.SendText("默认攻略设置方式为: \n" + a.Game.Prefix + "设置默认攻略[1-" + strconv.Itoa(len(sources)) + "]")
	}
	if _, err := a.GuideSettings.Manage("guides.configure", map[string]any{"revision": settings.Revision, "default_source": args[0]}); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("默认攻略已设置为: " + args[0])
}

// guideReply is what StarRail-plugin's strategy sends for a character's 攻略:
// the guide images of the sources asked for, or why there are none; nothing
// when the word names no character.
func (a *App) guideReply(ctx context.Context, event *rayleabot.EventContext) []rayleabot.Segment {
	sources := guideSources
	config, err := a.GuideSettings.Manage("guides.settings", nil)
	if err != nil {
		return []rayleabot.Segment{rayleabot.Text(friendlyError(err))}
	}
	settings := config["settings"].(GuideConfig)
	match := guideWord.FindStringSubmatch(strings.TrimSpace(event.Event.Command()))
	if match == nil {
		return nil
	}
	refresh, raw, group := match[1] != "", strings.TrimSpace(match[2]), match[3]
	entry, ok := a.Catalog.Resolve(raw, "character", a.aliasMap(event))
	if !ok {
		return nil
	}
	target := a.guideTargetFor(entry)
	name := strings.TrimSuffix(strings.ReplaceAll(entry.Name, "•", "·"), "Pro")
	picked := []int{}
	if group == "all" || group == "0" {
		for i := range min(len(sources), 4) {
			picked = append(picked, i)
		}
	} else {
		index, _ := strconv.Atoi(cmpOr(group, settings.Default))
		if index < 1 || index > len(sources) {
			return []rayleabot.Segment{rayleabot.Text("超过攻略数量（" + strconv.Itoa(len(sources)) + "）")}
		}
		picked = []int{index - 1}
	}
	segments := []rayleabot.Segment{}
	for _, index := range picked {
		source := sources[index]
		key := a.Game.ID + "/" + source.ID + "/" + entry.ID
		link, cached := a.guides.get(key)
		if !cached || refresh {
			if link, err = a.Content.guidePicture(ctx, source, target); err != nil {
				return []rayleabot.Segment{rayleabot.Text("暂无攻略数据，请稍后再试")}
			}
			if link != "" {
				a.guides.put(key, link)
			}
		}
		switch {
		case link != "":
			segments = append(segments, rayleabot.Image(link))
		case len(picked) == 1:
			return []rayleabot.Segment{rayleabot.Text("暂无" + name + "攻略（" + source.Name + "）\n请尝试其他的攻略来源查询\n" + a.Game.Prefix + "攻略帮助，查看说明\n或发送" + a.Game.Prefix + "米游社" + name + "攻略搜索帖子")}
		default:
			segments = append(segments, rayleabot.Text("暂无"+name+"攻略（"+source.Name+"）\n"))
		}
	}
	return segments
}
