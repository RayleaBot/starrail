package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/gacha"
	"github.com/RayleaBot/starrail/internal/localdata"
)

func managementEvent(protocol, bot string) *rayleabot.EventContext {
	payload := map[string]any{}
	if bot != "" {
		payload["page_id"] = "groups"
		payload["bot"] = map[string]any{"source_protocol": protocol, "bot_id": bot}
	}
	return &rayleabot.EventContext{Event: rayleabot.Event{EventType: "management.action", SourceProtocol: "management", SourceAdapter: "management.ui", Payload: payload}}
}

func TestAccountManagementRequiresHostBot(t *testing.T) {
	app := &App{}
	for _, action := range []string{"accounts.list", "accounts.select", "query", "groups.list", "groups.set", "groups.clear", "challenge.list", "challenge.clear", "content.subscription.list", "content.subscription.remove", "reminder.list", "reminder.create", "reminder.remove", "signin.status", "signin.run", "signin.task.create", "signin.task.remove", "monthly.list", "monthly.task.list", "challenge.reminder.create", "community.task.list", "gacha.sync.start", "gacha.sync.cancel", "gacha.task.list", "gacha.task.cancel", "billing.sync.cancel", "panel.history.export", "cloud.archive.export.close", "cloud.ocr.compare"} {
		t.Run(action, func(t *testing.T) {
			_, err := app.Manage(t.Context(), managementEvent("", ""), action, map[string]any{"bot": map[string]any{"source_protocol": "onebot11", "bot_id": "forged"}})
			if err == nil || PublicError(err).Code != "platform.invalid_request" {
				t.Fatal(err)
			}
		})
	}
	for _, key := range []string{"owner", "subject", "bot", "scope"} {
		_, err := app.Manage(t.Context(), managementEvent("onebot11", "a"), "groups.set", map[string]any{key: map[string]any{"source_protocol": "onebot11", "bot_id": "b"}})
		if err == nil || PublicError(err).Code != "platform.invalid_request" {
			t.Fatal(key, err)
		}
	}
}

func TestManagementGroupFilteringAndConcurrentWrites(t *testing.T) {
	a := &App{Groups: &GroupStore{Directory: t.TempDir()}}
	for _, bot := range []string{"a", "b"} {
		for i := range 51 {
			scope := GroupScope{Protocol: "onebot11", BotID: bot, GroupID: fmt.Sprint(i)}
			if err := a.Groups.Update(scope, func(data *GroupData) error { data.Rank = []RankEntry{{UID: bot}}; return nil }); err != nil {
				t.Fatal(err)
			}
		}
	}
	legacyGroup(t, a.Groups, GroupData{Scope: GroupScope{Protocol: "onebot11", Adapter: "lost", GroupID: "legacy"}})
	for page, count := range []int{50, 1} {
		result, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "groups.list", map[string]any{"page": page})
		if err != nil {
			t.Fatal(err)
		}
		items := result["items"].([]map[string]any)
		if len(items) != count {
			t.Fatal(page, len(items))
		}
		for _, item := range items {
			if item["scope"].(GroupScope).BotID != "a" {
				t.Fatal(item)
			}
		}
	}
	var wg sync.WaitGroup
	for _, bot := range []string{"a", "b"} {
		wg.Go(func() {
			_, err := a.Manage(t.Context(), managementEvent("onebot11", bot), "groups.set", map[string]any{"group_id": "0", "revision": 1, "config": map[string]any{"enabled": bot == "a"}})
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	for _, bot := range []string{"a", "b"} {
		data, err := a.Groups.Read(GroupScope{Protocol: "onebot11", BotID: bot, GroupID: "0"})
		if err != nil || data.Config.Enabled == nil || *data.Config.Enabled != (bot == "a") || data.Rank[0].UID != bot {
			t.Fatal(data, err)
		}
	}
	_, err := a.Manage(t.Context(), managementEvent("qqofficial", "a"), "groups.clear", map[string]any{"scope": GroupScope{Protocol: "onebot11", BotID: "a", GroupID: "0"}, "revision": 2})
	if err == nil {
		t.Fatal("protocol mismatch accepted")
	}
}

func TestManagementTaskIsolation(t *testing.T) {
	a := pluginApp(t)
	for _, bot := range []string{"a", "b", ""} {
		owner := Subject{SourceProtocol: "onebot11", BotID: bot, ActorID: "u"}
		for _, kind := range []string{"", "signin", "monthly", "community", "challenge"} {
			if err := seedReminders(a.Reminders, []Reminder{{Ref: bot + ":" + kind, Kind: kind, Owner: owner}}); err != nil {
				t.Fatal(err)
			}
		}
		if err := a.SyncTasks.edit("", func(items *[]SyncTask, _ int) error {
			*items = append(*items, SyncTask{Ref: "sync:" + bot, Owner: owner})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if err := a.Subscriptions.edit("", func(items *[]ContentSubscription, _ int) error {
			*items = append(*items, ContentSubscription{Ref: "push:" + bot, Owner: owner})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		a.BackgroundSyncs.begin(BackgroundSync{Ref: bot, Owner: owner, Role: Role{UID: bot}, State: "running"})
	}
	event := managementEvent("onebot11", "a")
	for _, action := range []string{"reminder.list", "signin.task.list", "monthly.task.list", "community.task.list", "challenge.reminder.list"} {
		result, err := a.Manage(t.Context(), event, action, nil)
		if err != nil {
			t.Fatal(err)
		}
		items := result["items"].([]Reminder)
		if len(items) != 1 || items[0].Owner.BotID != "a" {
			t.Fatal(action, items)
		}
	}
	result, err := a.Manage(t.Context(), event, "gacha.task.list", nil)
	if err != nil || len(result["items"].([]SyncTask)) != 1 || len(result["syncs"].([]BackgroundSync)) != 1 {
		t.Fatal(result, err)
	}
	result, err = a.Manage(t.Context(), event, "content.subscription.list", nil)
	if err != nil || len(result["items"].([]ContentSubscription)) != 1 {
		t.Fatal(result, err)
	}
	for action, ref := range map[string]string{"reminder.remove": "b:", "signin.task.remove": "b:signin", "monthly.task.remove": "b:monthly", "community.task.remove": "b:community", "challenge.reminder.remove": "b:challenge", "gacha.task.remove": "sync:b", "gacha.task.run": "sync:b", "gacha.task.cancel": "b", "content.subscription.remove": "push:b"} {
		_, err := a.Manage(t.Context(), event, action, map[string]any{"ref": ref, "confirm": true})
		if err == nil || PublicError(err).Code != "platform.invalid_request" {
			t.Fatal(action, err)
		}
	}
	if items, _ := a.Reminders.List(); len(items) != 15 {
		t.Fatal("cross-account delete changed tasks")
	}
	if !a.BackgroundSyncs.running("b") {
		t.Fatal("other background sync canceled")
	}
}

func TestHistoricalOwnershipAndMerge(t *testing.T) {
	a := pluginApp(t)
	old := GroupData{Scope: GroupScope{Protocol: "onebot11", Adapter: "removed", GroupID: "g"}, Config: GroupConfig{Aliases: map[string]string{"old": "1001"}}, Rank: []RankEntry{{UID: "old"}}}
	file := legacyGroup(t, a.Groups, old)
	scope := GroupScope{Protocol: "onebot11", BotID: "a", GroupID: "g"}
	if err := a.Groups.Update(scope, func(data *GroupData) error { data.Config.Aliases["current"] = "1002"; return nil }); err != nil {
		t.Fatal(err)
	}
	list, err := a.listHistory()
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(list, func(item historyRecord) bool { return item.Ref == filepath.Base(file) })
	item := list[i]
	input := map[string]any{"kind": item.Kind, "ref": item.Ref, "version": item.Version, "confirm": true}
	if _, err := a.Manage(t.Context(), managementEvent("", ""), "history.assign", input); err == nil {
		t.Fatal("assignment without host bot")
	}
	if _, err := a.Manage(t.Context(), managementEvent("qqofficial", "a"), "history.assign", input); err == nil {
		t.Fatal("QQ official reassignment")
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "history.assign", input); err != nil {
		t.Fatal(err)
	}
	data, _ := a.Groups.Read(scope)
	if len(data.Config.Aliases) != 2 || len(data.Rank) != 1 {
		t.Fatal(data)
	}
	list, _ = a.listHistory()
	if len(list) != 1 || list[0].Owner.BotID != "a" {
		t.Fatal(list)
	}
	item = list[0]
	input = map[string]any{"kind": item.Kind, "ref": item.Ref, "version": item.Version, "confirm": true}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "b"), "history.assign", input); err == nil {
		t.Fatal("complete identity moved")
	}
	if _, err := a.Manage(t.Context(), managementEvent("", ""), "history.get", input); err != nil {
		t.Fatal(err)
	}
	if err := a.Groups.Update(scope, func(data *GroupData) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Manage(t.Context(), managementEvent("", ""), "history.remove", input); err == nil || PublicError(err).Code != "plugin.game_history_changed" {
		t.Fatal(err)
	}
	list, _ = a.listHistory()
	input["version"] = list[0].Version
	if _, err := a.Manage(t.Context(), managementEvent("", ""), "history.remove", input); err != nil {
		t.Fatal(err)
	}
	if list, _ := a.listHistory(); len(list) != 0 {
		t.Fatal(list)
	}
}

func TestHistoricalAssignmentPausesTasksAndRejectsOfficialOwners(t *testing.T) {
	a := pluginApp(t)
	for _, protocol := range []string{"onebot11", "qqofficial"} {
		if err := seedReminders(a.Reminders, []Reminder{{Ref: protocol, Owner: Subject{SourceProtocol: protocol, ActorID: "user"}, Enabled: true}}); err != nil {
			t.Fatal(err)
		}
	}
	items, _ := a.listHistory()
	for _, item := range items {
		_, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "history.assign", map[string]any{"kind": item.Kind, "ref": item.Ref, "version": item.Version, "confirm": true})
		if (err == nil) != (item.Owner.SourceProtocol == "onebot11") {
			t.Fatal(item, err)
		}
	}
	tasks, _ := a.Reminders.List()
	if tasks[0].Owner.BotID != "a" || tasks[0].Enabled || tasks[1].Owner.BotID != "" || !tasks[1].Enabled {
		t.Fatal(tasks)
	}
}

func TestManagementServiceErrorsDoNotAlterAnotherAccountsSessions(t *testing.T) {
	a := pluginApp(t)
	choice := Selection{AccountRef: "other", RoleRef: "role"}
	role := Role{Ref: "role", Game: "starrail", UID: "600000001", Region: "prod_official_usa"}
	ref, err := a.CloudTransfers.start(cloudTransfer{Selection: choice, Provider: "provider", Role: role})
	if err != nil {
		t.Fatal(err)
	}
	syncInfo, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, role.UID, role.Region, false)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Syncs.Forget(syncInfo.Ref)
	job := &BillingJob{Ref: "billing", Selection: choice, Provider: "provider", Role: role, timer: time.NewTimer(time.Hour)}
	a.BillingJobs.items = map[string]*BillingJob{job.Ref: job}
	seen := 0
	host := newSDKHost(t, a, func(request rayleabot.ServiceCallRequest, _ bool) (map[string]any, string) {
		seen++
		if request.Params["bot"] != nil || request.Params["bot_id"] != nil || request.Params["subject"] != nil {
			t.Fatal("caller forged service origin", request)
		}
		return nil, "plugin.account_subject_denied"
	})
	for action, ref := range map[string]string{"cloud.archive.import.cancel": ref, "billing.sync.cancel": job.Ref, "gacha.sync.cancel": syncInfo.Ref, "accounts.list": ""} {
		end, _ := host.manage(action, map[string]any{"ref": ref})
		if end["code"] != "plugin.account_subject_denied" {
			t.Fatal(action, end)
		}
	}
	if seen != 4 || a.CloudTransfers.items[ref] == nil || a.BillingJobs.items[job.Ref] == nil {
		t.Fatal("denied action changed session")
	}
	if _, err := a.Syncs.Choice(syncInfo.Ref); err != nil {
		t.Fatal(err)
	}
}

func TestHistoricalListIncludesOpaqueArchives(t *testing.T) {
	a := pluginApp(t)
	if err := localdata.Write(a.Monthly.file("provider", Selection{AccountRef: "lost"}), MonthlyArchive{Items: []MonthlySnapshot{{Month: "2026-01"}}}); err != nil {
		t.Fatal(err)
	}
	items, err := a.listHistory()
	if err != nil || len(items) != 1 || items[0].Kind != "monthly" || items[0].Owner.BotID != "" {
		t.Fatal(items, err)
	}
}

func TestManagementCloudResultsKeepTheirAccount(t *testing.T) {
	bot := rayleabot.BotAccount{SourceProtocol: "onebot11", BotID: "a"}
	canceled := false
	a := &App{Cloud: CloudClient{jobs: map[string]*CloudJob{"job": {Ref: "job", bot: bot, State: "completed", CreatedAtMS: time.Now().UnixMilli(), cancel: func() { canceled = true }}}}}
	for _, event := range []*rayleabot.EventContext{managementEvent("", ""), managementEvent("onebot11", "b"), managementEvent("qqofficial", "a")} {
		for _, action := range []string{"cloud.poll", "cloud.cancel"} {
			_, err := a.Manage(t.Context(), event, action, map[string]any{"ref": "job"})
			if err == nil || PublicError(err).Code != "platform.invalid_request" {
				t.Fatal(action, err)
			}
		}
	}
	if canceled {
		t.Fatal("another account canceled the job")
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "cloud.poll", map[string]any{"ref": "job"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Manage(t.Context(), managementEvent("onebot11", "a"), "cloud.cancel", map[string]any{"ref": "job"}); err != nil || !canceled {
		t.Fatal(err)
	}
}

func TestHistoricalPushAssignmentKeepsOtherAccountsLatestData(t *testing.T) {
	a := pluginApp(t)
	owner := Subject{SourceProtocol: "onebot11", BotID: "b", ActorID: "u"}
	other := ContentSubscription{Ref: contentRef(a.Game.ID, owner, "group"), Owner: owner, GroupID: "group", Kinds: []string{"info"}, Sent: map[string]int64{}}
	legacy := ContentSubscription{Ref: "legacy", Owner: Subject{SourceProtocol: "onebot11", SourceAdapter: "removed", ActorID: "u"}, GroupID: "group", Kinds: []string{"announce"}}
	if err := a.Subscriptions.edit("", func(items *[]ContentSubscription, _ int) error { *items = append(*items, legacy, other); return nil }); err != nil {
		t.Fatal(err)
	}
	listed, err := a.listHistory()
	if err != nil {
		t.Fatal(err)
	}
	item := listed[slices.IndexFunc(listed, func(item historyRecord) bool { return item.Ref == legacy.Ref })]
	if err := a.Subscriptions.edit(other.Ref, func(items *[]ContentSubscription, i int) error { (*items)[i].Sent["new"] = 42; return nil }); err != nil {
		t.Fatal(err)
	}
	host := newSDKHost(t, a, nil)
	bot := &rayleabot.BotAccount{SourceProtocol: "onebot11", BotID: "a"}
	end, _ := host.await(host.postManageFor(bot, "history.assign", map[string]any{"kind": item.Kind, "ref": item.Ref, "version": item.Version, "confirm": true}))
	if end["type"] != "result" {
		t.Fatal(end)
	}
	rows, err := a.Subscriptions.List()
	if err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	i := slices.IndexFunc(rows, func(item ContentSubscription) bool { return item.Owner.BotID == "a" })
	if i < 0 || len(rows[i].PreviousRefs) != 0 || !slices.Contains(host.deleted, legacy.Ref) {
		t.Fatal(rows, host.deleted)
	}
	ref := rows[i].Ref
	if _, ok := host.jobs[ref]; !ok {
		t.Fatal("new owner has no scheduler task")
	}
	end, _ = host.await(host.postManageFor(bot, "content.subscription.remove", map[string]any{"ref": ref, "confirm": true}))
	if end["type"] != "result" {
		t.Fatal(end)
	}
	rows, err = a.Subscriptions.List()
	if err != nil || len(rows) != 1 || rows[0].Owner.BotID != "b" || rows[0].Sent["new"] != 42 {
		t.Fatal(rows, err)
	}
}
