package app

import (
	"context"
	"math"
	"strconv"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 星琼记录 and 在线时长 follow StarRail-plugin's hkrpg: a page of an asset's
// gains from the official self-service records, and the time online
// estimated from the Trailblaze Power recovered over time.

// payLogTypes are StarRail-plugin's record words with the category they read
// and the English subtitle its page shows. 充值 reads 古老梦华. Its 光锥 and
// 武器 records never answer, as upstream's 跃迁记录 rule takes those words
// first; so here 跃迁记录 does.
var payLogTypes = map[string][3]string{"星琼": {"星琼", "stellar", "Stellar"}, "古老梦华": {"古老梦华", "dreams", "Dreams"}, "充值": {"古老梦华", "dreams", "Dreams"}, "体力": {"体力", "power", "Power"}, "遗器": {"遗器", "relic", "Relic"}}

// PayLogImage is a page of 星琼记录 and the like, as StarRail-plugin's
// charge page shows it.
type PayLogImage struct {
	UID, Type, TypeEN string
	Page              int
	Records           []PayLogRecord
}
type PayLogRecord struct{ Time, Count, Action string }
type PayLogImageBuilder func(ImageContext, PayLogImage) (Image, bool)

// OnlineImage is what 在线时长 draws: the player, the hours online on every
// day since the earliest record, and the last seven days' online blocks.
type OnlineImage struct {
	Role    Role
	Dates   []string
	Hours   []string
	Details []OnlineDay
}
type OnlineDay struct {
	Date   string
	Blocks []OnlineBlock
}
type OnlineBlock struct {
	Online bool
	Width  int
}
type OnlineImageBuilder func(ImageContext, OnlineImage) (Image, bool)

// ownRole is the account role of the UID in use, which the self-service
// records need; missing is upstream's reply when there is none, noKey when
// the UID has no account to read the records with.
func (a *App) ownRole(ctx context.Context, event *rayleabot.EventContext, noKey string) (AccountsClient, Selection, Role, string) {
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return client, Selection{}, Role{}, friendlyError(err)
	}
	uid := listed.UIDBindings[a.Game.ID].Current
	choice, role, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		if uid == "" && len(listed.Items) == 0 {
			return client, Selection{}, Role{}, "尚未绑定uid,请发送" + a.Game.Prefix + "绑定uid进行绑定"
		}
		return client, Selection{}, Role{}, noKey
	}
	return client, choice, role, ""
}

func (a *App) payLogCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	kind, ok := payLogTypes[args[0]]
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	page := 1
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil && n > 0 {
			page = n
		}
	}
	client, choice, role, missing := a.ownRole(ctx, event, "authkey获取失败，请使用扫码登录重新绑定stoken后再进行查看~")
	if missing != "" {
		return event.SendText(missing)
	}
	result, err := client.Execute(ctx, choice, a.Game.ID+".billing", map[string]any{"category": kind[1], "page": page})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	image := PayLogImage{UID: role.UID, Type: kind[0], TypeEN: kind[2], Page: page}
	for _, raw := range asList(result.Data["items"]) {
		item := asObject(raw)
		count, _ := strconv.Atoi(asText(item["add_num"]))
		// Upstream keeps the gains, and of Trailblaze Power more than one.
		if count <= 0 || kind[1] == "power" && count <= 1 {
			continue
		}
		image.Records = append(image.Records, PayLogRecord{Time: asText(item["time"]), Count: strconv.Itoa(count), Action: plainGameText(asText(item["action"]))})
	}
	if len(image.Records) == 0 {
		return event.SendText("暂无记录")
	}
	view := View{Title: kind[0] + "记录", Subtitle: "UID " + role.UID + " · 第" + strconv.Itoa(page) + "页", Rows: []Row{}}
	for _, record := range image.Records {
		view.Rows = append(view.Rows, Row{Label: record.Time, Value: record.Count + " · " + record.Action})
	}
	if a.payLogImage != nil {
		if drawn, ok := a.payLogImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// onlineAction is the Trailblaze Power record StarRail-plugin counts.
const onlineAction = "随时间回复开拓力"

type onlineRecord struct {
	time  time.Time
	count int
}

func (a *App) onlineCommand(ctx context.Context, event *rayleabot.EventContext) error {
	client, choice, role, missing := a.ownRole(ctx, event, "请先使用cookie帮助绑定cookie和星铁绑定uid后再进行查看噢")
	if missing != "" {
		return event.SendText(missing)
	}
	notice(ctx, event, "正在统计中，时间可能比较久请多等一会~")
	// Upstream reads every page of fifty; this stops in time to reply
	// within the event, which only drops the earliest days.
	deadline := time.Now().Add(40 * time.Second)
	// The official list pages by the last record's ID; upstream's page numbers
	// no longer advance.
	records, cursor := []onlineRecord{}, "0"
	for page := 1; time.Now().Before(deadline); page++ {
		result, err := client.Execute(ctx, choice, a.Game.ID+".billing", map[string]any{"category": "power", "page": page, "page_size": 50, "end_id": cursor})
		if err != nil {
			if page == 1 {
				return event.SendText(friendlyError(err))
			}
			break
		}
		for _, raw := range asList(result.Data["items"]) {
			item := asObject(raw)
			if plainGameText(asText(item["action"])) != onlineAction {
				continue
			}
			when, err := time.ParseInLocation(time.DateTime, asText(item["time"]), chinaZone)
			count, _ := strconv.Atoi(asText(item["add_num"]))
			if err == nil {
				records = append(records, onlineRecord{when, count})
			}
		}
		next := asText(result.Data["next_end_id"])
		if result.Data["has_more"] != true || next == "" || next == cursor {
			break
		}
		cursor = next
		time.Sleep(500 * time.Millisecond)
	}
	if len(records) == 0 {
		return event.SendText("暂无记录")
	}
	today := a.now().In(chinaZone)
	image := OnlineImage{Role: role, Details: onlinePeriods(records, today)}
	image.Dates, image.Hours = onlineDays(records, today)
	view := View{Title: "在线时长", Subtitle: "UID " + role.UID, Rows: []Row{}, Note: "数据由米游社开拓力回复推算估计，不保证准确性，仅供参考"}
	for i := max(0, len(image.Dates)-7); i < len(image.Dates); i++ {
		view.Rows = append(view.Rows, Row{Label: image.Dates[i], Value: image.Hours[i] + " 小时"})
	}
	if a.onlineImage != nil {
		if drawn, ok := a.onlineImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

var chinaZone = time.FixedZone("UTC+8", 8*3600)

// onlineDays is StarRail-plugin's statisticsOnlineDateGeneral: every day from
// the earliest record to today, six minutes online for each point recovered
// alone, in hours with one decimal.
func onlineDays(records []onlineRecord, today time.Time) ([]string, []string) {
	minutes := map[string]int{}
	first := today
	for _, record := range records {
		if record.time.Before(first) {
			first = record.time
		}
		if record.count == 1 {
			minutes[record.time.Format(time.DateOnly)] += 6
		}
	}
	dates, hours := []string{}, []string{}
	end := today.Format(time.DateOnly)
	for day := first; day.Format(time.DateOnly) <= end; day = day.AddDate(0, 0, 1) {
		date := day.Format(time.DateOnly)
		dates = append(dates, date)
		hours = append(hours, strconv.FormatFloat(float64(minutes[date])/60, 'f', 1, 64))
	}
	return dates, hours
}

// onlinePeriods is StarRail-plugin's statisticOnlinePeriods: for the last
// seven days, today first, each point recovered alone marks the six minutes
// before it online, and each day becomes online and offline blocks 650px
// wide in all. Upstream's minute scan leaves each block one minute short,
// which is kept; a point just after midnight also marks the end of the day
// before, which upstream meant but reached the wrong date for.
func onlinePeriods(records []onlineRecord, today time.Time) []OnlineDay {
	dates := []string{}
	periods := map[string][][2]int{}
	for i := range 7 {
		date := today.AddDate(0, 0, -i).Format(time.DateOnly)
		dates = append(dates, date)
		periods[date] = [][2]int{}
	}
	for _, record := range records {
		if record.count != 1 {
			continue
		}
		date := record.time.Format(time.DateOnly)
		if _, ok := periods[date]; !ok {
			break
		}
		end := record.time.Hour()*60 + record.time.Minute()
		if end >= 6 {
			periods[date] = append(periods[date], [2]int{end - 6, end})
			continue
		}
		periods[date] = append(periods[date], [2]int{0, end})
		yesterday := record.time.AddDate(0, 0, -1).Format(time.DateOnly)
		if _, ok := periods[yesterday]; ok {
			periods[yesterday] = append(periods[yesterday], [2]int{1440 - (6 - end), 1440})
		}
	}
	days := []OnlineDay{}
	for _, date := range dates {
		days = append(days, OnlineDay{Date: date, Blocks: onlineBlocks(periods[date])})
	}
	return days
}

// onlineBlocks runs upstream's scan over one day's periods, newest first.
func onlineBlocks(periods [][2]int) []OnlineBlock {
	online := make([]bool, 1441)
	for _, period := range periods {
		for minute := period[0]; minute < period[1]; minute++ {
			online[minute] = true
		}
	}
	type run struct {
		online     bool
		start, end int
	}
	runs := []run{}
	state, current := 0, run{}
	for cursor := 0; cursor <= 1440; cursor++ {
		switch {
		case online[cursor] && state <= 0, !online[cursor] && state >= 0:
			if current.start != current.end {
				runs = append(runs, current)
			}
			current = run{online: online[cursor], start: cursor, end: cursor}
			state = map[bool]int{true: 1, false: -1}[online[cursor]]
		default:
			current.end++
		}
	}
	if current.start != current.end {
		runs = append(runs, current)
	}
	blocks := []OnlineBlock{}
	for _, run := range runs {
		// Upstream's floating point: 1008 minutes come to 454px, not 455.
		blocks = append(blocks, OnlineBlock{Online: run.online, Width: int(math.Floor(float64(run.end-run.start) / 1440 * 650))})
	}
	return blocks
}
