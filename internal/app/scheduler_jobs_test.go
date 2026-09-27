package app

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// taskAccounts answers the account plugin's service for one account of user
// "u" holding one overseas role: the role checks, delegation.create, which
// grants "d1", and the requests of the jobs, which it records by operation
// and refuses. Only a trigger may make a job's request, and with the
// delegation.
type taskAccounts struct {
	t        *testing.T
	executed []string
}

func (s *taskAccounts) answer(request rayleabot.ServiceCallRequest, scheduled bool) (map[string]any, string) {
	if request.TargetPluginID != "raylea.mihoyo-accounts" || request.Service != "accounts" || request.ServiceVersion != 1 {
		s.t.Errorf("call %+v", request)
		return nil, "plugin.service_not_found"
	}
	role := map[string]any{"ref": "role", "game": "starrail", "uid": "600000001", "region": "prod_official_usa", "nickname": "开拓者", "level": 70}
	switch request.Method {
	case "roles":
		return map[string]any{"roles": []any{role}}, ""
	case "list":
		owner := map[string]any{"source_protocol": "onebot11", "source_adapter": "a", "bot_id": "bot", "actor_id": "u"}
		return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "account_label": "账号", "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	case "delegation.create":
		return map[string]any{"delegation": map[string]any{"ref": "d1", "expires_at_ms": time.Now().Add(30 * 24 * time.Hour).UnixMilli()}}, ""
	case "execute":
		operation := asText(request.Params["operation"])
		if !scheduled || request.Params["delegation_ref"] != "d1" {
			s.t.Errorf("%s was requested outside its job", operation)
		}
		s.executed = append(s.executed, operation)
		return nil, "plugin.upstream_unavailable"
	}
	s.t.Errorf("unexpected %s", request.Method)
	return nil, "plugin.method_not_found"
}

// Each kind of job, created as the management page or a group administrator
// creates it, runs its own task on the host's triggers, which carry the job's
// payload but not its task ID.
func TestEachJobsTriggerRunsItsTask(t *testing.T) {
	a := pluginApp(t)
	news := 0
	a.Content = PublicContentClient{HTTP: doerFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/getNewsList") {
			t.Errorf("content request %s", r.URL)
		}
		news++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":{"list":[]}}`))}, nil
	})}
	accounts := &taskAccounts{t: t}
	host := newSDKHost(t, a, accounts.answer)
	role := func(input map[string]any) map[string]any {
		input["account_ref"], input["role_ref"], input["days"], input["confirm"] = "account", "role", 30, true
		return input
	}
	for _, job := range []struct {
		action    string
		input     map[string]any
		prefix    string
		operation string
	}{
		{"reminder.create", role(map[string]any{"threshold": 80}), "game.reminder.", "starrail.note"},
		{"challenge.reminder.create", role(map[string]any{"kind": "challenge", "metric": "floor", "threshold": 12, "hour": 20}), "game.challenge.", "starrail.challenge"},
		{"signin.task.create", role(map[string]any{"hour": 0}), "game.sign.", "starrail.sign"},
		{"monthly.task.create", role(map[string]any{"hour": 0}), "game.monthly.", "starrail.monthly"},
		{"community.task.create", role(map[string]any{"once": true, "read": true}), "game.community.", "starrail.community_run"},
		{"gacha.task.create", role(map[string]any{"kind": "once"}), "game.sync.", "starrail.gacha"},
	} {
		if end, _ := host.manage(job.action, job.input); end["type"] != "result" {
			t.Fatalf("%s ended with %v", job.action, end)
		}
		ref := host.job(job.prefix)
		if host.jobs[ref]["task_id"] != ref {
			t.Fatalf("%s created a job with payload %v", job.action, host.jobs[ref])
		}
		// Challenge reminders and monthly collection wait for their time of
		// day; the trigger comes when it is due.
		_ = a.Reminders.edit(ref, func(items *[]Reminder, i int) error {
			if i >= 0 {
				(*items)[i].NextCheckMS = 0
			}
			return nil
		})
		executed := len(accounts.executed)
		if end, _ := host.trigger(ref); end["type"] != "result" {
			t.Fatalf("the trigger of %s ended with %v", ref, end)
		}
		if len(accounts.executed) != executed+1 || accounts.executed[executed] != job.operation {
			t.Fatalf("the trigger of %s requested %v", ref, accounts.executed[executed:])
		}
	}
	if end, _ := host.groupMessage("*开启公告推送", "开启公告推送"); !strings.Contains(terminalText(end), "公告推送已开启") {
		t.Fatalf("开启公告推送 ended with %v", end)
	}
	ref := host.job("game.content.")
	if end, _ := host.trigger(ref); end["type"] != "result" || news != 1 {
		t.Fatalf("the trigger of %s ended with %v after %d reads", ref, end, news)
	}
	if items, _ := a.Subscriptions.List(); len(items) != 1 || items[0].LastCode != "checked" {
		t.Fatalf("subscriptions %+v", items)
	}
	// A push the management page removed has its job removed by the next
	// trigger.
	if end, _ := host.manage("content.subscription.remove", map[string]any{"ref": ref, "confirm": true}); end["type"] != "result" {
		t.Fatalf("the removal ended with %v", end)
	}
	if host.trigger(ref); len(host.deleted) != 1 || host.deleted[0] != ref || news != 1 {
		t.Fatalf("deleted %v after %d reads", host.deleted, news)
	}
}
