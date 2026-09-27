package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

// dailyFixture is a stored daily sync due at eight, Beijing time, on
// 2030-09-20, with its store, sync engine and archive store. The day is in
// the future so the times a round records from the wall clock come before
// the triggers'.
func dailyFixture(t *testing.T) (*SyncTaskStore, *gacha.Syncs, *gacha.Store, SyncTask, time.Time) {
	t.Helper()
	dir := t.TempDir()
	eight := time.Date(2030, 9, 20, 8, 0, 0, 0, time.FixedZone("CN", 28800))
	s := syncTaskStore(dir)
	archive := &gacha.Store{Directory: filepath.Join(dir, "gacha"), Game: "starrail"}
	task := SyncTask{Ref: "game.sync.fixture", Selection: Selection{"account", "role"}, Provider: "provider", Role: Role{Ref: "role", Game: "starrail", UID: "600000001", Region: "prod_official_usa"}, Hour: 8, State: "waiting", ExpiresAtMS: eight.AddDate(0, 0, 30).UnixMilli(), DelegationRef: "d1", Owner: Subject{ActorID: "owner"}}
	if err := s.edit("", func(items *[]SyncTask, _ int) error { *items = append(*items, task); return nil }); err != nil {
		t.Fatal(err)
	}
	return s, &gacha.Syncs{}, archive, task, eight
}

// dailyPage is a pool's only page: one record of the standard warp.
func dailyPage(_ context.Context, _ SyncTask, pool, end string, _ int) (gacha.RemotePage, error) {
	page := gacha.RemotePage{Records: []gacha.Record{}, Timezone: -5, Language: "zh-cn", NextID: end}
	if pool == "1" {
		page.Records = []gacha.Record{{ID: "100", ItemID: "1001", GachaType: "1", UIGFType: "1", GachaID: "4001", Time: "2026-09-01 08:00:00", Rank: "5"}}
	}
	return page, nil
}

// triggerAt runs one trigger of the task at now, its round too when due, and
// returns the stored task.
func (s *SyncTaskStore) triggerAt(t *testing.T, jobs *gacha.Syncs, archive *gacha.Store, now time.Time, fetch SyncTaskFetch) (SyncTask, bool) {
	t.Helper()
	task, claimed, err := s.claim("game.sync.fixture")
	if err != nil || !claimed {
		t.Fatal(claimed, err)
	}
	defer s.release(task.Ref)
	due, err := s.due(&task, now)
	if err == nil && due {
		_, err = s.run(t.Context(), &task, jobs, archive, func(ctx context.Context, _ time.Duration) error { return ctx.Err() }, fetch)
	}
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.List()
	if err != nil || len(items) != 1 {
		t.Fatal(items, err)
	}
	return items[0], due
}

// A daily sync waits for its hour, reads its round once that day, and runs
// again the next day.
func TestDailySyncRunsOnceADayAfterItsHour(t *testing.T) {
	s, jobs, archive, _, eight := dailyFixture(t)
	if task, due := s.triggerAt(t, jobs, archive, eight.Add(-time.Minute), dailyPage); due || task.NextCheckMS != eight.UnixMilli() {
		t.Fatalf("before the hour: due %v, task %+v", due, task)
	}
	task, due := s.triggerAt(t, jobs, archive, eight, dailyPage)
	if !due || task.State != "waiting" || task.RunDay != "2030-09-20" || task.Progress.Result == nil || task.Progress.Result.Added != 1 {
		t.Fatalf("at the hour: due %v, task %+v", due, task)
	}
	// The round's next check is the next day after the time it finished;
	// the same day does not run again even once that check is reached.
	_ = s.edit(task.Ref, func(items *[]SyncTask, i int) error { (*items)[i].NextCheckMS = 0; return nil })
	if task, due := s.triggerAt(t, jobs, archive, eight.Add(12*time.Hour), dailyPage); due || task.NextCheckMS != eight.AddDate(0, 0, 1).UnixMilli() {
		t.Fatalf("later that day: due %v, task %+v", due, task)
	}
	if task, due := s.triggerAt(t, jobs, archive, eight.AddDate(0, 0, 1), dailyPage); !due || task.RunDay != "2030-09-21" || task.Progress.Result.Added != 0 {
		t.Fatalf("the next day: due %v, task %+v", due, task)
	}
}

// A transient failure is tried again after five minutes, and three in a row
// leave the round for the next day; a revoked delegation or a record
// conflict pauses the task; an expired task stops.
func TestDailySyncFailuresConflictsAndExpiry(t *testing.T) {
	failing := func(code string) SyncTaskFetch {
		return func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error) {
			return gacha.RemotePage{}, &rayleabot.ActionError{Code: code}
		}
	}
	t.Run("transient", func(t *testing.T) {
		s, jobs, archive, _, eight := dailyFixture(t)
		for i := range 3 {
			task, due := s.triggerAt(t, jobs, archive, eight.Add(time.Duration(i)*time.Hour), failing("plugin.upstream_unavailable"))
			if !due || task.Failures != i+1 {
				t.Fatalf("failure %d: due %v, task %+v", i+1, due, task)
			}
			if i < 2 && (task.State != "running" || task.NextCheckMS < time.Now().Add(4*time.Minute).UnixMilli()) {
				t.Fatalf("failure %d: task %+v", i+1, task)
			}
			if i == 2 && task.State != "waiting" {
				t.Fatalf("third failure: task %+v", task)
			}
		}
	})
	t.Run("delegation", func(t *testing.T) {
		s, jobs, archive, _, eight := dailyFixture(t)
		if task, _ := s.triggerAt(t, jobs, archive, eight, failing("plugin.account_delegation_denied")); task.State != "paused" || task.LastCode != "plugin.account_delegation_denied" {
			t.Fatalf("task %+v", task)
		}
	})
	t.Run("conflict", func(t *testing.T) {
		s, jobs, archive, task, eight := dailyFixture(t)
		kept := gacha.Archive{UID: task.Role.UID, Region: task.Role.Region, Timezone: -5, Language: "zh-cn", Records: []gacha.Record{{ID: "100", ItemID: "1002", GachaType: "1", UIGFType: "1", GachaID: "4001", Time: "2026-09-01 08:00:00", Rank: "4"}}}
		if _, _, err := archive.Import(kept); err != nil {
			t.Fatal(err)
		}
		if task, _ := s.triggerAt(t, jobs, archive, eight, dailyPage); task.State != "paused" || task.LastCode != "plugin.game_sync_conflict" {
			t.Fatalf("task %+v", task)
		}
		if saved, _ := archive.Read(task.Role.UID, task.Role.Region); len(saved.Records) != 1 || saved.Records[0].ItemID != "1002" {
			t.Fatalf("the conflict changed the archive: %+v", saved.Records)
		}
	})
	t.Run("expiry", func(t *testing.T) {
		s, jobs, archive, task, _ := dailyFixture(t)
		if task, due := s.triggerAt(t, jobs, archive, time.UnixMilli(task.ExpiresAtMS), dailyPage); due || task.State != "expired" {
			t.Fatalf("due %v, task %+v", due, task)
		}
	})
}
