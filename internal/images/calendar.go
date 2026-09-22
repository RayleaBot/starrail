package images

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// Miao's Star Rail calendar skips these announcements by ID and by title.
var (
	calendarIgnoreIDs = []string{"257", "194", "203", "183", "187", "185", "171"}
	calendarIgnore    = regexp.MustCompile(`(更新概览|游戏优化|优化说明|内容专题页|专题展示页|版本更新说明|循星归程|调研|防沉迷|米游社|专项意见|更新修复与优化|问卷调查|版本更新通知|更新时间说明|预下载功能|周边限时|周边上新|角色演示|角色PV|版本PV|动画短片|bilibili|激励计划|调整说明|攻略征集|测试招募)`)
)

// The patterns miao reads Star Rail announcement times with.
var (
	calendarVersionTitle = regexp.MustCompile(`(\d\.\d)版本\S*更新(概览|说明)`)
	calendarUpdateTime   = regexp.MustCompile(`■(?:更新时间)\s*([^■]+)(?:■|$)`)
	calendarTimeText     = regexp.MustCompile(`([0-9/: ]){9,}`)
	calendarTags         = regexp.MustCompile(`(<|&lt;)[\w "%:;=\-/(),.]+(>|&gt;)`)
	calendarSection      = regexp.MustCompile(`(?:活动时间|活动跃迁|开放时间|开启时间|折扣时间|上架时间)\s*※([^※]+)(※|$)`)
	calendarRange        = regexp.MustCompile(`(?:活动时间)?(?:※|\s)*([0-9/: -]{6,})`)
	calendarAfterVersion = regexp.MustCompile(`(\d\.\d)版本更新后`)
	calendarVersionAnn   = regexp.MustCompile(`版本更新(概览|说明)`)
)

// calendarArtwork maps the images the calendar page names to miao's paths;
// the page is drawn on miao's default hydro background.
var calendarArtwork = [][2]string{
	{"NZBZ", "resources/common/font/NZBZ.woff"},
	{"YS", "resources/common/font/HYWH-65W.woff"},
	{"Number", "resources/common/font/tttgbnumber.woff"},
	{"common-bg-bg-hydro", "resources/common/bg/bg-hydro.webp"},
	{"common-bg-talent-hydro", "resources/common/bg/talent-hydro.webp"},
	{"common-cont-card-bg", "resources/common/cont/card-bg.png"},
	{"wiki-imgs-abyss-1", "resources/wiki/imgs/abyss-1.jpg"},
	{"wiki-imgs-abyss-2", "resources/wiki/imgs/abyss-2.jpg"},
	{"wiki-imgs-abyss-4", "resources/wiki/imgs/abyss-4.jpg"},
	{"abyss-1-icon", "resources/wiki/imgs/abyss-1-icon.png"},
	{"abyss-2-icon", "resources/wiki/imgs/abyss-2-icon.png"},
	{"abyss-3-icon", "resources/wiki/imgs/abyss-3-icon.png"},
	{"abyss-4-icon", "resources/wiki/imgs/abyss-4-icon.png"},
	{"calendar-icon", "resources/wiki/imgs/calendar-icon.png"},
}

// calendarRotations are the three challenge modes' 42-day rotations, from
// the start miao counts each from, with the bar type that styles it.
var calendarRotations = []struct {
	title, kind string
	base        time.Time
}{
	{"「混沌回忆」", "abyss-1", time.Date(2026, 4, 13, 4, 0, 0, 0, chinaTime)},
	{"「虚构叙事」", "abyss-2", time.Date(2026, 3, 30, 4, 0, 0, 0, chinaTime)},
	{"「末日幻影」", "abyss-3", time.Date(2026, 3, 16, 4, 0, 0, 0, chinaTime)},
}

var calendarWeekdays = []string{"一", "二", "三", "四", "五", "六", "日"}

// calendarItem is an announcement placed on the calendar.
type calendarItem struct {
	sort         int
	title, kind  string
	merge        int
	banner, icon string
	left, width  float64
	label        string
	duration     time.Duration
	start        string
	index        int
}

// Calendar draws 日历 the way miao's Star Rail calendar does on the shared
// wiki/calendar page: the thirteen days from six days ago, the events, warps
// and notices of the announcements as bars across those days (two that do not
// overlap share a row), the three challenge modes' rotations and this
// version's Anomaly Arbitration, and the current time. 日历列表 draws the
// list layout. Miao passes each rotation's bar type as a function, so its bars
// lose their styling and icon; this gives each the type miao names.
func Calendar(context app.ImageContext, calendar app.CalendarImage) (app.Image, bool) {
	announcements := calendar.Announcements
	if len(announcements.Groups) == 0 {
		return app.Image{}, false
	}
	now := context.Now.In(chinaTime)
	times, gacha := calendarTimes(append(slices.Clone(announcements.Contents), announcements.PictureContents...))
	first := time.Date(now.Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, chinaTime)
	last := time.Date(now.Year(), now.Month(), now.Day()+6, 23, 59, 59, 0, chinaTime)
	total := float64(last.Sub(first))
	resources := &app.ImageResources{Context: context}
	for _, item := range calendarArtwork {
		resources.Artwork(item[0], "miao-plugin", item[1])
	}
	// The banners, icons and warp pictures download together on the first
	// draw.
	urls := []any{}
	for _, url := range gacha {
		urls = append(urls, url)
	}
	groups := slices.Clone(announcements.Groups)
	for _, raw := range announcements.PictureGroups {
		group, _ := raw.(map[string]any)
		types, _ := group["type_list"].([]any)
		groups = append(groups, types...)
	}
	for _, group := range groups {
		for _, ann := range calendarGroup(group) {
			urls = append(urls, ann["banner"], ann["tag_icon"])
		}
	}
	resources.Prefetch("mihoyo", urls...)
	place := func(ann map[string]any, target *[]calendarItem, kind string) {
		id, title := app.Text(ann["ann_id"]), app.Text(ann["title"])
		if title == "" || slices.Contains(calendarIgnoreIDs, id) || calendarIgnore.MatchString(title) {
			return
		}
		item := calendarItem{title: title, kind: kind, sort: 5}
		banner := app.Text(ann["banner"])
		if item.kind == "" {
			item.kind = "activity"
		}
		// Warp banners take the warp announcements' pictures in order.
		switch {
		case strings.Contains(title, "流光定影"):
			item.kind, item.sort, banner = "weapon", 2, shift(&gacha)
		case strings.Contains(title, "活动跃迁"):
			item.kind, item.sort, banner = "character", 1, shift(&gacha)
		case strings.Contains(title, "无名勋礼"):
			item.kind = "pass"
		}
		window := times[id]
		sDate := calendarDate(window[0], app.Text(ann["start_time"]))
		eDate := calendarDate(window[1], app.Text(ann["end_time"]))
		sTime, eTime := sDate, eDate
		if sTime.Before(first) {
			sTime = first
		}
		if eTime.After(last) {
			eTime = last
		}
		item.left = float64(sTime.Sub(first)) / total * 100
		item.width = float64(eTime.Sub(first))/total*100 - item.left
		switch {
		case eDate.Sub(sDate) > 365*24*time.Hour:
			item.label = "永久有效"
			if sDate.Before(now) {
				item.label = sDate.Format("01-02 15:04") + " 后永久有效"
			}
		case now.After(sDate) && eDate.After(now):
			item.label = eDate.Format("01-02 15:04") + " (" + humanize(eDate.Sub(now)) + "后结束)"
			if item.width > 38 {
				item.label = sDate.Format("01-02 15:04") + " ~ " + item.label
			}
		case sDate.After(now):
			item.label = sDate.Format("01-02 15:04") + " (" + humanize(sDate.Sub(now)) + "后开始)"
		default:
			item.label = sDate.Format("01-02 15:04") + " ~ " + eDate.Format("01-02 15:04")
		}
		if sDate.After(last) || eDate.Before(first) {
			return
		}
		if banner != "" {
			item.banner = resources.URL("mihoyo", banner)
		}
		if app.Text(ann["tag_icon"]) != "" {
			item.icon = resources.URL("mihoyo", ann["tag_icon"])
		}
		if item.kind == "activity" {
			item.merge = 1
		}
		item.duration, item.start = eTime.Sub(sTime), sDate.Format("01-02 15:04")
		*target = append(*target, item)
	}
	list, abyss := []calendarItem{}, []calendarItem{}
	for _, raw := range calendarGroup(announcements.Groups[0]) {
		place(raw, &list, "")
	}
	if len(announcements.PictureGroups) > 0 {
		group, _ := announcements.PictureGroups[0].(map[string]any)
		types, _ := group["type_list"].([]any)
		for index := range min(2, len(types)) {
			for _, raw := range calendarGroup(types[index]) {
				place(raw, &list, "")
			}
		}
	}
	version := [2]string{}
	for _, raw := range calendarGroup(announcements.Groups[0]) {
		if calendarVersionAnn.MatchString(app.Text(raw["title"])) {
			version = [2]string{app.Text(raw["start_time"]), app.Text(raw["end_time"])}
		}
	}
	for _, period := range calendarAbyss(now, last, version) {
		place(map[string]any{"title": period.title, "start_time": period.start.Format("2006-01-02 15:04"), "end_time": period.end.Format("2006-01-02 15:04")}, &abyss, period.kind)
	}
	// Miao sorts by kind, then start as text, then duration.
	slices.SortStableFunc(list, func(a, b calendarItem) int {
		if a.sort != b.sort {
			return a.sort - b.sort
		}
		if a.start != b.start {
			return strings.Compare(a.start, b.start)
		}
		return int(a.duration - b.duration)
	})
	characterCount, characterOld, weaponCount := 0, 0, 0
	rows := []any{}
	for index := range list {
		item := &list[index]
		switch item.kind {
		case "character":
			characterCount++
			if item.left == 0 {
				characterOld++
			}
			item.index = characterCount
		case "weapon":
			weaponCount++
			item.index = weaponCount
		}
		if item.merge == 1 {
			for other := range list {
				if list[other].merge == 1 && item.left+item.width <= list[other].left {
					item.merge, list[other].merge = 2, 2
					rows = append(rows, []any{calendarView(*item, 0), calendarView(list[other], 1)})
					break
				}
			}
		}
		if item.merge != 2 {
			item.merge = 2
			rows = append(rows, []any{calendarView(*item, 0)})
		}
	}
	abyssViews := []any{}
	for _, item := range abyss {
		abyssViews = append(abyssViews, calendarView(item, 0))
	}
	months := []any{}
	var dates []any
	for index := range 13 {
		day := first.AddDate(0, 0, index)
		if index > 0 && day.Month() != day.AddDate(0, 0, -1).Month() {
			months = append(months, map[string]any{"month": int(day.AddDate(0, 0, -1).Month()), "dates": dates})
			dates = nil
		}
		dates = append(dates, map[string]any{"date": day.Day(), "week": calendarWeekdays[(int(day.Weekday())+6)%7], "current": day.Day() == now.Day()})
	}
	months = append(months, map[string]any{"month": int(last.Month()), "dates": dates})
	// The list layout is narrower, so it has its own template width.
	mode, template := "calendar", "calendar"
	if strings.HasSuffix(calendar.Word, "日历列表") || strings.HasSuffix(calendar.Word, "活动") {
		mode, template = "list", "calendar-list"
	}
	return app.Image{Template: template, Data: map[string]any{
		"mode": mode, "months": months, "list_class": fmt.Sprintf("char-%d-%d char-num-", characterCount, characterOld),
		"rows": rows, "abyss": abyssViews, "now_left": fmt.Sprintf("%.4f", float64(now.Sub(first))/total*100), "now_time": now.Format("2006-01-02 15:04"),
	}, Resources: resources.List}, true
}

func shift(queue *[]string) string {
	if len(*queue) == 0 {
		return ""
	}
	head := (*queue)[0]
	*queue = (*queue)[1:]
	return head
}

// calendarView is what the page shows of an item; column is its place in a
// shared row.
func calendarView(item calendarItem, column int) map[string]any {
	classes := []string{"type-" + item.kind}
	if item.index > 0 {
		classes = append(classes, "li-idx-"+strconv.Itoa(item.index))
	}
	if item.width < 20 {
		classes = append(classes, "small-mode")
	}
	classes = append(classes, "li-col"+strconv.Itoa(column))
	return map[string]any{"class": strings.Join(classes, " "), "kind": item.kind, "left": fmt.Sprintf("%.4f", item.left), "width": fmt.Sprintf("%.4f", item.width),
		"title": item.title, "label": item.label, "banner": item.banner, "icon": item.icon}
}

// calendarGroup is one group of an announcement list.
func calendarGroup(raw any) []map[string]any {
	group, _ := raw.(map[string]any)
	list, _ := group["list"].([]any)
	out := []map[string]any{}
	for _, item := range list {
		if ann, ok := item.(map[string]any); ok {
			out = append(out, ann)
		}
	}
	return out
}

// calendarTimes reads each announcement's window from its body the way
// miao's Star Rail reqCalData does, and collects the pictures of the warp
// announcements in order.
func calendarTimes(contents []any) (map[string][2]string, []string) {
	versions := map[string]string{"1.1": "2022-06-07 11:00:00"}
	for _, raw := range contents {
		ann, _ := raw.(map[string]any)
		match := calendarVersionTitle.FindStringSubmatch(app.Text(ann["title"]))
		if match == nil {
			continue
		}
		if section := calendarUpdateTime.FindStringSubmatch(app.Text(ann["content"])); section != nil {
			if found := calendarTimeText.FindString(section[1]); found != "" && versions[match[1]] == "" {
				versions[match[1]] = strings.Replace(found, "06:00", "11:00", 1)
			}
		}
	}
	times, gacha := map[string][2]string{}, []string{}
	for _, raw := range contents {
		ann, _ := raw.(map[string]any)
		if calendarIgnore.MatchString(app.Text(ann["title"])) {
			continue
		}
		content := strings.ReplaceAll(strings.ReplaceAll(app.Text(ann["content"]), `<h1 style="">`, "※"), "</h1>", "※")
		section := calendarSection.FindStringSubmatch(calendarTags.ReplaceAllString(content, ""))
		if section == nil || section[1] == "" {
			continue
		}
		if strings.Contains(section[0], "活动跃迁") {
			gacha = append(gacha, app.Text(ann["img"]))
		}
		var span []string
		match := calendarRange.FindStringSubmatch(section[1])
		if version := calendarAfterVersion.FindStringSubmatch(section[1]); version != nil {
			start := versions[version[1]]
			if start == "" {
				continue
			}
			if strings.Contains(section[1], "永久开放") {
				span = []string{start, "2099/01/01 00:00:00"}
			} else if found := calendarTimeText.FindString(section[1]); found != "" {
				span = []string{start, found}
			}
		} else if match != nil {
			span = strings.Split(match[1], "-")
		}
		if len(span) == 2 {
			times[app.Text(ann["ann_id"])] = [2]string{strings.ReplaceAll(strings.TrimSpace(span[0]), "/", "-"), strings.ReplaceAll(strings.TrimSpace(span[1]), "/", "-")}
		}
	}
	return times, gacha
}

// calendarDate is miao's getDate: the body's time when it is longer than six
// characters, otherwise the list's.
func calendarDate(body, list string) time.Time {
	text := list
	if len(body) > 6 {
		text = body
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-1-2 15:04:05", "2006-1-2 15:04", "2006-01-02"} {
		if at, err := time.ParseInLocation(layout, strings.TrimSpace(text), chinaTime); err == nil {
			return at
		}
	}
	return time.Time{}
}

// humanize is moment's duration.humanize() in the zh-cn locale.
func humanize(span time.Duration) string {
	ms := math.Abs(float64(span.Milliseconds()))
	seconds, minutes, hours := math.Round(ms/1000), math.Round(ms/6e4), math.Round(ms/36e5)
	days := math.Round(ms / 864e5)
	months := math.Round(ms / 864e5 * 4800 / 146097)
	years := math.Round(ms / 864e5 * 4800 / 146097 / 12)
	switch {
	case seconds < 45:
		return "几秒"
	case minutes <= 1:
		return "1 分钟"
	case minutes < 45:
		return fmt.Sprintf("%.0f 分钟", minutes)
	case hours <= 1:
		return "1 小时"
	case hours < 22:
		return fmt.Sprintf("%.0f 小时", hours)
	case days <= 1:
		return "1 天"
	case days < 26:
		return fmt.Sprintf("%.0f 天", days)
	case months <= 1:
		return "1 个月"
	case months < 11:
		return fmt.Sprintf("%.0f 个月", months)
	case years <= 1:
		return "1 年"
	}
	return fmt.Sprintf("%.0f 年", years)
}

type calendarPeriod struct {
	start, end  time.Time
	title, kind string
}

// calendarAbyss is miao's Star Rail getAbyssCal: each mode's rotation
// counted back from this week's Monday, and Anomaly Arbitration over the
// version, kept when it is running or starts before the window ends.
func calendarAbyss(now, last time.Time, version [2]string) []calendarPeriod {
	// moment's day(1): the Monday of the Sunday-to-Saturday week.
	monday := time.Date(now.Year(), now.Month(), now.Day()+1-int(now.Weekday()), 4, 0, 0, 0, chinaTime)
	periods := []calendarPeriod{}
	for _, rotation := range calendarRotations {
		start := monday
		days := int(rotation.base.Sub(monday).Hours() / 24)
		if offset := days % 42; offset != 0 {
			if offset < 0 {
				offset = -offset
			}
			start = time.Date(monday.Year(), monday.Month(), monday.Day()-offset, 4, 0, 0, 0, chinaTime)
		}
		periods = append(periods, calendarPeriod{start, start.AddDate(0, 0, 42), rotation.title, rotation.kind})
	}
	if start, end := calendarDate(version[0], ""), calendarDate(version[1], ""); !start.IsZero() && !end.IsZero() {
		periods = append(periods, calendarPeriod{time.Date(start.Year(), start.Month(), start.Day(), 11, start.Minute(), start.Second(), 0, chinaTime),
			time.Date(end.Year(), end.Month(), end.Day(), 3, 59, end.Second(), 0, chinaTime), "「异相仲裁」", "abyss-4"})
	}
	out := []calendarPeriod{}
	for _, period := range periods {
		if (!period.start.After(now) && !now.After(period.end)) || (!now.After(period.start) && !period.start.After(last)) {
			out = append(out, period)
		}
	}
	return out
}
