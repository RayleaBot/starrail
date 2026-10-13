package app

import (
	"context"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestRenderViewTimeoutStopsFallback(t *testing.T) {
	a := &App{}
	view := View{Title: "体力", Image: &Image{Template: "note"}}
	host := newSDKHost(t, rayleabot.HandlerFunc(func(ctx context.Context, event *rayleabot.EventContext) error {
		return a.sendView(ctx, event, view)
	}), nil)
	host.renderFailure = "platform.render_timeout"

	terminal, actions := host.groupMessage("体力", "体力")
	var templates []string
	for _, action := range actions {
		if action.Name == "render.image" {
			templates = append(templates, asText(action.Data["template"]))
		}
	}
	if len(templates) != 1 || templates[0] != "note" {
		t.Fatalf("rendered templates = %v, want only note", templates)
	}
	if terminal["action"] != "message.send" || terminalText(terminal) != view.Text() {
		t.Fatalf("expected text fallback, got %v", terminal)
	}
}
