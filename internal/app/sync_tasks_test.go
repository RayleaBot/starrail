package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

func syncTaskFixture(t *testing.T, kind string) (*SyncTaskStore, *gacha.Syncs, *gacha.Store, SyncTask, int64) {
	t.Helper()
	dir := t.TempDir()
	now := time.Date(2026, 9, 20, 8, 0, 0, 0, time.FixedZone("CN", 28800)).UnixMilli()
	s := syncTaskStore(dir)
	jobs := &gacha.Syncs{}
	archive := &gacha.Store{Directory: filepath.Join(dir, "gacha"), Game: "starrail"}
	task := SyncTask{Ref: "game.sync.fixture", Selection: Selection{"account", "role"}, Provider: "provider", Role: Role{Ref: "role", Game: "starrail", UID: "100000001", Region: "prod_gf_cn"}, Kind: kind, Hour: 8, State: "waiting", ExpiresAtMS: now + 30*86400000, DelegationRef: "grant", Owner: Subject{ActorID: "owner"}}
	if err := seedSyncTasks(s, task); err != nil {
		t.Fatal(err)
	}
	return s, jobs, archive, task, now
}
func syncPage(_ context.Context, _ SyncTask, pool, end string, page int) (gacha.RemotePage, error) {
	p := gacha.RemotePage{Records: []gacha.Record{}, Timezone: 8, Language: "zh-cn", NextID: end}
	if pool == "2" {
		p.Records = []gacha.Record{{ID: "100", ItemID: "1001", GachaType: "2", UIGFType: "2", GachaID: "4001", Time: "2026-09-01 08:00:00", Rank: "5"}}
		p.NextID = "100"
	}
	return p, nil
}

// fixedPace is a trigger at now whose clock stands still.
func fixedPace(now int64) SyncPace {
	at := time.UnixMilli(now)
	return SyncPace{Now: func() time.Time { return at }, Sleep: func(ctx context.Context, _ time.Duration) error { return ctx.Err() }, Stop: at.Add(chatTaskBudget)}
}
func tickOne(t *testing.T, s *SyncTaskStore, j *gacha.Syncs, a *gacha.Store, task SyncTask, now int64, fetch SyncTaskFetch, send SyncTaskSend) SyncTask {
	t.Helper()
	if err := s.Tick(t.Context(), task.Ref, fixedPace(now), j, a, fetch, send); err != nil {
		t.Fatal(err)
	}
	items, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	return items[0]
}
func TestBackgroundSyncContinuesWithoutUIAndRestartsSafely(t *testing.T) {
	s, j, a, task, now := syncTaskFixture(t, "once")
	task = tickOne(t, s, j, a, task, now, syncPage, nil)
	if task.Progress.Pages != syncTaskPages || task.State != "running" {
		t.Fatal(task)
	}
	if _, err := a.Read(task.Role.UID, task.Role.Region); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("partial page written")
	}
	// Fresh process has only persisted task metadata. It reads again from the
	// current archive; no in-memory version or authorization is reconstructed.
	s = &SyncTaskStore{taskFiles: taskFiles[SyncTask]{Directory: s.Directory}}
	j = &gacha.Syncs{}
	a = &gacha.Store{Directory: a.Directory, Game: a.Game}
	for i := 1; i <= 6; i++ {
		task = tickOne(t, s, j, a, task, now+int64(i)*60000, syncPage, nil)
	}
	if task.State != "completed" || task.Progress.Result.Added != 1 || task.Restarts != 1 {
		t.Fatal(task)
	}
	actual, err := a.Read(task.Role.UID, task.Role.Region)
	if err != nil || len(actual.Records) != 1 {
		t.Fatal(actual, err)
	}
	tickOne(t, s, j, a, task, now+86400000, func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
		t.Fatal("completed single task repeated")
		return gacha.RemotePage{}, nil
	}, nil)
}
func TestBackgroundSyncDeletionAndConcurrentAdmission(t *testing.T) {
	s, j, a, task, now := syncTaskFixture(t, "once")
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Tick(t.Context(), task.Ref, fixedPace(now), j, a, syncPage, nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	items, _ := s.List()
	if items[0].Progress.Pages != syncTaskPages {
		t.Fatal("duplicate tick advanced pages", items)
	}
	if err := s.RemoveArchive(j, a, task.Role.UID, task.Role.Region); err != nil {
		t.Fatal(err)
	}
	s = &SyncTaskStore{taskFiles: taskFiles[SyncTask]{Directory: s.Directory}}
	j = &gacha.Syncs{}
	task = tickOne(t, s, j, a, task, now+60000, func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
		t.Fatal("deleted archive fetched again")
		return gacha.RemotePage{}, nil
	}, nil)
	if task.State != "paused" || task.LastCode != "archive_removed" {
		t.Fatal(task)
	}
	if _, err := a.Read(task.Role.UID, task.Role.Region); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("deleted archive revived")
	}
}
func TestBackgroundSyncDailyHourCompletionAndNotifyOnce(t *testing.T) {
	s, j, a, task, now := syncTaskFixture(t, "daily")
	task.Notify = true
	_ = s.edit(task.Ref, func(items *[]SyncTask, i int) error { (*items)[i] = task; return nil })
	sent := 0
	send := func(context.Context, SyncTask, string) error { sent++; return nil }
	task = tickOne(t, s, j, a, task, now-60000, func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
		t.Fatal("ran before hour")
		return gacha.RemotePage{}, nil
	}, send)
	for i := 0; i < 6; i++ {
		task = tickOne(t, s, j, a, task, now+int64(i)*60000, syncPage, send)
	}
	if task.State != "waiting" || sent != 1 || task.LastNotificationMS == 0 {
		t.Fatal(task, sent)
	}
	s = &SyncTaskStore{taskFiles: taskFiles[SyncTask]{Directory: s.Directory}}
	task = tickOne(t, s, j, a, task, now+12*3600000, func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
		t.Fatal("same day restarted")
		return gacha.RemotePage{}, nil
	}, send)
	task = tickOne(t, s, j, a, task, now+86400000, syncPage, send)
	if task.State != "running" || task.Progress.Pages != syncTaskPages || sent != 1 {
		t.Fatal(task, sent)
	}
}
func TestBackgroundSyncFailuresExpiryAndMergeConflict(t *testing.T) {
	for _, mode := range []string{"auth", "transient", "daily", "conflict", "expiry"} {
		t.Run(mode, func(t *testing.T) {
			kind := "once"
			if mode == "daily" {
				kind = "daily"
			}
			s, j, a, task, now := syncTaskFixture(t, kind)
			if mode == "expiry" {
				task.ExpiresAtMS = now
				_ = s.edit(task.Ref, func(items *[]SyncTask, i int) error { (*items)[i] = task; return nil })
				task = tickOne(t, s, j, a, task, now, syncPage, nil)
				if task.State != "expired" {
					t.Fatal(task)
				}
				return
			}
			if mode == "conflict" {
				task = tickOne(t, s, j, a, task, now, syncPage, nil)
				_, _, err := a.Import(gacha.Archive{UID: task.Role.UID, Region: task.Role.Region, Timezone: 8, Language: "zh-cn", Records: []gacha.Record{{ID: "99", ItemID: "1002", GachaType: "2", UIGFType: "2", GachaID: "4001", Time: "2026-09-01 07:00:00", Rank: "3"}}})
				if err != nil {
					t.Fatal(err)
				}
				for i := 1; i <= 5; i++ {
					task = tickOne(t, s, j, a, task, now+int64(i)*60000, syncPage, nil)
				}
				if task.State != "paused" || task.LastCode != "plugin.game_sync_conflict" {
					t.Fatal(task)
				}
				saved, _ := a.Read(task.Role.UID, task.Role.Region)
				if len(saved.Records) != 1 || saved.Records[0].ID != "99" {
					t.Fatal("conflict modified archive")
				}
				return
			}
			code := "plugin.upstream_unavailable"
			if mode == "auth" {
				code = "plugin.account_delegation_denied"
			}
			fail := func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
				return gacha.RemotePage{}, &rayleabot.ActionError{Code: code}
			}
			for i := 0; i < 3; i++ {
				task = tickOne(t, s, j, a, task, now+int64(i)*5*60000, fail, nil)
			}
			want := "paused"
			if mode == "daily" {
				want = "waiting"
			}
			if task.State != want {
				t.Fatal(task)
			}
		})
	}
}

// syncRead is a gacha record request: when it came and what it asked for.
type syncRead struct {
	at        time.Time
	pool, end string
	page      int
}

// A background sync run through the SDK as the host runs it: each trigger
// asks for at most three pages, a second or more apart, and starts none
// after the event budget; the page still open when the event's time is up
// is asked for again by the next trigger without counting as a failure,
// and the triggers finish the sync.
func TestBackgroundSyncTriggersStayWithinTheirEvents(t *testing.T) {
	j := newJobsHost(t)
	reads := []syncRead{}
	// Pages take five seconds; slow gives some their own time, and the
	// page that answers after the host timed its event out.
	slow := map[int]time.Duration{}
	timeout := -1
	j.accounts.gacha = func(input map[string]any) (map[string]any, string) {
		pool, end := asText(input["gacha_type"]), asText(input["end_id"])
		page, _ := strconv.Atoi(asText(input["page"]))
		reads = append(reads, syncRead{j.clock.Now(), pool, end, page})
		n := len(reads) - 1
		if n == timeout {
			j.clock.set(j.clock.Now().Add(time.Minute))
			return nil, "plugin.event_timeout"
		}
		took := 5 * time.Second
		if d, ok := slow[n]; ok {
			took = d
		}
		j.clock.set(j.clock.Now().Add(took))
		records := []any{}
		if pool == "11" {
			first := 1
			if id, _ := strconv.Atoi(end); id != 0 {
				first = 5000 - id + 1
			}
			for k := first; k <= 45 && len(records) < 20; k++ {
				records = append(records, map[string]any{"uid": "600000001", "gacha_id": "2003", "gacha_type": "11", "item_id": "1308", "count": "1", "time": fmt.Sprintf("2024-01-01 00:%02d:00", 59-k), "name": "黄泉", "item_type": "角色", "rank_type": "5", "id": strconv.Itoa(5000 - k)})
			}
		}
		return map[string]any{"list": records, "region": "prod_official_usa"}, ""
	}
	if end, _ := j.host.manage("gacha.task.create", map[string]any{"account_ref": "account", "role_ref": "role", "kind": "once", "days": 30, "confirm": true}); end["type"] != "result" {
		t.Fatalf("gacha.task.create ended with %v", end)
	}
	ref := j.host.job("game.sync.")
	start := j.clock.Now()
	trigger := func(minute int) SyncTask {
		t.Helper()
		j.clock.set(start.Add(time.Duration(minute) * time.Minute))
		if end, _ := j.host.trigger(ref); end["type"] != "result" {
			t.Fatalf("trigger %d ended with %v", minute, end)
		}
		items, _ := j.a.SyncTasks.List()
		return items[0]
	}
	// Three pages, the second and third each a second after the previous
	// answer.
	if task := trigger(1); len(reads) != 3 || task.Progress.Pages != 3 || task.State != "running" {
		t.Fatalf("the first trigger read %v, task %+v", reads, task)
	}
	for i := 1; i < 3; i++ {
		if gap := reads[i].at.Sub(reads[i-1].at); gap != 6*time.Second {
			t.Fatalf("read %d came %v after the previous", i, gap)
		}
	}
	// Slow pages: the second answers 51 seconds in, past the budget, so no
	// third is asked for.
	slow[3], slow[4] = 25*time.Second, 25*time.Second
	if task := trigger(2); len(reads) != 5 || task.Progress.Pages != 5 {
		t.Fatalf("the slow trigger read %v, task %+v", reads[3:], task)
	}
	// The page still open when the event's time is up is asked for again by
	// the next trigger.
	timeout = 5
	task := trigger(3)
	if len(reads) != 6 || task.Progress.Pages != 5 || task.Failures != 0 || task.State != "running" || task.NextCheckMS > start.Add(4*time.Minute).UnixMilli() {
		t.Fatalf("the trigger that ran out read %v, task %+v", reads[5:], task)
	}
	task = trigger(4)
	if reads[6].pool != reads[5].pool || reads[6].end != reads[5].end || reads[6].page != reads[5].page {
		t.Fatalf("the next trigger read %+v instead of %+v", reads[6], reads[5])
	}
	for minute := 5; task.State == "running"; minute++ {
		if minute > 8 {
			t.Fatalf("the sync did not finish: %+v", task)
		}
		task = trigger(minute)
	}
	if task.State != "completed" || task.Progress.Result.Added != 45 {
		t.Fatalf("task %+v", task)
	}
	if archive, err := j.a.Gacha.Read("600000001", "prod_official_usa"); err != nil || len(archive.Records) != 45 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
}

func seedSyncTasks(s *SyncTaskStore, tasks ...SyncTask) error {
	return s.edit("", func(items *[]SyncTask, _ int) error {
		*items = append(*items, tasks...)
		return nil
	})
}
