package app

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

func (a *App) manageSync(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	return a.syncAction(ctx, a.accountClient(event), action, input)
}
func (a *App) syncAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (result map[string]any, err error) {
	defer func() { err = syncError(err) }()
	ref := asText(input["ref"])
	switch action {
	case "gacha.sync.start":
		choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
		var response struct {
			Roles []Role `json:"roles"`
		}
		if err := client.call(ctx, "roles", map[string]any{"account_ref": choice.AccountRef, "game": a.Game.ID}, &response); err != nil {
			return nil, err
		}
		var role Role
		for _, candidate := range response.Roles {
			if candidate.Ref == choice.RoleRef && candidate.Game == a.Game.ID {
				role = candidate
				break
			}
		}
		if role.Ref == "" {
			return nil, gameError("role_missing", "请选择已授权的游戏角色。")
		}
		if !overseasGameRegion(role.Region) {
			return nil, gameError("sync_unavailable", "星铁国服完整记录同步暂不可用，请导入 UIGF 或 SRGF 文件。")
		}
		if !syncRegionAllowed(role.Region) {
			return nil, gameError("region_unsupported", "此区服暂未适配官方同步。")
		}
		full, ok := input["full"].(bool)
		if input["full"] != nil && !ok {
			return nil, gameError("input_invalid", "同步模式无效。")
		}
		info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, role.UID, role.Region, full)
		return map[string]any{"sync": info}, err
	case "gacha.sync.step":
		var payload struct {
			Sequence *int `json:"sequence"`
		}
		if decodeObject(input, &payload) != nil || payload.Sequence == nil || *payload.Sequence < 0 {
			return nil, gameError("input_invalid", "同步进度无效。")
		}
		choice, err := a.Syncs.Choice(ref)
		if err != nil || choice.Link {
			return nil, gacha.ErrSync
		}
		info, err := a.Syncs.Step(ctx, a.Gacha, ref, *payload.Sequence, a.syncPage(client, choice))
		return map[string]any{"sync": info}, err
	case "gacha.sync.cancel":
		err := a.Syncs.Cancel(ref)
		return map[string]any{"canceled": err == nil}, err
	default:
		return nil, gameError("operation_denied", "操作不存在。")
	}
}

// syncPage reads a page of the role's official gacha records through the
// account plugin, as the event running the sync.
func (a *App) syncPage(client AccountsClient, choice gacha.SyncChoice) gacha.FetchPage {
	return func(ctx context.Context, pool, endID string, page int) (gacha.RemotePage, error) {
		response, err := client.Execute(ctx, Selection{AccountRef: choice.AccountRef, RoleRef: choice.RoleRef}, a.Game.ID+".gacha", map[string]any{"gacha_type": pool, "end_id": endID, "page": page})
		if err != nil {
			return gacha.RemotePage{}, err
		}
		return gacha.ParsePage(response.Role.UID, response.Role.Region, pool, endID, response.Data)
	}
}

// A background sync is an official sync the management page starts and
// leaves: the management action moves to the background with the page's
// answer and reads every pool in that one event, calling the account plugin
// as the page's user did. It needs neither a scheduled task nor a delegation.

// syncPageGap is the least time between two pages of a background sync, to
// stay under the official rate limits.
const syncPageGap = time.Second

// backgroundSyncsKept is how many background syncs the page can list.
const backgroundSyncsKept = 16

// BackgroundSync is a background sync as the page lists it: the role it
// reads, the chat user told of its result, its state (running, completed,
// failed or canceled), the code of its last outcome and its progress.
type BackgroundSync struct {
	Ref        string         `json:"ref"`
	Role       Role           `json:"role"`
	Owner      Subject        `json:"owner"`
	Full       bool           `json:"full"`
	Notify     bool           `json:"notify"`
	State      string         `json:"state"`
	LastCode   string         `json:"last_code"`
	StartedMS  int64          `json:"started_ms"`
	FinishedMS int64          `json:"finished_ms,omitempty"`
	Progress   gacha.SyncInfo `json:"progress"`
}

// backgroundSyncs are the latest background syncs of this process, newest
// first. They live in memory only: a sync ends with the event running it.
type backgroundSyncs struct {
	mu    sync.Mutex
	items []BackgroundSync
}

// begin lists a starting sync, unless one of the same role is running, and
// drops the oldest finished syncs past backgroundSyncsKept.
func (s *backgroundSyncs) begin(item BackgroundSync) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(s.items, func(other BackgroundSync) bool {
		return other.State == "running" && other.Role.UID == item.Role.UID && other.Role.Region == item.Role.Region
	}) {
		return false
	}
	s.items = append([]BackgroundSync{item}, s.items...)
	for i := len(s.items) - 1; i >= 0 && len(s.items) > backgroundSyncsKept; i-- {
		if s.items[i].State != "running" {
			s.items = slices.Delete(s.items, i, i+1)
		}
	}
	return true
}

// edit changes a listed sync; false when it is not listed.
func (s *backgroundSyncs) edit(ref string, fn func(*BackgroundSync)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.items, func(item BackgroundSync) bool { return item.Ref == ref })
	if i >= 0 {
		fn(&s.items[i])
	}
	return i >= 0
}

// drop removes a sync that never started.
func (s *backgroundSyncs) drop(ref string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = slices.DeleteFunc(s.items, func(item BackgroundSync) bool { return item.Ref == ref })
}

func (s *backgroundSyncs) list() []BackgroundSync {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.items)
}

// cancel marks the running syncs match picks as canceled with code, each to
// stop before its next page; false when none matched.
func (s *backgroundSyncs) cancel(match func(BackgroundSync) bool, code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	canceled := false
	for i := range s.items {
		if item := &s.items[i]; item.State == "running" && match(*item) {
			item.State, item.LastCode, item.FinishedMS = "canceled", code, time.Now().UnixMilli()
			canceled = true
		}
	}
	return canceled
}

// running reports whether a listed sync is still to read its pages.
func (s *backgroundSyncs) running(ref string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.ContainsFunc(s.items, func(item BackgroundSync) bool { return item.Ref == ref && item.State == "running" })
}

// cancelArchiveSyncs cancels the background syncs of an archive being
// removed; a page merged meanwhile finds the archive changed.
func (a *App) cancelArchiveSyncs(uid, region string) {
	a.BackgroundSyncs.cancel(func(item BackgroundSync) bool { return item.Role.UID == uid && item.Role.Region == region }, "archive_removed")
}

func (a *App) backgroundSyncAction(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	switch action {
	case "gacha.task.cancel":
		ref := asText(input["ref"])
		if !a.BackgroundSyncs.cancel(func(item BackgroundSync) bool { return item.Ref == ref }, "sync_canceled") {
			return nil, gameError("sync_task_missing", "没有正在进行的这项后台同步。")
		}
		return map[string]any{"canceled": true}, nil
	case "gacha.task.start":
		return a.startBackgroundSync(ctx, event, input)
	}
	return nil, gameError("operation_denied", "操作不存在。")
}

// startBackgroundSync checks the role, starts its sync and moves the
// management action to the background with the listed sync, then reads the
// pages one after another, syncPageGap apart, until the sync completes,
// fails, the page cancels it or the background deadline passes. A completed
// sync tells the account's chat user when asked.
func (a *App) startBackgroundSync(ctx context.Context, event *rayleabot.EventContext, input map[string]any) (map[string]any, error) {
	var q struct {
		Selection
		Full   bool `json:"full"`
		Notify bool `json:"notify"`
	}
	if input["confirm"] != true || decodeObject(input, &q) != nil {
		return nil, gameError("input_invalid", "请确认开始后台同步。")
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
	choice := gacha.SyncChoice{AccountRef: q.AccountRef, RoleRef: q.RoleRef}
	info, err := a.Syncs.Start(a.Gacha, choice, role.UID, role.Region, q.Full)
	if err != nil {
		return nil, syncError(err)
	}
	defer a.Syncs.Forget(info.Ref)
	item := BackgroundSync{Ref: info.Ref, Role: role, Owner: account.Owner, Full: q.Full, Notify: q.Notify, State: "running", LastCode: "sync_running", StartedMS: time.Now().UnixMilli(), Progress: info}
	if !a.BackgroundSyncs.begin(item) {
		return nil, gameError("sync_task_running", "此角色正在后台同步，请查看进度。")
	}
	if err = detach(ctx, event, map[string]any{"task": item}); err != nil {
		a.BackgroundSyncs.drop(item.Ref)
		return nil, err
	}
	fetch := a.syncPage(client, choice)
	for info.State == "running" {
		if info.Pages > 0 {
			err = a.sleep(ctx, syncPageGap)
		}
		if err != nil || !a.BackgroundSyncs.running(item.Ref) {
			break
		}
		var next gacha.SyncInfo
		if next, err = a.Syncs.Step(ctx, a.Gacha, info.Ref, info.Sequence, fetch); err != nil {
			break
		}
		info = next
		a.BackgroundSyncs.edit(item.Ref, func(listed *BackgroundSync) { listed.Progress = info })
	}
	code := "sync_completed"
	switch {
	case ctx.Err() != nil:
		code = "sync_timeout"
	case err != nil:
		code = PublicError(syncError(err)).Code
	case info.State != "completed":
		// Canceled: the listed sync already says so.
	case q.Notify && a.notifySync(ctx, event, item, *info.Result) != nil:
		code = "sync_completed.notification_failed"
	}
	failed := false
	a.BackgroundSyncs.edit(item.Ref, func(listed *BackgroundSync) {
		// A sync the page or an archive removal canceled keeps that code.
		if listed.State == "running" {
			listed.State, listed.LastCode, listed.FinishedMS, failed = "completed", code, time.Now().UnixMilli(), err != nil
			if failed {
				listed.State = "failed"
			}
		}
		listed.Progress = info
		item = *listed
	})
	if failed {
		return nil, syncError(err)
	}
	return map[string]any{"task": item}, nil
}

// notifySync tells the account's chat user, through the bot the account
// belongs to, what a completed background sync added.
func (a *App) notifySync(ctx context.Context, event *rayleabot.EventContext, item BackgroundSync, result gacha.ImportResult) error {
	owner := item.Owner
	if !slices.ContainsFunc(event.Bots, func(bot rayleabot.Bot) bool {
		return bot.ID == owner.BotID && bot.SourceProtocol == owner.SourceProtocol && bot.SourceAdapter == owner.SourceAdapter
	}) {
		return gameError("bot_missing", "所属机器人暂不可用。")
	}
	text := fmt.Sprintf("抽卡后台同步完成\n%s · %s\n新增 %d 条，档案共 %d 条。", item.Role.Nickname, item.Role.UID, result.Added, result.Total)
	_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: owner.SourceProtocol, SourceAdapter: owner.SourceAdapter, TargetType: "private", TargetID: owner.ActorID, Message: rayleabot.MessageOut{Segments: []rayleabot.Segment{rayleabot.Text(text)}}})
	return err
}
