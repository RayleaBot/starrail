package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// PoolImage is 卡池 as miao's gacha/gacha-info draws it: the banners of a
// version, or those that featured one character or weapon, which show only
// the item's own row unless Detail asks for the whole banner.
type PoolReprint struct {
	Up   bool
	Text string
}

type PoolImage struct {
	Reprint *PoolReprint
	Pools   []PoolInfo
	Item    Entry
	Detail  bool
}

// PoolImageBuilder draws PoolImage.
type PoolImageBuilder func(ImageContext, PoolImage) (Image, bool)

// poolCommand answers miao's 卡池信息: a version's banners (x.x卡池,
// x.x上半卡池) and a character's or weapon's (名称卡池, 名称卡池详情), events
// before chronicled wishes as miao lists poolDetail before mixPoolDetail.
func (a *App) poolCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	page := PoolImage{}
	label := ""
	matches := func(pool PoolInfo) bool { return false }
	switch command {
	case "pool-help":
		pool, current := currentPool(a.Game.Data.Resources.Pools, a.now())
		if pool.Version == "" {
			return event.SendText("暂无卡池资料")
		}
		if !current {
			ended, _, _ := strings.Cut(pool.To, " ")
			notice(ctx, event, "卡池资料只收录到 "+pool.Version+pool.Half+"（"+ended+" 结束），尚未收录当前卡池，以下为最近收录的版本。")
		}
		label = a.Game.Name + pool.Version
		matches = func(p PoolInfo) bool { return p.Version == pool.Version }
	case "calendar":
		version, half := args[0], ""
		if len(args) > 1 && (args[1] == "上半" || args[1] == "下半") {
			half = args[1]
		}
		label = a.Game.Name + version + half
		matches = func(pool PoolInfo) bool { return pool.Version == version && (half == "" || pool.Half == half) }
	default:
		label = args[0]
		page.Detail = len(args) > 1 && (args[1] == "详情" || args[1] == "详细")
		aliases := a.aliasMap(event)
		entry, ok := a.Catalog.Resolve(args[0], "character", aliases)
		if !ok {
			entry, ok = a.Catalog.Resolve(args[0], "weapon", aliases)
		}
		if ok {
			page.Item = entry
			matches = func(pool PoolInfo) bool { return slices.Contains(allBannerNames(pool, entry.Kind), entry.Name) }
		}
	}
	for _, kind := range []string{"event", "chronicled"} {
		for _, pool := range a.Game.Data.Resources.Pools {
			if pool.Kind == kind && matches(pool) {
				page.Pools = append(page.Pools, pool)
			}
		}
	}
	sortPoolsDesc(page.Pools)
	if page.Item.ID != "" {
		page.Reprint = poolReprint(page.Pools, a.now())
	}
	if len(page.Pools) == 0 {
		return event.SendText("未找到 " + label + " 的卡池信息")
	}
	view := View{Title: a.Game.Name + "卡池", Rows: []Row{}}
	if page.Reprint != nil {
		view.Note = page.Reprint.Text
	}
	for _, pool := range page.Pools {
		names := []string{}
		for _, group := range [][]string{pool.Characters5, pool.Characters4, pool.Weapons5, pool.Weapons4} {
			names = append(names, group...)
		}
		view.Rows = append(view.Rows, Row{Label: strings.TrimSpace(pool.Version + " " + pool.Half), Value: strings.Join(names, "、")})
	}
	if a.poolImage != nil {
		if drawn, ok := a.poolImage(a.imageContext(ctx), page); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// currentPool is the banner open at now (China time) or, between banners,
// the next to open; when every banner has ended it is the last one to have
// opened and current is false. The data's times are China time written as
// 2006-01-02 15:04:05, which compare as text.
func currentPool(pools []PoolInfo, now time.Time) (pool PoolInfo, current bool) {
	stamp := now.In(chinaZone).Format(time.DateTime)
	for _, p := range pools {
		switch {
		case p.To >= stamp && (!current || p.From < pool.From):
			pool, current = p, true
		case !current && p.From > pool.From:
			pool = p
		}
	}
	return pool, current
}

// sortPoolsDesc orders numeric versions first, then actual start times.
func sortPoolsDesc(pools []PoolInfo) {
	version := func(s string) int {
		value := 0
		for _, part := range strings.Split(s, ".") {
			n, _ := strconv.Atoi(part)
			value = value*1000 + n
		}
		return value
	}
	slices.SortStableFunc(pools, func(a, b PoolInfo) int {
		if order := cmp.Compare(version(b.Version), version(a.Version)); order != 0 {
			return order
		}
		return cmp.Compare(b.From, a.From)
	})
}

func poolReprint(pools []PoolInfo, now time.Time) *PoolReprint {
	stamp, lastTo := now.In(chinaZone).Format(time.DateTime), ""
	for _, p := range pools {
		if p.From > stamp {
			continue
		}
		if p.To >= stamp {
			return &PoolReprint{Up: true, Text: "该角色/武器正在UP"}
		}
		if p.To > lastTo {
			lastTo = p.To
		}
	}
	if lastTo == "" {
		return nil
	}
	ended, err := time.ParseInLocation(time.DateTime, lastTo, chinaZone)
	if err != nil {
		return nil
	}
	return &PoolReprint{Text: fmt.Sprintf("该角色/武器已 %d 天未复刻", int(now.Sub(ended)/(24*time.Hour)))}
}
