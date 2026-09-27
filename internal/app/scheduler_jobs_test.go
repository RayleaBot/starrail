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
// and refuses unless data answers the operation with its data. Only a
// trigger may make a job's request, and with the delegation.
type taskAccounts struct {
	t        *testing.T
	executed []string
	data     map[string]func(input map[string]any) (map[string]any, string)
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
	case "delegation.revoke":
		return map[string]any{}, ""
	case "execute":
		operation := asText(request.Params["operation"])
		if !scheduled || request.Params["delegation_ref"] != "d1" {
			s.t.Errorf("%s was requested outside its job", operation)
		}
		s.executed = append(s.executed, operation)
		if answer := s.data[operation]; answer != nil {
			data, failure := answer(asObject(request.Params["input"]))
			if failure != "" {
				return nil, failure
			}
			return map[string]any{"operation": operation, "role": role, "data": data}, ""
		}
		return nil, "plugin.upstream_unavailable"
	}
	s.t.Errorf("unexpected %s", request.Method)
	return nil, "plugin.method_not_found"
}

// jobsHost is the plugin on the SDK host with taskAccounts' account and 米游社
// news without posts, whose reads it counts after onNews sees them. The
// plugin's clock starts at the current time.
type jobsHost struct {
	a        *App
	clock    *fakeClock
	host     *sdkHost
	accounts *taskAccounts
	news     int
	onNews   func(*http.Request)
}

func newJobsHost(t *testing.T) *jobsHost {
	t.Helper()
	j := &jobsHost{a: pluginApp(t), clock: &fakeClock{at: time.Now()}, accounts: &taskAccounts{t: t}}
	j.a.clock = j.clock
	j.a.Content = PublicContentClient{HTTP: doerFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/getNewsList") {
			t.Errorf("content request %s", r.URL)
		}
		if j.onNews != nil {
			j.onNews(r)
		}
		j.news++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"retcode":0,"data":{"list":[]}}`))}, nil
	})}
	j.host = newSDKHost(t, j.a, j.accounts.answer)
	return j
}

// createdJob is a job created as a user creates it, with the account request
// its task makes; a push's reads the news instead.
type createdJob struct{ ref, operation string }

// createJobs creates each kind of job as the management page or a group
// administrator does.
func (j *jobsHost) createJobs(t *testing.T) []createdJob {
	t.Helper()
	role := func(input map[string]any) map[string]any {
		input["account_ref"], input["role_ref"], input["days"], input["confirm"] = "account", "role", 30, true
		return input
	}
	jobs := []createdJob{}
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
	} {
		if end, _ := j.host.manage(job.action, job.input); end["type"] != "result" {
			t.Fatalf("%s ended with %v", job.action, end)
		}
		jobs = append(jobs, createdJob{j.host.job(job.prefix), job.operation})
	}
	if end, _ := j.host.groupMessage("*开启公告推送", "开启公告推送"); !strings.Contains(terminalText(end), "公告推送已开启") {
		t.Fatalf("开启公告推送 ended with %v", end)
	}
	jobs = append(jobs, createdJob{j.host.job("game.content."), ""})
	for _, job := range jobs {
		if j.host.jobs[job.ref]["task_id"] != job.ref {
			t.Fatalf("%s has payload %v", job.ref, j.host.jobs[job.ref])
		}
	}
	return jobs
}

// runJob triggers a job once its task is due and checks that the task ran.
func (j *jobsHost) runJob(t *testing.T, job createdJob) {
	t.Helper()
	// Challenge reminders and monthly collection wait for their time of day.
	_ = j.a.Reminders.edit(job.ref, func(items *[]Reminder, i int) error {
		if i >= 0 {
			(*items)[i].NextCheckMS = 0
		}
		return nil
	})
	executed, news := len(j.accounts.executed), j.news
	if end, _ := j.host.trigger(job.ref); end["type"] != "result" {
		t.Fatalf("the trigger of %s ended with %v", job.ref, end)
	}
	if job.operation == "" {
		if j.news != news+1 {
			t.Fatalf("the trigger of %s read the news %d times", job.ref, j.news-news)
		}
		return
	}
	if len(j.accounts.executed) != executed+1 || j.accounts.executed[executed] != job.operation {
		t.Fatalf("the trigger of %s requested %v", job.ref, j.accounts.executed[executed:])
	}
}

// Each kind of job, created as the management page or a group administrator
// creates it, runs its own task on the host's triggers, which carry the job's
// payload but not its task ID.
func TestEachJobsTriggerRunsItsTask(t *testing.T) {
	j := newJobsHost(t)
	jobs := j.createJobs(t)
	for _, job := range jobs {
		j.runJob(t, job)
	}
	if items, _ := j.a.Subscriptions.List(); len(items) != 1 || items[0].LastCode != "checked" {
		t.Fatalf("subscriptions %+v", items)
	}
}

// Removing a task, as the management page or its user in chat does, deletes
// its job in that event. A trigger whose task is gone, as one removed before
// removals deleted jobs, deletes its own job without running anything.
func TestRemovedTasksDeleteTheirJobs(t *testing.T) {
	j := newJobsHost(t)
	jobs := j.createJobs(t)
	removals := []func(ref string) map[string]any{
		func(ref string) map[string]any {
			end, _ := j.host.manage("reminder.remove", map[string]any{"ref": ref})
			return end
		},
		func(ref string) map[string]any {
			end, _ := j.host.manage("challenge.reminder.remove", map[string]any{"ref": ref})
			return end
		},
		func(string) map[string]any {
			end, _ := j.host.message("*自动签到 关闭", "自动签到", "关闭")
			return end
		},
		func(ref string) map[string]any {
			end, _ := j.host.manage("monthly.task.remove", map[string]any{"ref": ref})
			return end
		},
		func(ref string) map[string]any {
			end, _ := j.host.manage("community.task.remove", map[string]any{"ref": ref})
			return end
		},
		func(ref string) map[string]any {
			end, _ := j.host.manage("content.subscription.remove", map[string]any{"ref": ref, "confirm": true})
			return end
		},
	}
	payloads := map[string]map[string]any{}
	for i, job := range jobs {
		payloads[job.ref] = j.host.jobs[job.ref]
		if end := removals[i](job.ref); end["type"] == "error" {
			t.Fatalf("removing %s ended with %v", job.ref, end)
		}
		if _, kept := j.host.jobs[job.ref]; kept || len(j.host.deleted) != i+1 || j.host.deleted[i] != job.ref {
			t.Fatalf("removing %s deleted %v", job.ref, j.host.deleted)
		}
	}
	for _, job := range jobs {
		j.host.jobs[job.ref] = payloads[job.ref]
		deleted := len(j.host.deleted)
		if end, _ := j.host.trigger(job.ref); end["type"] != "result" {
			t.Fatalf("the trigger of removed %s ended with %v", job.ref, end)
		}
		if _, kept := j.host.jobs[job.ref]; kept || len(j.host.deleted) != deleted+1 || j.host.deleted[deleted] != job.ref {
			t.Fatalf("the trigger of removed %s deleted %v", job.ref, j.host.deleted[deleted:])
		}
	}
	if len(j.accounts.executed) != 0 || j.news != 0 || len(j.host.sent) != 0 {
		t.Fatalf("removed tasks ran: %v, %d reads, sent %v", j.accounts.executed, j.news, j.host.sent)
	}
}

// Jobs an earlier version created have no task ID in their payloads. The
// first of their triggers creates every stored task's job again with the
// same schedule, after which each job runs its task; a job without a stored
// task, as a gacha link's, does nothing and is not created again.
func TestJobsWithoutTaskIDsAreCreatedAgain(t *testing.T) {
	j := newJobsHost(t)
	jobs := j.createJobs(t)
	first := map[string]map[string]any{}
	for _, created := range j.host.created {
		first[asText(created["task_id"])] = created
	}
	for _, job := range jobs {
		j.host.jobs[job.ref] = map[string]any{"kind": j.host.jobs[job.ref]["kind"]}
	}
	j.host.jobs["game.link.OLD"] = map[string]any{"kind": "gacha_link"}
	before := len(j.host.created)
	for range 2 {
		if end, _ := j.host.trigger("game.link.OLD"); end["type"] != "result" {
			t.Fatalf("the old job's trigger ended with %v", end)
		}
	}
	again := j.host.created[before:]
	if len(again) != len(jobs) {
		t.Fatalf("created %d jobs again for %d tasks", len(again), len(jobs))
	}
	for _, created := range again {
		ref := asText(created["task_id"])
		old := first[ref]
		if old == nil || created["cron"] != old["cron"] || created["log_label"] != old["log_label"] || asObject(created["payload"])["task_id"] != ref || asObject(created["payload"])["kind"] != asObject(old["payload"])["kind"] {
			t.Fatalf("%s was created again as %v, first as %v", ref, created, old)
		}
	}
	if _, kept := j.host.jobs["game.link.OLD"]; !kept || len(j.host.deleted) != 0 || len(j.host.sent) != 0 || len(j.accounts.executed) != 0 || j.news != 0 {
		t.Fatalf("the old job did work: deleted %v, sent %v", j.host.deleted, j.host.sent)
	}
	for _, job := range jobs {
		j.runJob(t, job)
	}
}

// Every trigger's work ends within the host's event time: the news request
// of a push still open when the event's time is up is canceled, and a
// stamina reminder whose check answers only then sends nothing.
func TestTriggersEndWithinTheEventTime(t *testing.T) {
	j := newJobsHost(t)
	jobs := j.createJobs(t)
	late := func() { j.clock.set(j.clock.Now().Add(time.Minute)) }
	canceled := false
	j.onNews = func(r *http.Request) {
		late()
		canceled = r.Context().Err() != nil
	}
	j.accounts.data = map[string]func(map[string]any) (map[string]any, string){"starrail.note": func(map[string]any) (map[string]any, string) {
		late()
		return map[string]any{"current_stamina": 240, "max_stamina": 240}, ""
	}}
	for _, ref := range []string{jobs[len(jobs)-1].ref, jobs[0].ref} {
		if end, _ := j.host.trigger(ref); end["type"] == "action" {
			t.Fatalf("the trigger of %s ended with %v", ref, end)
		}
	}
	if !canceled || j.news != 1 || len(j.accounts.executed) != 1 || j.accounts.executed[0] != "starrail.note" || len(j.host.sent) != 0 {
		t.Fatalf("canceled %v after %d reads; requested %v, sent %v", canceled, j.news, j.accounts.executed, j.host.sent)
	}
}
