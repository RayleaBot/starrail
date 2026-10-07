package app

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/RayleaBot/starrail/internal/gacha"
)

// SyncTask is a daily sync: each day after Hour, Beijing time, the job's
// trigger reads every pool of the role under the task's delegation.
type SyncTask struct {
	Ref string `json:"ref"`
	Selection
	Owner              Subject        `json:"owner"`
	Role               Role           `json:"role"`
	Provider           string         `json:"provider"`
	DelegationRef      string         `json:"delegation_ref"`
	ExpiresAtMS        int64          `json:"expires_at_ms"`
	Hour               int            `json:"hour"`
	Full               bool           `json:"full"`
	Notify             bool           `json:"notify"`
	State              string         `json:"state"`
	NextCheckMS        int64          `json:"next_check_ms"`
	LastCheckedMS      int64          `json:"last_checked_ms"`
	LastFinishedMS     int64          `json:"last_finished_ms"`
	LastNotificationMS int64          `json:"last_notification_ms"`
	RunDay             string         `json:"run_day"`
	Failures           int            `json:"failures"`
	LastCode           string         `json:"last_code"`
	Progress           gacha.SyncInfo `json:"progress"`
}

func (t SyncTask) taskRef() string { return t.Ref }

// SyncTaskStore keeps each daily sync in its own file.
type SyncTaskStore struct {
	taskFiles[SyncTask]
}

func syncTaskStore(directory string) *SyncTaskStore {
	return &SyncTaskStore{taskFiles[SyncTask]{Directory: filepath.Join(directory, "sync-tasks")}}
}

// syncTaskTime is the Beijing day of now, and when hour falls on it and on the
// next day.
func syncTaskTime(now time.Time, hour int) (string, int64, int64) {
	china := now.In(time.FixedZone("UTC+8", 8*3600))
	at := time.Date(china.Year(), china.Month(), china.Day(), hour, 0, 0, 0, china.Location())
	return china.Format("2006-01-02"), at.UnixMilli(), at.AddDate(0, 0, 1).UnixMilli()
}

func syncTaskError(err error) string {
	switch {
	case errors.Is(err, gacha.ErrConflict):
		return "plugin.game_sync_conflict"
	case errors.Is(err, gacha.ErrInvalid):
		return "plugin.game_sync_invalid"
	case errors.Is(err, gacha.ErrSync):
		return "plugin.game_sync_busy"
	}
	return PublicError(err).Code
}

// syncTaskFailure records a failed round: it is tried again after five
// minutes, and after three failures in a row waits for the next day. A
// revoked or unverified account, a record conflict or an invalid page pause
// the task until it is run again explicitly.
func syncTaskFailure(task *SyncTask, code string, now time.Time) {
	task.LastCode = code
	task.Failures++
	task.NextCheckMS = now.Add(5 * time.Minute).UnixMilli()
	switch code {
	case "plugin.account_delegation_denied", "plugin.account_caller_denied", "plugin.account_not_found", "plugin.account_role_denied", "plugin.account_subject_denied", "plugin.upstream_auth_invalid", "plugin.upstream_device_required", "plugin.upstream_challenge_required", "plugin.upstream_gacha_unavailable", "plugin.game_sync_conflict", "plugin.game_sync_invalid":
		task.State = "paused"
	default:
		if task.Failures >= 3 {
			_, _, next := syncTaskTime(now, task.Hour)
			task.State = "waiting"
			task.NextCheckMS = next
		}
	}
}

type SyncTaskFetch func(context.Context, SyncTask, string, string, int) (gacha.RemotePage, error)

// syncTaskGap is the least time between two pages of a daily sync, to stay
// under the official rate limits.
const syncTaskGap = time.Second

// due tells whether a claimed task's round runs on a trigger at now. A task
// not due is given its next check, and an expired one is marked so; a due
// task in the waiting state starts the day's round.
func (s *SyncTaskStore) due(task *SyncTask, now time.Time) (bool, error) {
	ms := now.UnixMilli()
	if task.State != "running" && task.State != "waiting" || task.NextCheckMS > ms {
		return false, nil
	}
	if task.ExpiresAtMS <= ms {
		task.State, task.LastCode = "expired", "expired"
		return false, s.save(*task)
	}
	day, at, next := syncTaskTime(now, task.Hour)
	switch {
	case task.State == "waiting" && ms < at:
		task.NextCheckMS = at
		return false, s.save(*task)
	case task.State == "waiting" && task.RunDay == day:
		task.NextCheckMS = next
		return false, s.save(*task)
	case task.State == "waiting":
		task.State, task.RunDay, task.Failures = "running", day, 0
	}
	return true, nil
}

// run reads a due task's round from its trigger's event: page after page,
// syncTaskGap apart, each page written back with the task so the page shows
// its progress. It stops without writing anything more once the task was
// removed, paused or run again meanwhile. A completed round waits for the
// next day; when the account's chat user is to be told, the notification is
// marked and its text returned, for the trigger to send once it let go of
// the task, and a restart does not send it again.
func (s *SyncTaskStore) run(ctx context.Context, task *SyncTask, jobs *gacha.Syncs, archive *gacha.Store, pause func(context.Context, time.Duration) error, fetch SyncTaskFetch) (string, error) {
	task.LastCheckedMS, task.LastCode = time.Now().UnixMilli(), "sync_running"
	info, err := jobs.Start(archive, gacha.SyncChoice{AccountRef: task.AccountRef, RoleRef: task.RoleRef}, task.Role.UID, task.Role.Region, task.Full)
	if err != nil {
		syncTaskFailure(task, syncTaskError(err), time.Now())
		return "", s.save(*task)
	}
	defer jobs.Forget(info.Ref)
	task.Progress = info
	if err = s.save(*task); err != nil {
		return "", err
	}
	for task.Progress.State == "running" {
		if task.Progress.Pages > 0 {
			err = pause(ctx, syncTaskGap)
		}
		if err == nil {
			info, err = jobs.Step(ctx, archive, info.Ref, info.Sequence, func(ctx context.Context, pool, end string, page int) (gacha.RemotePage, error) {
				return fetch(ctx, *task, pool, end, page)
			})
		}
		if err != nil {
			code := syncTaskError(err)
			if ctx.Err() != nil {
				code = "sync_timeout"
			}
			syncTaskFailure(task, code, time.Now())
			return "", s.save(*task)
		}
		task.Progress, task.Failures = info, 0
		if err = s.save(*task); err != nil {
			return "", err
		}
	}
	now := time.Now()
	_, _, next := syncTaskTime(now, task.Hour)
	task.State, task.NextCheckMS, task.LastFinishedMS, task.LastCode = "waiting", next, now.UnixMilli(), "sync_completed"
	if task.Notify {
		task.LastNotificationMS = now.UnixMilli()
	}
	if err = s.save(*task); err != nil || !task.Notify {
		return "", err
	}
	return syncNotice(task.Role, *task.Progress.Result), nil
}

// notificationFailed marks the notification of the round that finished at
// finished as not sent, unless the task changed since.
func (s *SyncTaskStore) notificationFailed(ref string, finished int64) error {
	return s.edit(ref, func(items *[]SyncTask, i int) error {
		if i >= 0 && (*items)[i].LastFinishedMS == finished && (*items)[i].LastCode == "sync_completed" {
			(*items)[i].LastCode = "sync_completed.notification_failed"
		}
		return nil
	})
}

// PauseArchive pauses the daily syncs of an archive being removed; a round
// reading it stops at its next page.
func (s *SyncTaskStore) PauseArchive(uid, region string) error {
	return s.edit("", func(items *[]SyncTask, _ int) error {
		for i := range *items {
			if t := &(*items)[i]; t.Role.UID == uid && t.Role.Region == region && t.State != "expired" {
				t.State, t.LastCode = "paused", "archive_removed"
			}
		}
		return nil
	})
}
