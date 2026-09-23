package app

import (
	"context"
	"encoding/base64"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 照片, 老婆 and 图鉴 read the upstream images an administrator downloaded:
// photos the way miao-plugin reads character-img, 图鉴 the way Atlas reads
// its libraries, from the sources game.json declares.

// Pictures are where the downloaded chat images are.
type Pictures struct {
	// Photos are directories of a character's photos; {name} stands for the
	// character.
	Photos []PictureSource `json:"photos"`
	// Atlas are the Atlas libraries, searched in order.
	Atlas []AtlasLibrary `json:"atlas"`
	// Static are commands answered with fixed upstream images, by command.
	Static map[string]StaticPicture `json:"static"`
}

type PictureSource struct {
	Source string   `json:"source"`
	Paths  []string `json:"paths"`
}

// photoSources are the sources of the character photos.
func (p Pictures) photoSources() []string {
	sources := []string{}
	for _, source := range p.Photos {
		sources = append(sources, source.Source)
	}
	return sources
}

// catalogSources are the sources of the 图鉴 pictures, the Atlas libraries.
func (p Pictures) catalogSources() []string {
	sources := []string{}
	for _, library := range p.Atlas {
		sources = append(sources, library.Source)
	}
	return sources
}

// artworkFile is a downloaded file of an artwork source.
type artworkFile struct{ Source, Path string }

var pictureExtensions = []string{".png", ".jpg", ".jpeg", ".webp", ".gif"}

// characterPhotos lists a character's downloaded photos.
func (a *App) characterPhotos(entry Entry) []artworkFile {
	names := []string{entry.Name}
	files := []artworkFile{}
	for _, source := range a.Game.Pictures.Photos {
		for _, pattern := range source.Paths {
			for _, name := range names {
				for _, file := range a.Artwork.List(source.Source, strings.ReplaceAll(pattern, "{name}", name)) {
					if slices.Contains(pictureExtensions, strings.ToLower(path.Ext(file))) {
						files = append(files, artworkFile{source.Source, file})
					}
				}
			}
		}
	}
	return files
}

// aheadOfAtlas are the commands that answer before Atlas sees a message:
// they follow StarRail-plugin rules, which Miao-Yunzai runs ahead of Atlas's
// (priority 10) and which end the messages they answer. Its strategy answers
// a character's 攻略 and passes the message on, so guides hands the message
// to Atlas after answering. Its 绑定uid, 收益曲线, 强度榜, 攻略 and 深渊攻略
// pass the message on too, but Atlas finds nothing for them, so Atlas sees
// them first here.
var aheadOfAtlas = []string{
	// MiaoOrStarrail (priority -1000) and panel (1).
	"character", "panel-list", "panel-refresh", "api-panel-list", "api-panel-refresh", "api-switch", "api-list",
	// alias, gachasimulation, gatcha, hkrpg and srxx (-114514).
	"alias-add", "alias-remove", "aliases", "simulation",
	"gacha-link", "gacha", "gacha-help", "gacha-background",
	"profile", "pay-log", "online",
	"ref-panel", "ref-panel-help", "estimate", "lc-recommend",
	// strategy (1).
	"guides", "guide-help", "guide-default",
	// challenge, gridFight, month, note and rogue (5 while its noteFlag is
	// on, as by default).
	"abyss", "abyss-latest", "challenge", "story", "boss", "peak",
	"currency_wars", "currency-wars-archive", "monthly", "note",
	"rogue", "swarm", "gears", "unknowable", "divergent", "divergent-normal", "divergent-week", "divergent-cycle",
}

// pictureMessage answers the messages Atlas takes: on Miao-Yunzai its rule
// sees every message at priority 10, after StarRail-plugin's rules and ahead
// of miao's and Yunzai's. handled is false for the messages left to the
// plugin's own handlers.
func (a *App) pictureMessage(ctx context.Context, event *rayleabot.EventContext) (handled bool, err error) {
	if event.Event.Target.Type == "group" {
		if config, err := a.Groups.Config(groupScope(event)); err == nil && config.Enabled != nil && !*config.Enabled {
			return false, nil
		}
	}
	// Every message passes here, so the custom aliases are read only when a
	// name is looked up.
	aliases := sync.OnceValue(func() map[string]string { return a.aliasMap(event) })
	// miao's accept checks run before every rule and hand the message on as
	// their own command word, which Atlas does not take.
	run := atlasRun{app: a, owner: chatOwner(event)}
	ended := run.atlas(a.miaoAccept(yunzaiMessage(event), aliases))
	for _, answer := range run.answers {
		if answer.list != nil {
			err = a.postAtlasList(ctx, event, answer)
		} else {
			_, err = post(ctx, event, a.artworkReply(event, answer.picture))
		}
		if err != nil {
			return true, err
		}
	}
	if ended {
		return true, event.Result(map[string]any{"handled": true})
	}
	return false, nil
}

// yunzaiMessage is a chat message as Yunzai's plugins read it (e.msg): the
// text segments, each trimmed, with #星铁 for the prefix the host parsed, as
// Miao-Yunzai writes every Star Rail prefix before any plugin reads it.
func yunzaiMessage(event *rayleabot.EventContext) string {
	if command := event.Event.Command(); command != "" {
		return "#星铁" + strings.Join(append([]string{command}, event.Event.Args()...), " ")
	}
	if len(event.Event.Message.Segments) == 0 {
		return strings.TrimSpace(event.Event.Message.PlainText)
	}
	text := ""
	for _, segment := range event.Event.Message.Segments {
		if segment.Type == "text" {
			text += strings.TrimSpace(asText(segment.Data["text"]))
		}
	}
	return text
}

// pictureHint tells how to download the sources a reply found nothing in.
func (a *App) pictureHint(sources ...string) string {
	missing := []string{}
	for _, source := range sources {
		if !a.Artwork.Ready(source) && !slices.Contains(missing, source) {
			missing = append(missing, source)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "管理员可发送“" + a.Game.Prefix + "素材更新 " + strings.Join(missing, " ") + "”下载图片素材。"
}

// rememberImage keeps the image just sent for 原图.
func (a *App) rememberImage(event *rayleabot.EventContext, ref string) error {
	return a.Interactions.edit(chatOwner(event), func(p *InteractionProfile) error {
		p.LastImage = ref
		p.LastImageMS = time.Now().UnixMilli()
		p.LastImageTargetType = event.Event.Target.Type
		p.LastImageTargetID = event.Event.Target.ID
		return nil
	})
}

// artworkReply is a downloaded image to send, kept for 原图, or the text that
// tells why it cannot be sent.
func (a *App) artworkReply(event *rayleabot.EventContext, file artworkFile) rayleabot.Segment {
	data, err := a.Artwork.Open(file.Source, file.Path)
	if err != nil {
		return rayleabot.Text("图片素材读取失败，请重新下载素材。")
	}
	if err = a.rememberImage(event, "artwork:"+file.Source+"/"+file.Path); err != nil {
		return rayleabot.Text(friendlyError(err))
	}
	return rayleabot.Image("base64://" + base64.StdEncoding.EncodeToString(data))
}

// sendArtwork sends a downloaded image.
func (a *App) sendArtwork(event *rayleabot.EventContext, file artworkFile) error {
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, a.artworkReply(event, file))
}
