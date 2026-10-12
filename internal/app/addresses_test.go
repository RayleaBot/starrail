package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/localdata"
)

func TestRankCacheMigrationKeepsRemainingLifetime(t *testing.T) {
	handler := rayleabot.HandlerFunc(func(ctx context.Context, event *rayleabot.EventContext) error {
		value, err := customRankCache(ctx, event, "message")
		if err != nil {
			return event.Fail("cache_failed", err.Error())
		}
		return event.Result(value)
	})
	host := newSDKHost(t, handler, nil)
	expires := time.Now().Add(time.Minute).UnixMilli()
	cache := map[string]any{"query_id": "query", "character_id": "1001"}
	var migrated map[string]any
	host.storage = func(input map[string]any) (map[string]any, string) {
		key := asText(input["key"])
		if input["operation"] == "get" {
			if key == "ark-query:a:message" {
				return map[string]any{"exists": true, "value": cache, "expires_at_ms": expires}, ""
			}
			if key != "ark-query:onebot11:bot:message" {
				t.Fatal("unexpected cache key", key)
			}
			return map[string]any{"exists": migrated != nil, "value": migrated}, ""
		}
		if input["operation"] != "set" || key != "ark-query:onebot11:bot:message" {
			t.Fatal(input)
		}
		deadline := time.Now().Add(time.Duration(number(input["ttl_seconds"])) * time.Second).UnixMilli()
		if number(input["ttl_seconds"]) <= 0 || deadline > expires {
			t.Fatal("migration extended cache lifetime", input)
		}
		migrated = asObject(input["value"])
		return map[string]any{"expires_at_ms": deadline}, ""
	}
	if end, _ := host.message("query", "query"); end["type"] != "result" || asText(migrated["query_id"]) != "query" {
		t.Fatal(end, migrated)
	}
	if end, actions := host.message("query", "query"); end["type"] != "result" || len(actions) != 1 {
		t.Fatal("canonical cache was not reused", end, actions)
	}
}

func TestAccountIdentityIgnoresAdapter(t *testing.T) {
	one := Subject{"onebot11", "old", "bot", "user"}
	two := Subject{"onebot11", "new", "bot", "user"}
	if !one.same(two) || interactionRef(one) != interactionRef(two) || contentRef("starrail", one, "group") != contentRef("starrail", two, "group") {
		t.Fatal("adapter changed the owner identity")
	}
	for _, other := range []Subject{{"qqofficial", "old", "bot", "user"}, {"onebot11", "old", "other", "user"}, {"onebot11", "old", "bot", "other"}} {
		if one.same(other) || interactionRef(one) == interactionRef(other) {
			t.Fatal("different owners collided")
		}
	}
	event := &rayleabot.EventContext{Event: rayleabot.Event{SourceProtocol: "onebot11", SourceAdapter: "new", BotID: "bot", Actor: rayleabot.Actor{ID: "user"}, Target: rayleabot.Target{Type: "group", ID: "group"}}}
	if !chatOwner(event).same(one) || groupScope(event).BotID != "bot" {
		t.Fatal("chat identity did not use Event.BotID")
	}
	s := &GroupStore{Directory: t.TempDir()}
	old := GroupScope{"onebot11", "old", "bot", "group"}
	if s.file(old) != s.file(groupScope(event)) {
		t.Fatal("adapter changed group filename")
	}
	if err := s.Update(old, func(d *GroupData) error { d.Config.Aliases = map[string]string{"a": "1001"}; return nil }); err != nil {
		t.Fatal(err)
	}
	if config, err := s.Config(groupScope(event)); err != nil || config.Aliases["a"] != "1001" {
		t.Fatal(config, err)
	}
	for _, value := range []any{one, old, sender(event)} {
		raw, err := json.Marshal(value)
		if err != nil || strings.Contains(string(raw), "source_adapter") {
			t.Fatalf("outgoing identity: %s, %v", raw, err)
		}
	}
}

func legacyGroup(t *testing.T, s *GroupStore, data GroupData) string {
	t.Helper()
	// Alias avoids the new writer so the fixture has the old hash and JSON.
	type oldScope GroupScope
	scope := oldScope(data.Scope)
	raw, _ := json.Marshal(scope)
	sum := sha256.Sum256(raw)
	path := filepath.Join(s.Directory, hex.EncodeToString(sum[:])+".json")
	var record map[string]any
	if err := decodeObject(data, &record); err != nil {
		t.Fatal(err)
	}
	record["scope"] = scope
	if err := localdata.Write(path, record); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGroupMigrationPreservesSettingsAndRankings(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	on, off := true, false
	scope := GroupScope{"onebot11", "", "bot", "group"}
	canonical := GroupData{Scope: scope, Revision: 3, Config: GroupConfig{Enabled: &off, Aliases: map[string]string{"same": "new"}}, Rank: []RankEntry{{UID: "u", CharacterID: "c", UpdatedAtMS: 10}}}
	if err := localdata.Write(s.file(scope), canonical); err != nil {
		t.Fatal(err)
	}
	old := GroupData{Scope: GroupScope{"onebot11", "removed", "bot", "group"}, Revision: 4, Config: GroupConfig{Enabled: &on, ImageReplies: &off, Aliases: map[string]string{"same": "old", "kept": "1001"}}, Rank: []RankEntry{{UID: "u", CharacterID: "c", UpdatedAtMS: 20}, {UID: "another", CharacterID: "c", UpdatedAtMS: 5}}, Challenges: []ChallengeEntry{{ActorID: "u", Kind: "challenge", Season: "1", FirstMS: 2, UpdatedMS: 5}}}
	path := legacyGroup(t, s, old)
	if err := s.migrate(nil); err != nil {
		t.Fatal(err)
	}
	data, err := s.Read(scope)
	if err != nil || *data.Config.Enabled || data.Config.ImageReplies == nil || *data.Config.ImageReplies || data.Config.Aliases["same"] != "new" || data.Config.Aliases["kept"] != "1001" || len(data.Rank) != 2 || data.Rank[0].UpdatedAtMS != 20 || len(data.Challenges) != 1 {
		t.Fatalf("lost group data: %+v, %v", data, err)
	}
	if _, err := os.Stat(path + ".legacy"); err != nil {
		t.Fatal("missing conflict backup", err)
	}
	if list, _, err := s.List(0); err != nil || len(list) != 1 {
		t.Fatal("duplicate active group", list, err)
	}
	if err := s.migrate(nil); err != nil {
		t.Fatal(err)
	}
	after, _ := s.Read(scope)
	if !sameTask(data, after) {
		t.Fatal("migration changed data on restart")
	}
}

func TestMigrationRecoversOnlyMatchingLegacyBotAndKeepsUnresolved(t *testing.T) {
	dir := t.TempDir()
	reminders := reminderStore(dir)
	task := Reminder{Ref: "r", Owner: Subject{SourceProtocol: "onebot11", SourceAdapter: "a", ActorID: "u"}, Armed: false, LastCode: "notified", LastAttemptMS: 123}
	type oldOwner Subject
	var oldTask map[string]any
	if err := decodeObject(task, &oldTask); err != nil {
		t.Fatal(err)
	}
	oldTask["owner"] = oldOwner(task.Owner)
	if err := localdata.Write(reminders.file(task.Ref), oldTask); err != nil {
		t.Fatal(err)
	}
	groups := &GroupStore{Directory: filepath.Join(dir, "groups")}
	path := legacyGroup(t, groups, GroupData{Scope: GroupScope{"onebot11", "a", "", "g"}, Config: GroupConfig{Aliases: map[string]string{"saved": "1001"}}})
	bots := []rayleabot.Bot{{SourceProtocol: "qqofficial", SourceAdapter: "a", ID: "wrong"}}
	resolve := func() {
		t.Helper()
		if err := migrateTaskOwners(&reminders.taskFiles, bots, func(t *Reminder) *Subject { return &t.Owner }); err != nil {
			t.Fatal(err)
		}
		if err := groups.migrate(bots); err != nil {
			t.Fatal(err)
		}
	}
	resolve()
	items, _ := reminders.List()
	if len(items) != 1 || items[0].Owner.BotID != "" || items[0].Owner.SourceAdapter != "a" {
		t.Fatal("unresolved owner discarded", items)
	}
	// A normal failed-notification write must retain the recovery hint too.
	if err := reminders.edit("r", func(items *[]Reminder, i int) error { (*items)[i].LastCode = "notification_failed"; return nil }); err != nil {
		t.Fatal(err)
	}
	unresolved, _ := os.ReadFile(reminders.file("r"))
	if strings.Contains(string(unresolved), "source_adapter") || !strings.Contains(string(unresolved), "legacy_adapter") {
		t.Fatal("unresolved rewrite must only keep a migration hint", string(unresolved))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("unresolved group removed", err)
	}
	bots = append(bots, rayleabot.Bot{SourceProtocol: "onebot11", SourceAdapter: "a", ID: "bot"})
	resolve()
	items, _ = reminders.List()
	if items[0].Owner.BotID != "bot" || items[0].Owner.SourceAdapter != "" || items[0].Armed || items[0].LastAttemptMS != 123 || items[0].LastCode != "notification_failed" {
		t.Fatal("task state lost", items)
	}
	data, err := groups.Read(GroupScope{Protocol: "onebot11", BotID: "bot", GroupID: "g"})
	if err != nil || data.Config.Aliases["saved"] != "1001" {
		t.Fatal(data, err)
	}
	raw, _ := os.ReadFile(reminders.file("r"))
	if strings.Contains(string(raw), "source_adapter") {
		t.Fatal("legacy adapter still written", string(raw))
	}
	// Daily syncs use the same migration without losing completion markers.
	syncs := syncTaskStore(dir)
	daily := SyncTask{Ref: "s", Owner: task.Owner, RunDay: "2026-10-11", LastNotificationMS: 456, LastCode: "sync_completed"}
	if err := localdata.Write(syncs.file("s"), daily); err != nil {
		t.Fatal(err)
	}
	if err := migrateTaskOwners(&syncs.taskFiles, bots, func(t *SyncTask) *Subject { return &t.Owner }); err != nil {
		t.Fatal(err)
	}
	saved, _ := syncs.List()
	if saved[0].Owner.BotID != "bot" || saved[0].RunDay != daily.RunDay || saved[0].LastNotificationMS != 456 {
		t.Fatal(saved)
	}
}

func TestGroupMigrationWriteFailureLeavesOldFile(t *testing.T) {
	s := &GroupStore{Directory: t.TempDir()}
	data := GroupData{Scope: GroupScope{"onebot11", "old", "bot", "g"}, Config: GroupConfig{Aliases: map[string]string{"saved": "1001"}}}
	old := legacyGroup(t, s, data)
	// A directory at the destination causes a real filesystem failure.
	if err := os.Mkdir(s.file(data.Scope), 0700); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(nil); err == nil {
		t.Fatal("migration hid a write failure")
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatal("old file was lost", err)
	}
	if err := os.Remove(s.file(data.Scope)); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(nil); err != nil {
		t.Fatal("migration did not resume", err)
	}
}

func TestSubscriptionMigrationResumesSchedulerCleanupWithoutReplaying(t *testing.T) {
	a := pluginApp(t)
	now := time.Now().UnixMilli()
	items := []ContentSubscription{
		{Ref: "game.content.old1", Owner: Subject{"onebot11", "removed", "bot", "u"}, GroupID: "g", Kinds: []string{"announce"}, Sent: map[string]int64{"post": now - 1}, WarnedOn: "2026-10-10"},
		{Ref: "game.content.old2", Owner: Subject{"onebot11", "a", "", "u"}, GroupID: "g", Kinds: []string{"info", "expiry"}, Sent: map[string]int64{"post": now, "other": now}, WarnedOn: "2026-10-11"},
	}
	if err := localdata.Write(a.Subscriptions.Path, items); err != nil {
		t.Fatal(err)
	}
	host := newSDKHost(t, a, func(request rayleabot.ServiceCallRequest, _ bool) (map[string]any, string) {
		t.Fatalf("unexpected accounts request: %+v", request)
		return nil, "plugin.method_not_found"
	})
	for _, item := range items {
		host.jobs[item.Ref] = nil
	}
	host.createFailure = "scheduler.unavailable"
	if end, _ := host.manage("status", nil); end["code"] != "plugin.game_address_migration_failed" {
		t.Fatal(end)
	}
	merged, _ := a.Subscriptions.List()
	ref := contentRef(a.Game.ID, items[0].Owner, "g")
	if len(merged) != 1 || merged[0].Ref != ref || len(merged[0].PreviousRefs) != 2 || merged[0].Sent["post"] != now || merged[0].Sent["other"] != now || merged[0].WarnedOn != "2026-10-11" || len(merged[0].Kinds) != 3 {
		t.Fatalf("lost push state: %+v", merged)
	}
	host.createFailure = ""
	if end, _ := host.manage("status", nil); end["type"] != "result" {
		t.Fatal(end)
	}
	if len(host.jobs) != 1 || len(host.created) != 1 || len(host.deleted) != 2 {
		t.Fatal(host.jobs, host.created, host.deleted)
	}
	merged, _ = a.Subscriptions.List()
	if len(merged[0].PreviousRefs) != 0 {
		t.Fatal("cleanup was not persisted")
	}
	if _, ok := postToPush([]pushPost{{"post", "news", "announce", time.Now().Unix()}}, merged[0], time.Now()); ok {
		t.Fatal("migrated post would be replayed")
	}
	// A delayed old trigger must not send or check the canonical subscription.
	host.jobs[items[0].Ref] = nil
	if end, _ := host.trigger(items[0].Ref); end["type"] != "result" || len(host.sent) != 0 {
		t.Fatal(end, host.sent)
	}
	if !slices.Contains(host.deleted, items[1].Ref) {
		t.Fatal("old scheduler not removed")
	}
}

func TestReminderSendsByAccountAfterAdapterReplacement(t *testing.T) {
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
			var record map[string]any
			if err := localdata.Read(j.a.Reminders.file(ref), &record); err != nil {
				t.Fatal(err)
			}
			asObject(record["owner"])["source_adapter"] = "removed"
			if err := localdata.Write(j.a.Reminders.file(ref), record); err != nil {
				t.Fatal(err)
			}
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

func TestSimulationScopeMigrationKeepsPityAndRequestDedupe(t *testing.T) {
	s := &SimulationStore{Directory: t.TempDir()}
	event := &rayleabot.EventContext{Event: rayleabot.Event{SourceProtocol: "onebot11", SourceAdapter: "a", BotID: "bot", Actor: rayleabot.Actor{ID: "u"}, Target: rayleabot.Target{Type: "private", ID: "u"}}}
	raw, _ := json.Marshal([]string{"onebot11", "a", "bot", "private", "u", "u"})
	state := SimulationState{Total: 42, Pools: map[string]*SimulationPity{"character": {Five: 41}}, Requests: []SimulationBatch{{RequestRef: "last"}}}
	if err := localdata.Write(s.filename(string(raw)), state); err != nil {
		t.Fatal(err)
	}
	scope, err := simulationScope(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.migrateScope(event, scope); err != nil {
		t.Fatal(err)
	}
	var saved SimulationState
	if err := localdata.Read(s.filename(scope), &saved); err != nil || !sameTask(state, saved) {
		t.Fatal(saved, err)
	}
	event.Event.SourceAdapter = "new"
	newScope, _ := simulationScope(event)
	if scope != newScope {
		t.Fatal("simulation is still instance scoped")
	}
}
