package app

import (
	"context"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// errBackgroundFull answers a flow the host would not move to the
// background: the plugin already holds its limit of background events.
var errBackgroundFull = gameError("background_full", "后台处理的任务已达上限，请稍后再试。")

// detach moves the event of a flow that may outlast the event deadline to
// the background, ending its delivery with result: what the management page
// receives for a management action, nothing for a chat message. The flow then
// runs in the same handler until the background deadline its ctx carries. A
// refused detach leaves the event in the foreground; the host's limit of
// background events is errBackgroundFull.
func detach(ctx context.Context, event *rayleabot.EventContext, result any) error {
	_, err := event.Detach(ctx, result)
	if err != nil && PublicError(err).Code == "platform.rate_limited" {
		return errBackgroundFull
	}
	return err
}

// clock is the time triggers count by and background flows pace their pages
// by; tests set their own.
type clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
	// Until is ctx ending at deadline.
	Until(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc)
}

func (a *App) now() time.Time {
	if a.clock != nil {
		return a.clock.Now()
	}
	return time.Now()
}

// sleep waits for d or until ctx ends.
func (a *App) sleep(ctx context.Context, d time.Duration) error {
	if a.clock != nil {
		return a.clock.Sleep(ctx, d)
	}
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
