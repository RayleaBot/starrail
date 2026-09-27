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
// warps: ten pages of that pool and one empty page of each other. onRead
// sees each record request after it is counted.
type syncAccounts struct {
	t      *testing.T
	clock  *fakeClock
	reads  []syncRead
	onRead func(n int)
}

func (s *syncAccounts) answer(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
	role := map[string]any{"ref": "role", "game": "starrail", "uid": "600000001", "region": "prod_official_usa", "nickname": "开拓者", "level": 70}
	switch request.Method {
	case "roles":
		return map[string]any{"roles": []any{role}}, ""
	case "list":
		owner := map[string]any{"source_protocol": "onebot11", "source_adapter": "a", "bot_id": "bot", "actor_id": "u"}
		return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "account_label": "账号", "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "execute":
		// The management action's own origin reads the records.
		if scheduled || request.Params["delegation_ref"] != nil || request.Params["operation"] != "starrail.gacha" {
			s.t.Errorf("records requested with %v", request.Params)
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
	items := asList(asObject(listed["data"])["items"])
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
			if _, err := a.backgroundSyncAction(t.Context(), nil, "gacha.task.cancel", map[string]any{"ref": items[0].Ref}); err != nil {
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
	if end["type"] != "error" || end["code"] != "plugin.game_background_full" || end["message"] != "后台处理的任务已达上限，请稍后再试。" {
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
