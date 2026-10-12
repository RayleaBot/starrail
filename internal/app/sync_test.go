package app

import (
	"strconv"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// syncRead is a gacha record request: when it came and what it asked for.
type syncRead struct {
	at        time.Time
	pool, end string
	page      int
}

// syncAccounts answers the account plugin for user "u"'s account, which
// holds an overseas role whose official records are 190 character event
// warps: ten pages of that pool and one empty page of each other. It grants
// the delegation "d1", which only a trigger's requests carry, and counts
// those as delegated. onRead sees each record request after it is counted.
type syncAccounts struct {
	t         *testing.T
	clock     *fakeClock
	reads     []syncRead
	delegated int
	onRead    func(n int)
}

func (s *syncAccounts) answer(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
	role := map[string]any{"ref": "role", "game": "starrail", "uid": "600000001", "region": "prod_official_usa", "nickname": "开拓者", "level": 70}
	switch request.Method {
	case "roles":
		return map[string]any{"roles": []any{role}}, ""
	case "list":
		owner := map[string]any{"source_protocol": "onebot11", "bot_id": "bot", "actor_id": "u"}
		return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "account_label": "账号", "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "delegation.create":
		if request.Params["operation"] != "starrail.gacha" || request.Params["days"] != float64(30) {
			s.t.Errorf("delegation requested with %v", request.Params)
		}
		return map[string]any{"delegation": map[string]any{"ref": "d1", "expires_at_ms": time.Now().Add(30 * 24 * time.Hour).UnixMilli()}}, ""
	case "delegation.revoke":
		return map[string]any{}, ""
	case "execute":
		// A management action reads with its own origin, a trigger under the
		// delegation.
		if scheduled != (request.Params["delegation_ref"] == "d1") || !scheduled && request.Params["delegation_ref"] != nil || request.Params["operation"] != "starrail.gacha" {
			s.t.Errorf("records requested with %v", request.Params)
		}
		if scheduled {
			s.delegated++
		}
		input := asObject(request.Params["input"])
		pool, end := asText(input["gacha_type"]), asText(input["end_id"])
		page, _ := strconv.Atoi(asText(input["page"]))
		s.reads = append(s.reads, syncRead{s.clock.Now(), pool, end, page})
		if s.onRead != nil {
			s.onRead(len(s.reads))
		}
		records := []any{}
		if pool == "11" {
			first := 1
			if id, _ := strconv.Atoi(end); id != 0 {
				first = 5000 - id + 1
			}
			for k := first; k <= 190 && len(records) < 20; k++ {
				at := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Add(-time.Duration(k) * time.Minute)
				records = append(records, map[string]any{"uid": "600000001", "gacha_id": "2003", "gacha_type": "11", "item_id": "1308", "count": "1", "time": at.Format(time.DateTime), "name": "黄泉", "item_type": "角色", "rank_type": "5", "id": strconv.Itoa(5000 - k)})
			}
		}
		return map[string]any{"operation": "starrail.gacha", "role": role, "data": map[string]any{"list": records, "region": "prod_official_usa", "region_time_zone": -5}}, ""
	}
	s.t.Errorf("unexpected %s", request.Method)
	return nil, "plugin.method_not_found"
}

// syncHost is the plugin with syncAccounts on the SDK host, its clock
// starting at the current time.
func syncHost(t *testing.T) (*App, *sdkHost, *syncAccounts) {
	t.Helper()
	a := pluginApp(t)
	clock := &fakeClock{at: time.Now()}
	a.clock = clock
	accounts := &syncAccounts{t: t, clock: clock}
	return a, newSDKHost(t, a, accounts.answer), accounts
}

// startSync starts a background sync of the role on the management page.
func startSync(h *sdkHost, input map[string]any) (map[string]any, []hostAction) {
	h.t.Helper()
	input["account_ref"], input["role_ref"], input["confirm"] = "account", "role", true
	return h.manage("gacha.task.start", input)
}

// A background sync started on the management page answers the page as it
// moves to the background, then reads every pool in that one event, a
// second or more between pages, merges the records and tells the account's
// chat user.
func TestBackgroundSyncReadsEveryPageInItsEvent(t *testing.T) {
	a, host, accounts := syncHost(t)
	end, actions := startSync(host, map[string]any{"notify": true})
	if end["type"] != "result" || len(host.detached) != 1 {
		t.Fatalf("the sync ended with %v after %d detaches", end, len(host.detached))
	}
	answered := asObject(asObject(host.detached[0]["result"])["task"])
	if answered["state"] != "running" || asObject(answered["role"])["uid"] != "600000001" {
		t.Fatalf("the page received %v", answered)
	}
	if !detachedBeforeReading(actions) {
		t.Fatalf("records were read before the action moved to the background: %v", actions)
	}
	if len(accounts.reads) != 15 {
		t.Fatalf("read %d pages", len(accounts.reads))
	}
	for i := 1; i < len(accounts.reads); i++ {
		if gap := accounts.reads[i].at.Sub(accounts.reads[i-1].at); gap < syncPageGap {
			t.Fatalf("read %d came %v after the previous", i, gap)
		}
	}
	if archive, err := a.Gacha.Read("600000001", "prod_official_usa"); err != nil || len(archive.Records) != 190 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
	if len(host.sent) != 1 || host.sent[0].TargetType != "private" || host.sent[0].TargetID != "u" || !strings.HasPrefix(sentText(host.sent[0].Message), "抽卡后台同步完成\n开拓者 · 600000001\n新增 190 条") {
		t.Fatalf("told %v", host.sent)
	}
	listed, _ := host.manage("gacha.task.list", map[string]any{})
	items := asList(asObject(listed["data"])["syncs"])
	if len(items) != 1 || asObject(items[0])["state"] != "completed" || asObject(items[0])["last_code"] != "sync_completed" || asObject(asObject(asObject(items[0])["progress"])["result"])["added"] != float64(190) {
		t.Fatalf("listed %v", items)
	}
}

// A background sync the page cancels stops before its next page and leaves
// the archive as it was.
func TestBackgroundSyncStopsWhenCanceled(t *testing.T) {
	a, host, accounts := syncHost(t)
	accounts.onRead = func(n int) {
		if n == 3 {
			items := a.BackgroundSyncs.list()
			if _, err := a.backgroundSyncAction(t.Context(), managementEvent("onebot11", "bot"), "gacha.task.cancel", map[string]any{"ref": items[0].Ref}); err != nil {
				t.Error(err)
			}
		}
	}
	if end, _ := startSync(host, map[string]any{"notify": true}); end["type"] != "result" {
		t.Fatalf("the canceled sync ended with %v", end)
	}
	items := a.BackgroundSyncs.list()
	if len(accounts.reads) != 3 || len(items) != 1 || items[0].State != "canceled" || items[0].LastCode != "sync_canceled" || len(host.sent) != 0 {
		t.Fatalf("read %d pages, listed %+v, told %v", len(accounts.reads), items, host.sent)
	}
	if _, err := a.Gacha.Read("600000001", "prod_official_usa"); err == nil {
		t.Fatal("the canceled sync kept records")
	}
}

// A sync the host would not move to the background, as the plugin holds its
// limit of background events, tells the page so and reads nothing.
func TestBackgroundSyncRefusedByTheHostTellsThePage(t *testing.T) {
	a, host, accounts := syncHost(t)
	host.busy = true
	end, _ := startSync(host, map[string]any{})
	if end["type"] != "error" || end["code"] != "plugin.game_background_busy" || end["message"] != "后台处理的任务已达上限，请稍后再试。" {
		t.Fatalf("the refused sync ended with %v", end)
	}
	if len(accounts.reads) != 0 || len(a.BackgroundSyncs.list()) != 0 {
		t.Fatalf("the refused sync read %d pages and listed %v", len(accounts.reads), a.BackgroundSyncs.list())
	}
	host.busy = false
	if end, _ := startSync(host, map[string]any{}); end["type"] != "result" || len(accounts.reads) != 15 {
		t.Fatalf("the sync after the refusal ended with %v after %d pages", end, len(accounts.reads))
	}
}

// A daily sync created on the management page grants its delegation and
// schedules a job checked each minute. Its first trigger after the hour moves
// to the background and reads every pool under the delegation, a second or
// more between pages, then tells the account's chat user; later triggers that
// day do nothing, until the page runs it again.
func TestDailySyncReadsEveryPageOnItsTrigger(t *testing.T) {
	a, host, accounts := syncHost(t)
	end, _ := host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "hour": 0, "days": 30, "notify": true, "confirm": true})
	if end["type"] != "result" {
		t.Fatalf("gacha.task.create ended with %v", end)
	}
	ref := host.job("game.sync.")
	if created := host.created[len(host.created)-1]; created["cron"] != "* * * * *" || created["payload"] != nil {
		t.Fatalf("created %v", created)
	}
	end, actions := host.trigger(ref)
	if end["type"] != "result" || !detachedBeforeReading(actions) || accounts.delegated != 15 {
		t.Fatalf("the trigger ended with %v after %d delegated reads", end, accounts.delegated)
	}
	for i := 1; i < len(accounts.reads); i++ {
		if gap := accounts.reads[i].at.Sub(accounts.reads[i-1].at); gap < syncTaskGap {
			t.Fatalf("read %d came %v after the previous", i, gap)
		}
	}
	if archive, err := a.Gacha.Read("600000001", "prod_official_usa"); err != nil || len(archive.Records) != 190 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
	if len(host.sent) != 1 || host.sent[0].TargetID != "u" || !strings.HasPrefix(sentText(host.sent[0].Message), "抽卡后台同步完成\n开拓者 · 600000001\n新增 190 条") {
		t.Fatalf("told %v", host.sent)
	}
	task := dailyTask(t, a)
	if _, _, next := syncTaskTime(time.Now(), 0); task.State != "waiting" || task.LastCode != "sync_completed" || task.NextCheckMS != next || task.Progress.Result == nil || task.Progress.Pages != 15 {
		t.Fatalf("task %+v", task)
	}
	// The day's round is done.
	if end, actions := host.trigger(ref); end["type"] != "result" || len(actions) != 0 || accounts.delegated != 15 {
		t.Fatalf("the same day's trigger ended with %v after %v", end, actions)
	}
	if end, _ := host.manage("gacha.task.run", map[string]any{"ref": ref, "confirm": true}); asObject(end["data"])["queued"] != true {
		t.Fatalf("gacha.task.run ended with %v", end)
	}
	// The round reads only new records: a page of each pool.
	host.trigger(ref)
	if task := dailyTask(t, a); accounts.delegated != 21 || task.State != "waiting" || task.Progress.Result.Added != 0 {
		t.Fatalf("the run again read %d pages, task %+v", accounts.delegated-15, task)
	}
}

// A due daily sync the host would not move to the background, as the plugin
// holds its limit of background events, reads nothing and is tried again by
// the next trigger.
func TestDailySyncRefusedByTheHostWaitsForTheNextTrigger(t *testing.T) {
	a, host, accounts := syncHost(t)
	host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "hour": 0, "days": 30, "confirm": true})
	ref := host.job("game.sync.")
	host.busy = true
	if end, _ := host.trigger(ref); end["type"] != "result" || len(accounts.reads) != 0 {
		t.Fatalf("the refused trigger ended with %v after %d reads", end, len(accounts.reads))
	}
	host.busy = false
	host.trigger(ref)
	if task := dailyTask(t, a); len(accounts.reads) != 15 || task.LastCode != "sync_completed" {
		t.Fatalf("the next trigger read %d pages, task %+v", len(accounts.reads), task)
	}
}

// Removing the archive a daily sync reads pauses the sync; a round then
// reading it stops at its next page and does not bring the archive back.
func TestDailySyncStopsWhenItsArchiveIsRemoved(t *testing.T) {
	a, host, accounts := syncHost(t)
	host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "hour": 0, "days": 30, "confirm": true})
	ref := host.job("game.sync.")
	accounts.onRead = func(n int) {
		if n == 3 {
			if _, err := a.manageGacha("gacha.remove", map[string]any{"uid": "600000001", "region": "prod_official_usa"}); err != nil {
				t.Error(err)
			}
		}
	}
	host.trigger(ref)
	if task := dailyTask(t, a); len(accounts.reads) != 3 || task.State != "paused" || task.LastCode != "archive_removed" {
		t.Fatalf("read %d pages, task %+v", len(accounts.reads), task)
	}
	if _, err := a.Gacha.Read("600000001", "prod_official_usa"); err == nil {
		t.Fatal("the removed archive came back")
	}
	if end, actions := host.trigger(ref); end["type"] != "result" || len(actions) != 0 {
		t.Fatalf("the paused sync's trigger ended with %v after %v", end, actions)
	}
}

// dailyTask is the only daily sync.
func dailyTask(t *testing.T, a *App) SyncTask {
	t.Helper()
	items, err := a.SyncTasks.List()
	if err != nil || len(items) != 1 {
		t.Fatalf("daily syncs %v, %v", items, err)
	}
	return items[0]
}

// detachedBeforeReading reports whether an event moved to the background
// before it first asked the account plugin for records.
func detachedBeforeReading(actions []hostAction) bool {
	for _, action := range actions {
		switch {
		case action.Name == "event.detach":
			return true
		case action.Name == "plugin.call" && action.Data["method"] == "execute":
			return false
		}
	}
	return false
}

// The owner may act as soon as a completed sync's notification arrives: a
// background sync of the same role started right then is accepted, as the
// finished one no longer counts as running.
func TestBackgroundSyncStartedRightAfterItsNotificationIsAccepted(t *testing.T) {
	_, host, accounts := syncHost(t)
	var followUp map[string]any
	host.onSend = func(rayleabot.MessageSendRequest) {
		if followUp == nil {
			followUp = map[string]any{}
			followUp, _ = host.await(host.postManage("gacha.task.start", map[string]any{"account_ref": "account", "role_ref": "role", "confirm": true}))
		}
	}
	if end, _ := startSync(host, map[string]any{"notify": true}); end["type"] != "result" {
		t.Fatalf("the first sync ended with %v", end)
	}
	// The second, incremental sync reads a page of each pool.
	if followUp["type"] != "result" || len(host.detached) != 2 || len(accounts.reads) != 21 {
		t.Fatalf("the sync started after the notification ended with %v after %d pages", followUp, len(accounts.reads)-15)
	}
}

// A trigger the host hands over as soon as a daily sync's notification
// arrives finds the task free rather than still being read.
func TestDailySyncTriggerRightAfterItsNotificationFindsTheTaskFree(t *testing.T) {
	_, host, _ := syncHost(t)
	host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "hour": 0, "days": 30, "notify": true, "confirm": true})
	ref := host.job("game.sync.")
	var followUp map[string]any
	host.onSend = func(rayleabot.MessageSendRequest) {
		if followUp == nil {
			followUp = map[string]any{}
			followUp, _ = host.await(host.postTrigger(ref))
		}
	}
	host.trigger(ref)
	if followUp["type"] != "result" || asObject(followUp["data"])["checked"] != true {
		t.Fatalf("the trigger right after the notification ended with %v", followUp)
	}
}
