package app

import (
	"encoding/json"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestAccountIdentityIgnoresAdapter(t *testing.T) {
	event := &rayleabot.EventContext{Event: rayleabot.Event{SourceProtocol: "onebot11", SourceAdapter: "old", BotID: "bot", Actor: rayleabot.Actor{ID: "user"}, Target: rayleabot.Target{Type: "group", ID: "group"}}}
	one := chatOwner(event)
	scope := groupScope(event)
	event.Event.SourceAdapter = "new"
	two := chatOwner(event)
	if one != two || interactionRef(one) != interactionRef(two) || contentRef("starrail", one, "group") != contentRef("starrail", two, "group") {
		t.Fatal("adapter changed the owner identity")
	}
	for _, other := range []Subject{{"qqofficial", "bot", "user"}, {"onebot11", "other", "user"}, {"onebot11", "bot", "other"}} {
		if one == other || interactionRef(one) == interactionRef(other) {
			t.Fatal("different owners collided")
		}
	}
	if chatOwner(event) != (Subject{"onebot11", "bot", "user"}) || groupScope(event).BotID != "bot" {
		t.Fatal("chat identity did not use Event.BotID")
	}
	s := &GroupStore{Directory: t.TempDir()}
	if s.file(scope) != s.file(groupScope(event)) {
		t.Fatal("adapter changed group filename")
	}
	if err := s.Update(scope, func(d *GroupData) error { d.Config.Aliases = map[string]string{"a": "1001"}; return nil }); err != nil {
		t.Fatal(err)
	}
	if config, err := s.Config(groupScope(event)); err != nil || config.Aliases["a"] != "1001" {
		t.Fatal(config, err)
	}
	for _, value := range []any{one, scope, sender(event)} {
		raw, err := json.Marshal(value)
		if err != nil || strings.Contains(string(raw), "source_adapter") {
			t.Fatalf("outgoing identity: %s, %v", raw, err)
		}
	}
}

func TestReminderSendsByAccount(t *testing.T) {
	for _, failure := range []string{"", "adapter.transport_unavailable"} {
		t.Run(failure, func(t *testing.T) {
			j := newJobsHost(t)
			j.accounts.data = map[string]func(map[string]any) (map[string]any, string){"starrail.note": func(map[string]any) (map[string]any, string) {
				return map[string]any{"current_stamina": 200, "max_stamina": 240}, ""
			}}
			if end, _ := j.host.manage("reminder.create", map[string]any{"account_ref": "account", "role_ref": "role", "threshold": 80, "days": 30}); end["type"] != "result" {
				t.Fatal(end)
			}
			ref := j.host.job("game.reminder.")
			j.host.sendFailure = failure
			end, actions := j.host.trigger(ref)
			if end["type"] != "result" || len(j.host.sent) != 1 {
				t.Fatal(end, j.host.sent)
			}
			for _, action := range actions {
				if action.Name != "message.send" {
					continue
				}
				if action.Data["source_protocol"] != "onebot11" || action.Data["bot_id"] != "bot" || action.Data["target_type"] != "private" || action.Data["target_id"] != "u" {
					t.Fatal(action)
				}
				if _, ok := action.Data["source_adapter"]; ok {
					t.Fatal("adapter in request", action)
				}
			}
			tasks, _ := j.a.Reminders.List()
			want := "notified"
			if failure != "" {
				want = "notification_failed"
			}
			if tasks[0].Armed || tasks[0].LastCode != want {
				t.Fatal(tasks)
			}
			j.host.trigger(ref)
			if len(j.host.sent) != 1 {
				t.Fatal("notification retried")
			}
		})
	}
}
