package app

import (
	"cmp"
	"context"
	"encoding/base64"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// StaticPicture is a command upstream answers with images it ships, as
// StarRail-plugin's 收益曲线: its parts in order, each an optional caption
// and an image of Source. Forward names the forwarded message upstream
// sends them in.
type StaticPicture struct {
	Source  string       `json:"source"`
	Forward string       `json:"forward,omitempty"`
	Parts   []StaticPart `json:"parts"`
}

type StaticPart struct {
	Text string `json:"text,omitempty"`
	Path string `json:"path,omitempty"`
}

// staticCommand sends a static picture command's parts.
func (a *App) staticCommand(ctx context.Context, event *rayleabot.EventContext, static StaticPicture) error {
	if !a.Artwork.Ready(static.Source) {
		return event.SendText(strings.TrimSpace("暂无图片素材。" + a.pictureHint(static.Source)))
	}
	parts := a.staticParts(static)
	if len(parts) == 0 {
		return event.SendText("图片素材不完整，请管理员重新下载素材。")
	}
	if static.Forward != "" {
		return a.sendForward(ctx, event, parts)
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, slices.Concat(parts...)...)
}

// staticParts reads a static picture's parts.
func (a *App) staticParts(static StaticPicture) [][]rayleabot.Segment {
	parts := [][]rayleabot.Segment{}
	for _, part := range static.Parts {
		segments := []rayleabot.Segment{}
		if part.Text != "" {
			segments = append(segments, rayleabot.Text(part.Text))
		}
		if part.Path != "" {
			if data, err := a.Artwork.Open(static.Source, part.Path); err == nil {
				segments = append(segments, rayleabot.Image("base64://"+base64.StdEncoding.EncodeToString(data)))
			}
		}
		if len(segments) > 0 {
			parts = append(parts, segments)
		}
	}
	return parts
}

// sendForward sends parts as one forwarded message, as upstream's
// makeForwardMsg, or as one ordinary message where forwarding is not
// available.
func (a *App) sendForward(ctx context.Context, event *rayleabot.EventContext, parts [][]rayleabot.Segment) error {
	if a.forward(ctx, event, parts) {
		return event.Result(map[string]any{"handled": true})
	}
	return event.Send(event.Event.Target.Type, event.Event.Target.ID, slices.Concat(parts...)...)
}

// forward sends parts as one forwarded message without ending the event;
// false when the chat platform does not take it.
func (a *App) forward(ctx context.Context, event *rayleabot.EventContext, parts [][]rayleabot.Segment) bool {
	if event.Event.SourceProtocol != "onebot11" {
		return false
	}
	messages := []rayleabot.ForwardMessage{}
	for _, part := range parts {
		content := []any{}
		for _, segment := range part {
			content = append(content, map[string]any{"type": segment.Type, "data": segment.Data})
		}
		messages = append(messages, rayleabot.ForwardMessage{"type": "node", "data": map[string]any{"user_id": event.Event.BotID, "nickname": cmp.Or(event.Bot.Nickname, a.Game.Name), "content": content}})
	}
	_, err := event.Actions().ForOneBotAccount(event.Event.BotID).MessageForwardSend(ctx, rayleabot.MessageForwardSendRequest{TargetType: rayleabot.ConversationType(event.Event.Target.Type), TargetID: event.Event.Target.ID, Messages: messages})
	return err == nil
}
