package app

import (
	"fmt"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/gacha"
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
	for _, bot := range []string{"a", "b"} {
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
	if items, _ := a.Reminders.List(); len(items) != 10 {
		t.Fatal("cross-account delete changed tasks")
	}
	if !a.BackgroundSyncs.running("b") {
		t.Fatal("other background sync canceled")
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
