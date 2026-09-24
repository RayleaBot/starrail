package app

import (
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

// photoCommand is miao's CharWiki in its pic mode: a message ending in 照片,
// 写真, 图片 or 图像 after a character miao knows gets a random picture of
// what CharImg.getCardImg reads for it, sent alone, or 暂无图片. Any other
// message is left to other plugins.
func (a *App) photoCommand(event *rayleabot.EventContext, args []string) error {
	entry, ok := Entry{}, false
	if len(args) == 1 {
		entry, ok = a.miaoCharacter(args[0], a.aliasMap(event))
	}
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	media, photos := a.cardImages(entry)
	total := len(media) + len(photos)
	if total == 0 {
		return event.SendText("暂无图片")
	}
	var data []byte
	if pick := rand.IntN(total); pick < len(media) {
		a.Media.mu.Lock()
		v, err := a.Media.read(media[pick].Ref)
		a.Media.mu.Unlock()
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		data = v.Data
	} else {
		file := photos[pick-len(media)]
		raw, err := a.Artwork.Open(file.Source, file.Path)
		if err != nil {
			return event.SendText("图片素材读取失败，请重新下载素材。")
		}
		data = raw
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(data)))
}

// originalImageCommand is 原图: the image the sender last asked for in this
// chat within ten minutes.
func (a *App) originalImageCommand(event *rayleabot.EventContext) error {
	owner := chatOwner(event)
	if !validInteractionOwner(owner) {
		return event.SendText("聊天身份不完整。")
	}
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
}
