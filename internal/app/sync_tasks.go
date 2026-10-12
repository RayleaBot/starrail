package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/gacha"
)

func syncTaskID(game, provider string, choice Selection) string {
	sum := sha256.Sum256([]byte(provider + "\x00" + choice.AccountRef + "\x00" + choice.RoleRef))
	return "game.sync." + game + "." + hex.EncodeToString(sum[:])
}

// syncTaskAction manages the daily syncs: create, list, run again and
// remove. The list also carries the background syncs started on demand.
func (a *App) syncTaskAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	switch action {
	case "gacha.task.list":
		items, err := a.SyncTasks.List()
		items = slices.DeleteFunc(items, func(item SyncTask) bool { return !visibleOwner(event, item.Owner) })
		syncs := slices.DeleteFunc(a.BackgroundSyncs.list(), func(item BackgroundSync) bool { return !visibleOwner(event, item.Owner) })
		return map[string]any{"items": items, "syncs": syncs}, err
	case "gacha.task.remove":
		var t SyncTask
		err := a.SyncTasks.edit(asText(input["ref"]), func(items *[]SyncTask, i int) error {
			if i < 0 {
				return gameError("sync_task_missing", "同步任务不存在。")
			}
			t = (*items)[i]
			if !visibleOwner(event, t.Owner) {
				return accountMismatch()
			}
			*items = slices.Delete(*items, i, i+1)
			return nil
		})
		if err != nil {
			return map[string]any{"removed": false, "delegation_revoked": false}, err
		}
		_, _ = event.Actions().SchedulerDelete(ctx, t.Ref)
		client := AccountsClient{Caller: event.Actions(), Provider: t.Provider, Game: a.Game.ID}
		revoked := client.call(ctx, "delegation.revoke", map[string]any{"account_ref": t.AccountRef, "delegation_ref": t.DelegationRef}, nil) == nil
		return map[string]any{"removed": true, "delegation_revoked": revoked}, nil
	case "gacha.task.run":
		if input["confirm"] != true {
			return nil, gameError("input_invalid", "请明确重新运行每日同步。")
		}
		return a.rerunSyncTask(ctx, event, asText(input["ref"]))
	case "gacha.task.create":
		return a.createSyncTask(ctx, event, input)
	}
	return nil, gameError("operation_denied", "操作不存在。")
}

// rerunSyncTask checks the account again and queues the task's round for
// the next trigger.
func (a *App) rerunSyncTask(ctx context.Context, event *rayleabot.EventContext, ref string) (map[string]any, error) {
	items, err := a.SyncTasks.List()
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(items, func(t SyncTask) bool { return t.Ref == ref })
	if i < 0 {
		return nil, gameError("sync_task_missing", "同步任务不存在。")
	}
	if !visibleOwner(event, items[i].Owner) {
		return nil, accountMismatch()
	}
	// The account check runs before the task is edited, outside the store's
	// lock.
	client := AccountsClient{Caller: event.Actions(), Provider: items[i].Provider, Game: a.Game.ID}
	_, role, err := client.Authorize(ctx, items[i].Selection)
	if err != nil {
		return nil, err
	}
	err = a.SyncTasks.edit(ref, func(items *[]SyncTask, i int) error {
		if i < 0 {
			return gameError("sync_task_missing", "同步任务不存在。")
		}
		t := &(*items)[i]
		if !visibleOwner(event, t.Owner) {
			return accountMismatch()
		}
		if t.State == "running" || t.State == "creating" {
			return gameError("sync_task_running", "此任务正在处理，请查看进度。")
		}
		if t.ExpiresAtMS <= time.Now().UnixMilli() || t.DelegationRef == "" {
			return gameError("sync_task_expired", "委托已过期，请移除后重新创建任务。")
		}
		if role.UID != t.Role.UID || role.Region != t.Role.Region {
			return gameError("role_missing", "账号角色已变更，请重新创建任务。")
		}
		if !syncRegionAllowed(role.Region) {
			return gameError("sync_unavailable", "此区服暂不可同步完整记录。")
		}
		t.Progress = gacha.SyncInfo{}
		t.State, t.Failures, t.NextCheckMS, t.LastCode = "running", 0, 0, "queued"
		t.RunDay, _, _ = syncTaskTime(time.Now(), t.Hour)
		return nil
	})
	return map[string]any{"queued": err == nil}, err
}

// createSyncTask creates a daily sync with a delegation of the role's gacha
// records for days and its scheduled job.
func (a *App) createSyncTask(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	var q struct {
		Selection
		Hour   int  `json:"hour"`
		Days   int  `json:"days"`
		Full   bool `json:"full"`
		Notify bool `json:"notify"`
	}
	if input["confirm"] != true || decodeObject(input, &q) != nil || q.Days < 1 || q.Days > 90 || q.Hour < 0 || q.Hour > 23 {
		return nil, gameError("input_invalid", "请选择 1–90 天有效期和 0–23 时，并确认开启每日同步。")
	}
	client := a.accountClient(event)
	account, role, err := client.Authorize(ctx, q.Selection)
	if err != nil {
		return nil, err
	}
	if !overseasGameRegion(role.Region) {
		return nil, gameError("sync_unavailable", "星铁国服完整抽卡同步暂不可用，请导入 UIGF 或 SRGF。")
	}
	if !syncRegionAllowed(role.Region) {
		return nil, gameError("region_unsupported", "此区服暂未适配后台同步。")
	}
	task := SyncTask{Ref: syncTaskID(a.Game.ID, client.Provider, q.Selection), Selection: q.Selection, Owner: account.Owner, Role: role, Provider: client.Provider, Hour: q.Hour, Full: q.Full, Notify: q.Notify, State: "creating"}
	// The task is kept as "creating" while the delegation and job are made
	// outside the store's lock, then enabled or dropped.
	err = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
		if len(*items) >= 256 {
			return gameError("sync_task_limit", "每日同步任务已达上限。")
		}
		if i >= 0 {
			return gameError("sync_task_exists", "此角色已有每日同步，请重新运行已有任务，或移除后调整设置。")
		}
		*items = append(*items, task)
		return nil
	})
	if err != nil {
		return nil, err
	}
	var grant struct {
		Delegation struct {
			Ref         string `json:"ref"`
			ExpiresAtMS int64  `json:"expires_at_ms"`
		} `json:"delegation"`
	}
	err = client.call(ctx, "delegation.create", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "task_id": task.Ref, "operation": a.Game.ID + ".gacha", "days": q.Days}, &grant)
	if err == nil {
		task.DelegationRef = grant.Delegation.Ref
		task.ExpiresAtMS = grant.Delegation.ExpiresAtMS
		_, err = event.Actions().SchedulerCreate(ctx, a.syncJob(task))
	}
	if err == nil {
		task.State, task.LastCode = "waiting", "queued"
		err = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
			if i < 0 {
				return gameError("sync_task_missing", "同步任务创建已取消。")
			}
			(*items)[i] = task
			return nil
		})
	}
	if err != nil {
		_ = a.SyncTasks.edit(task.Ref, func(items *[]SyncTask, i int) error {
			if i >= 0 {
				*items = slices.Delete(*items, i, i+1)
			}
			return nil
		})
		if task.DelegationRef != "" {
			_ = client.call(ctx, "delegation.revoke", map[string]any{"account_ref": task.AccountRef, "delegation_ref": task.DelegationRef}, nil)
		}
		return nil, err
	}
	return map[string]any{"task": task}, nil
}

// runSyncTask answers a trigger of a daily sync. A due round moves the
// trigger to the background and reads every page there, under the task's
// delegation; when the host refuses, as the plugin holds its limit of
// background events, the next trigger tries again. Another trigger still
// reading the task leaves it alone. The trigger lets go of the task before it
// sends anything: the host hands over the next trigger, and the owner may
// act, as soon as the notification or the trigger's result arrives.
func (a *App) runSyncTask(ctx context.Context, event *rayleabot.EventContext, ref string) error {
	task, claimed, err := a.SyncTasks.claim(ref)
	switch {
	case errors.Is(err, errTaskMissing):
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
		return event.Result(map[string]any{"checked": false})
	case err != nil:
		failure := PublicError(err)
		return event.Fail(failure.Code, failure.Message)
	case !claimed:
		return event.Result(map[string]any{"checked": false})
	}
	release := sync.OnceFunc(func() { a.SyncTasks.release(ref) })
	defer release()
	due, err := a.SyncTasks.due(&task, time.Now())
	if err != nil || !due || detach(ctx, event, nil) != nil {
		release()
		return event.Result(map[string]any{"checked": err == nil})
	}
	notice, err := a.SyncTasks.run(ctx, &task, &a.Syncs, a.Gacha, a.sleep, func(ctx context.Context, task SyncTask, pool, end string, page int) (gacha.RemotePage, error) {
		client := AccountsClient{Caller: event.Actions(), Provider: task.Provider, Game: a.Game.ID}
		var response QueryResult
		err := client.call(ctx, "execute", map[string]any{"account_ref": task.AccountRef, "role_ref": task.RoleRef, "operation": a.Game.ID + ".gacha", "delegation_ref": task.DelegationRef, "input": map[string]any{"gacha_type": pool, "end_id": end, "page": page}}, &response)
		if err != nil {
			return gacha.RemotePage{}, err
		}
		if response.Role.UID != task.Role.UID || response.Role.Region != task.Role.Region || response.Role.Ref != task.Role.Ref {
			return gacha.RemotePage{}, gacha.ErrInvalid
		}
		return gacha.ParsePage(response.Role.UID, response.Role.Region, pool, end, response.Data)
	})
	release()
	if err == nil && notice != "" && tellOwner(ctx, event, task.Owner, notice) != nil {
		err = a.SyncTasks.notificationFailed(ref, task.LastFinishedMS)
	}
	if err != nil && !errors.Is(err, errTaskChanged) {
		failure := PublicError(err)
		return event.Fail(failure.Code, failure.Message)
	}
	return event.Result(map[string]any{"checked": true})
}
