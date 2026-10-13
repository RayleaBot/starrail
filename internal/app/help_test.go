package app

import (
	"slices"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/pluginmeta"
)

// The help word alone answers the menu. With more after it, which the host
// hands over as arguments, the message is not the command, as upstream's
// rule ends at the word: the help another bot answers in a group, whose text
// starts with a prefix and the word, gets no reply.
func TestHelpAnswersTheWordAlone(t *testing.T) {
	a := pluginApp(t)
	host := newSDKHost(t, a, func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
		return nil, "plugin.service_unavailable"
	})
	end, _ := host.groupMessage("*帮助", "帮助")
	if !strings.Contains(terminalText(end), a.Game.Name+"帮助") {
		t.Fatalf("answered %+v", end)
	}
	end, _ = host.groupMessage("星穹铁道帮助\r>下图为指令集，点击查看原图", "帮助", ">下图为指令集，点击查看原图")
	if asObject(end["data"])["handled"] != false || len(host.sent) != 0 {
		t.Fatalf("answered the other bot's help with %+v and %+v", end, host.sent)
	}
}

func TestHelpListsWhatTheRequesterCanRun(t *testing.T) {
	a := pluginApp(t)
	visible := func(id, role string, cfg map[string]any, private bool) []string {
		t.Helper()
		e := &rayleabot.EventContext{Event: rayleabot.Event{EventType: "message.group", SourceProtocol: "onebot11", BotID: "bot", Actor: rayleabot.Actor{ID: id, Role: role}, Target: rayleabot.Target{Type: "group", ID: "g"}}, SuperAdmins: []string{"root"}, Config: cfg}
		if private {
			e.Event.EventType = "message.private"
			e.Event.Target = rayleabot.Target{Type: "private", ID: id}
		}
		out, _, err := a.help(e, "")
		if err != nil {
			t.Fatal(err)
		}
		ids := []string{}
		for _, c := range out["commands"].([]pluginmeta.Command) {
			ids = append(ids, c.ID)
		}
		return ids
	}
	for i, actor := range []struct{ id, role string }{{"member", "member"}, {"admin", "admin"}, {"root", "member"}} {
		ids := visible(actor.id, actor.role, nil, false)
		for id, want := range map[string][3]bool{
			"rank-switch": {false, true, true}, "rank-reset": {false, false, true},
			"challenge-clear": {false, true, true}, "subscribe": {false, true, true},
			"group-settings": {false, true, true}, "alias-add": {true, true, true},
			"cloud-import": {false, false, true}, "help": {true, true, true},
		} {
			if slices.Contains(ids, id) != want[i] {
				t.Errorf("%s sees %s: %v, want %v", actor.id, id, slices.Contains(ids, id), want[i])
			}
		}
	}
	if slices.Contains(visible("member", "member", map[string]any{"alias_permission": 1}, false), "alias-add") {
		t.Error("member can change aliases restricted to administrators")
	}
	if slices.Contains(visible("member", "", nil, true), "alias-remove") {
		t.Error("private chat can change aliases")
	}
}

func TestHelpQueryWritesUsagesWithThePrefix(t *testing.T) {
	host := newSDKHost(t, pluginApp(t), nil)
	end, _ := host.manage("help.query", map[string]any{"query": ""})
	commands := asList(asObject(end["data"])["commands"])
	for _, raw := range commands {
		c := asObject(raw)
		usage := asText(c["usage"])
		for _, form := range strings.Split(usage, " / ") {
			if !strings.HasPrefix(form, "*") {
				t.Errorf("%v: %s", c["id"], usage)
			}
		}
	}
}
