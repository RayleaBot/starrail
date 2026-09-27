package app

import (
	"context"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// runTrigger answers a trigger of a job the plugin created, dispatched by the
// job's task ID: a group's pushes, a daily sync, else a stored reminder or
// timed task. A trigger whose task is not stored (removed, or of a kind the
// plugin no longer schedules) deletes its own job.
func (a *App) runTrigger(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	task := event.Event.TaskID()
	switch {
	case task == "":
		// The host names every trigger's task; there is no job to delete.
		return event.Result(map[string]any{"checked": false})
	case strings.HasPrefix(task, "game.content."):
		return a.runContentSubscription(ctx, event, task)
	case strings.HasPrefix(task, "game.sync."):
		return a.runSyncTask(ctx, event, task)
	}
	return a.runReminder(ctx, event, task)
}

// timedTaskLabels name the timed tasks signinTask creates, by kind.
var timedTaskLabels = map[string]string{"signin": "每日签到", "monthly": "每日月报收集", "community": "米游社任务"}

// reminderJob is the scheduler job of a stored reminder or timed task.
func (a *App) reminderJob(task Reminder) rayleabot.SchedulerCreateRequest {
	cron, label := "*/10 * * * *", "体力提醒"
	switch task.Kind {
	case "challenge":
		challenge, _ := challengeKind(task.ChallengeKind)
		cron, label = "*/5 * * * *", challenge.Label+"挑战提醒"
	case "signin", "monthly":
		label = timedTaskLabels[task.Kind]
	case "community":
		cron, label = "* * * * *", timedTaskLabels[task.Kind]
	}
	return rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: cron, LogLabel: a.Game.Name + label}
}

// syncJob is the scheduler job of a daily sync, which checks each minute
// whether its round is due.
func (a *App) syncJob(task SyncTask) rayleabot.SchedulerCreateRequest {
	return rayleabot.SchedulerCreateRequest{TaskID: task.Ref, Cron: "* * * * *", LogLabel: a.Game.Name + "抽卡每日同步"}
}

// contentJob is the scheduler job of a group's 米游社 pushes.
func (a *App) contentJob(ref string) rayleabot.SchedulerCreateRequest {
	return rayleabot.SchedulerCreateRequest{TaskID: ref, Cron: "*/5 * * * *", LogLabel: a.Game.Name + "米游社推送"}
}
