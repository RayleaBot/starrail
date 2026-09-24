package app

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/localdata"
)

type InteractionProfile struct {
	Ref                 string  `json:"ref"`
	Owner               Subject `json:"owner"`
	LastImage           string  `json:"last_image,omitempty"`
	LastImageMS         int64   `json:"last_image_ms,omitempty"`
	LastImageTargetType string  `json:"last_image_target_type,omitempty"`
	LastImageTargetID   string  `json:"last_image_target_id,omitempty"`
}
type InteractionStore struct {
	mu   sync.Mutex
	Path string
}

func interactionRef(owner Subject) string {
	raw, _ := json.Marshal(owner)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (s *InteractionStore) read() ([]InteractionProfile, error) {
	items := []InteractionProfile{}
	err := localdata.Read(s.Path, &items)
	return items, err
}
func (s *InteractionStore) List() ([]InteractionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}
func (s *InteractionStore) Get(owner Subject) (InteractionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return InteractionProfile{}, err
	}
	for _, v := range items {
		if v.Owner == owner {
			return v, nil
		}
	}
	return InteractionProfile{Ref: interactionRef(owner), Owner: owner}, nil
}
func (s *InteractionStore) edit(owner Subject, fn func(*InteractionProfile) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(items, func(v InteractionProfile) bool { return v.Owner == owner })
	if i < 0 {
		if len(items) >= 2000 {
			return gameError("interaction_limit", "互动偏好超过2000个主体，请先清理。")
		}
		items = append(items, InteractionProfile{Ref: interactionRef(owner), Owner: owner})
		i = len(items) - 1
	}
	if err = fn(&items[i]); err != nil {
		return err
	}
	return localdata.Write(s.Path, items)
}
func chatOwner(event *rayleabot.EventContext) Subject {
	return Subject{event.Event.SourceProtocol, event.Event.SourceAdapter, event.Bot.ID, event.Event.Actor.ID}
}
func validInteractionOwner(s Subject) bool {
	return s.SourceProtocol != "" && s.SourceAdapter != "" && s.BotID != "" && s.ActorID != "" && len(s.ActorID) <= 256 && len(s.BotID) <= 256
}
func (a *App) sendMedia(event *rayleabot.EventContext, entry MediaEntry) error {
	a.Media.mu.Lock()
	v, err := a.Media.read(entry.Ref)
	a.Media.mu.Unlock()
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if err = a.rememberImage(event, entry.Ref); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Text(entry.Title+"\n来源："+entry.Source), rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(v.Data)))
}
func (a *App) sendCharacterMedia(ctx context.Context, event *rayleabot.EventContext, entry Entry, photoOnly bool) error {
	a.Media.mu.Lock()
	all, err := a.Media.entries()
	a.Media.mu.Unlock()
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	matches := []MediaEntry{}
	for _, v := range all {
		if v.CatalogID == entry.ID && (v.Category == "photo" || v.Category == "character" || v.Category == "birthday") {
			matches = append(matches, v)
		}
	}
	// Imported images and downloaded photos are drawn from together, as
	// miao draws from character-img and its upload directory.
	photos := a.characterPhotos(entry)
	if total := len(matches) + len(photos); total > 0 {
		if pick := rand.IntN(total); pick < len(matches) {
			return a.sendMedia(event, matches[pick])
		} else {
			return a.sendArtwork(event, photos[pick-len(matches)])
		}
	}
	if photoOnly {
		return event.SendText(strings.TrimSpace("暂无图片。" + a.pictureHint(a.Game.Pictures.photoSources()...) + "机器人管理员也可在图库中导入图片并关联角色。"))
	}
	return a.sendView(ctx, event, EntryView(a.Game, entry))
}

func (a *App) interactionCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	owner := chatOwner(event)
	if !validInteractionOwner(owner) {
		return event.SendText("聊天身份不完整。")
	}
	switch command {
	case "original-image":
		p, err := a.Interactions.Get(owner)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if p.LastImage == "" || time.Now().UnixMilli()-p.LastImageMS > 600000 || p.LastImageTargetType != event.Event.Target.Type || p.LastImageTargetID != event.Event.Target.ID {
			return event.SendText("最近十分钟没有由你查询的本地图片。")
		}
		if ref, ok := strings.CutPrefix(p.LastImage, "panel:"); ok {
			return a.sendPanelPicture(event, ref)
		}
		if ref, ok := strings.CutPrefix(p.LastImage, "artwork:"); ok {
			source, file, _ := strings.Cut(ref, "/")
			return a.sendArtwork(event, artworkFile{source, file})
		}
		a.Media.mu.Lock()
		v, err := a.Media.read(p.LastImage)
		a.Media.mu.Unlock()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(v.Data)))
	case "photo":
		entry, ok := a.Catalog.Resolve(strings.Join(args, " "), "character", a.aliasMap(event))
		if len(args) == 0 || !ok {
			return event.SendText("请提供可唯一识别的角色名。")
		}
		return a.sendCharacterMedia(ctx, event, entry, true)
	}
	return event.Result(map[string]any{"handled": false})
}
