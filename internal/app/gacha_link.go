package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

// A warp link pasted in chat is read as Yunzai's gcLog reads a Star Rail
// link: its authkey fetches the records from the official warp history page,
// the UID comes from the records and becomes the sender's current UID, and in
// a group the sender is asked to recall the link. No account is needed. As
// StarRail-plugin binds a link to its sender for 更新跃迁记录, the last link a
// sender pasted is kept in memory for a day, the authkey's lifetime; it is
// never written to disk. A history too long for one event is finished by a
// scheduled task, which is dropped if the plugin restarts.

const (
	gachaLinkCN = "https://public-operation-hkrpg.mihoyo.com/common/gacha_record/api/"
	gachaLinkOS = "https://public-operation-hkrpg-sg.hoyoverse.com/common/gacha_record/api/"
	// gachaLinkTask prefixes the scheduled tasks that finish long histories.
	gachaLinkTask = "game.link."
)

// gachaLinkPools are the pools Yunzai reports after a link, in its order.
var gachaLinkPools = [][2]string{{"11", "角色"}, {"12", "光锥"}, {"21", "角色联动"}, {"22", "光锥联动"}, {"1", "常驻"}, {"2", "新手"}}

type gachaLink struct {
	key, region, biz string
}

// gachaLinkJob is a link whose records are still being fetched. update marks
// a 更新跃迁记录, answered with StarRail-plugin's replies instead of Yunzai's.
type gachaLinkJob struct {
	link     gachaLink
	uid      string
	sync     string
	sequence int
	before   map[string]int
	owner    Subject
	target   rayleabot.Target
	update   bool
	expires  time.Time
}

// ownedLink is the last link a sender pasted, for 更新跃迁记录.
type ownedLink struct {
	link    gachaLink
	uid     string
	expires time.Time
}

type gachaLinkJobs struct {
	mu   sync.Mutex
	jobs map[string]*gachaLinkJob
	// owned is each sender's last link; updated is when each UID was last
	// updated from a link, for StarRail-plugin's one-hour interval.
	owned   map[Subject]ownedLink
	updated map[string]time.Time
}

func (j *gachaLinkJobs) put(ref string, job *gachaLinkJob) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.jobs == nil {
		j.jobs = map[string]*gachaLinkJob{}
	}
	j.jobs[ref] = job
}

func (j *gachaLinkJobs) take(ref string) *gachaLinkJob {
	j.mu.Lock()
	defer j.mu.Unlock()
	job := j.jobs[ref]
	delete(j.jobs, ref)
	return job
}

func (j *gachaLinkJobs) own(owner Subject, link gachaLink, uid string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.owned == nil {
		j.owned = map[Subject]ownedLink{}
	}
	for subject, owned := range j.owned {
		if time.Now().After(owned.expires) {
			delete(j.owned, subject)
		}
	}
	j.owned[owner] = ownedLink{link: link, uid: uid, expires: time.Now().Add(24 * time.Hour)}
}

func (j *gachaLinkJobs) owner(owner Subject) (ownedLink, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	owned, ok := j.owned[owner]
	return owned, ok && time.Now().Before(owned.expires)
}

// markUpdated records an update of uid and returns the previous one, zero
// when it was more than an hour ago.
func (j *gachaLinkJobs) markUpdated(uid string) time.Time {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.updated == nil {
		j.updated = map[string]time.Time{}
	}
	last := j.updated[uid]
	if time.Since(last) < time.Hour {
		return last
	}
	j.updated[uid] = time.Now()
	return time.Time{}
}

// parseGachaLink reads a warp history link as Yunzai's dealUrl does: 〈= is a
// mangled &lang=, only the query after getGachaLog? or index.html? counts
// and the page fragment (#/log) is cut off. Links without hkrpg belong
// to the other games, and customer service links to asset records; ok is
// false when the text holds no Star Rail warp link.
func parseGachaLink(text string) (link gachaLink, ok bool, err error) {
	if !strings.Contains(text, "authkey=") || !strings.Contains(text, "hkrpg") {
		return gachaLink{}, false, nil
	}
	for _, service := range []string{"user-game-search", "bill-record-user", "customer-claim", "player-log", "user.mihoyo.com", "csc-service"} {
		if strings.Contains(text, service) {
			return gachaLink{}, false, nil
		}
	}
	text = strings.TrimSpace(strings.ReplaceAll(text, "〈=", "&"))
	for _, marker := range []string{"getGachaLog?", "getLdGachaLog?", "index.html?"} {
		if _, after, found := strings.Cut(text, marker); found {
			text = after
		}
	}
	if _, after, found := strings.Cut(text, "?"); found && !strings.HasPrefix(text, "authkey") {
		text = after
	}
	text, _, _ = strings.Cut(text, "#")
	params, _ := url.ParseQuery(text)
	key := params.Get("authkey")
	if len(key) < 8 || len(key) > 16384 || strings.ContainsAny(key, " \r\n\t") {
		return gachaLink{}, true, gameError("gacha_link_invalid", "链接复制错误")
	}
	return gachaLink{key: key, region: params.Get("region"), biz: params.Get("game_biz")}, true, nil
}

// gachaLinkPage reads one page of a pool with the link's authkey; the
// collaboration pools have their own page. The official retcodes get
// Yunzai's replies.
func (a *App) gachaLinkPage(ctx context.Context, link gachaLink, pool, endID string, page, size int) (map[string]any, error) {
	endpoint := gachaLinkCN
	biz := "hkrpg_cn"
	if !slices.Contains([]string{"prod_gf_cn", "prod_qd_cn"}, link.region) {
		endpoint, biz = gachaLinkOS, "hkrpg_global"
	}
	if link.biz != "" {
		biz = link.biz
	}
	endpoint += "getGachaLog"
	if pool == "21" || pool == "22" {
		endpoint = strings.TrimSuffix(endpoint, "getGachaLog") + "getLdGachaLog"
	}
	query := url.Values{"authkey_ver": {"1"}, "lang": {"zh-cn"}, "authkey": {link.key}, "region": {link.region}, "game_biz": {biz}, "gacha_type": {pool}, "page": {strconv.Itoa(page)}, "size": {strconv.Itoa(size)}, "end_id": {endID}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	client := a.LinkHTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, gameError("gacha_link_unavailable", "获取数据错误")
	}
	defer response.Body.Close()
	var envelope struct {
		RetCode int            `json:"retcode"`
		Data    map[string]any `json:"data"`
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil || response.StatusCode != http.StatusOK || json.Unmarshal(body, &envelope) != nil {
		return nil, gameError("gacha_link_unavailable", "获取数据错误")
	}
	switch envelope.RetCode {
	case 0:
		return envelope.Data, nil
	case -101:
		return nil, gameError("gacha_link_expired", "该链接已失效，请重新进入游戏，重新复制链接")
	case -100:
		return nil, gameError("gacha_link_incomplete", "链接不完整，请长按全选复制全部内容（可能输入法复制限制），或者复制的不是历史记录页面链接")
	default:
		return nil, gameError("gacha_link_invalid", "链接复制错误")
	}
}

// checkGachaLink finds the link's region when it names none, as Yunzai tries
// the mainland server and then America, and takes the UID from the first
// record of any pool.
func (a *App) checkGachaLink(ctx context.Context, link *gachaLink, length int) (string, error) {
	regions := []string{link.region}
	if link.region == "" {
		regions = []string{"prod_gf_cn", "prod_official_usa"}
	}
	var failure error
	for _, region := range regions {
		link.region = region
		for _, pool := range gachaLinkPools {
			data, err := a.gachaLinkPage(ctx, *link, pool[0], "0", 1, 6)
			if err != nil {
				var public *rayleabot.ActionError
				if errors.As(err, &public) && public.Code == "plugin.game_gacha_link_incomplete" && length == 1000 {
					return "", gameError("gacha_link_incomplete", "输入法限制，链接复制不完整，请更换输入法复制完整链接")
				}
				failure = err
				break
			}
			if region := asText(data["region"]); region != "" {
				link.region = region
			}
			if list := asList(data["list"]); len(list) > 0 {
				return asText(asObject(list[0])["uid"]), nil
			}
		}
	}
	if failure != nil {
		if len(regions) > 1 {
			return "", gameError("gacha_link_expired", "链接复制错误或已失效")
		}
		return "", failure
	}
	return "", gameError("gacha_link_empty", "暂无数据，请等待记录后再查询")
}

// gachaLinkMessage answers a message holding a warp link; handled is false
// for any other message.
func (a *App) gachaLinkMessage(ctx context.Context, event *rayleabot.EventContext) (handled bool, err error) {
	text := event.Event.Message.PlainText
	link, ok, err := parseGachaLink(text)
	if !ok {
		return false, nil
	}
	group := event.Event.Target.Type == "group"
	if group {
		config, configErr := a.Groups.Config(groupScope(event))
		if configErr == nil && config.Enabled != nil && !*config.Enabled {
			return false, nil
		}
	}
	if err != nil {
		return true, event.SendText(friendlyError(err))
	}
	uid, err := a.checkGachaLink(ctx, &link, len([]rune(text)))
	if err != nil {
		return true, event.SendText(friendlyError(err))
	}
	owner := syncOwner(event)
	a.LinkJobs.own(owner, link, uid)
	notice(ctx, event, "链接发送成功，数据获取中……")
	a.useLinkUID(ctx, event, uid)
	err = a.fetchByLink(ctx, event, &gachaLinkJob{link: link, uid: uid, owner: owner, target: event.Event.Target})
	if group {
		notice(ctx, event, "已收到链接，请撤回")
	}
	return true, err
}

// updateByLink is StarRail-plugin's 更新跃迁记录 with the sender's last link:
// at most once an hour for a UID.
func (a *App) updateByLink(ctx context.Context, event *rayleabot.EventContext, owned ownedLink) error {
	if last := a.LinkJobs.markUpdated(owned.uid); !last.IsZero() {
		return event.SendText("[" + owned.uid + "]近期已经更新过数据了，上次更新时间：" + last.In(time.FixedZone("UTC+8", 8*3600)).Format("2006-01-02 15:04:05") + "，两次更新间隔请大于1小时")
	}
	notice(ctx, event, "正在获取["+owned.uid+"]的跃迁数据...")
	return a.fetchByLink(ctx, event, &gachaLinkJob{link: owned.link, uid: owned.uid, owner: syncOwner(event), target: event.Event.Target, update: true})
}

// fetchByLink merges a link's records within this event, or hands the rest
// to a scheduled task.
func (a *App) fetchByLink(ctx context.Context, event *rayleabot.EventContext, job *gachaLinkJob) error {
	job.before = gachaPoolCounts(a.archiveOrEmpty(job.uid, job.link.region))
	job.expires = time.Now().Add(15 * time.Minute)
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{Link: true}, job.uid, job.link.region, false)
	if err != nil {
		return event.SendText(a.gachaLinkFailure(job, err))
	}
	job.sync = info.Ref
	result, err := a.stepGachaLink(ctx, job, time.Now().Add(40*time.Second))
	switch {
	case err != nil:
		return event.SendText(a.gachaLinkFailure(job, err))
	case result == nil:
		ref := gachaLinkTask + rand.Text()
		a.LinkJobs.put(ref, job)
		if _, err := event.Actions().SchedulerCreate(ctx, rayleabot.SchedulerCreateRequest{TaskID: ref, Cron: "* * * * *", LogLabel: a.Game.Name + "抽卡链接记录", Payload: map[string]any{"kind": "gacha_link"}}); err != nil {
			a.LinkJobs.take(ref)
			return event.SendText("记录较多，本次未能全部获取，请稍后重试。")
		}
		return event.SendText("记录较多，将在后台继续获取，完成后在此回复。")
	}
	return a.sendGachaLinkResult(ctx, event, job, *result, a.sendHere(event))
}

// gachaLinkFailure words a failed fetch: StarRail-plugin's expiry reply for
// 更新跃迁记录, else the error itself.
func (a *App) gachaLinkFailure(job *gachaLinkJob, err error) string {
	if job.update {
		return "抽卡链接已过期，请重新获取并绑定\n输入【" + a.Game.Prefix + "抽卡帮助】以查看抽卡链接获取方法"
	}
	return friendlyError(syncError(err))
}

// stepGachaLink fetches pages until the sync completes or stop passes; the
// result is nil while pages remain.
func (a *App) stepGachaLink(ctx context.Context, job *gachaLinkJob, stop time.Time) (*gacha.ImportResult, error) {
	for {
		info, err := a.Syncs.Step(ctx, a.Gacha, job.sync, job.sequence, func(ctx context.Context, pool, endID string, page int) (gacha.RemotePage, error) {
			data, err := a.gachaLinkPage(ctx, job.link, pool, endID, page, 20)
			if err != nil {
				return gacha.RemotePage{}, err
			}
			return gacha.ParsePage(job.uid, job.link.region, pool, endID, data)
		})
		if err != nil {
			_ = a.Syncs.Cancel(job.sync)
			return nil, err
		}
		job.sequence = info.Sequence
		if info.State == "completed" {
			return info.Result, nil
		}
		if time.Now().After(stop) {
			return nil, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// runGachaLink continues a long history on its scheduled task and answers in
// the chat the link was sent to.
func (a *App) runGachaLink(ctx context.Context, event *rayleabot.EventContext) error {
	if event.Event.SourceProtocol != "scheduler" || event.Event.SourceAdapter != "scheduler.internal" {
		return event.Fail("plugin.game_source_invalid", "任务来源无效。")
	}
	ref := asText(event.Event.Payload["task_id"])
	job := a.LinkJobs.take(ref)
	if job == nil || time.Now().After(job.expires) {
		_, _ = event.Actions().SchedulerDelete(ctx, ref)
		return event.Result(map[string]any{"handled": true})
	}
	send := func(segments ...rayleabot.Segment) error {
		_, err := event.Actions().MessageSend(ctx, rayleabot.MessageSendRequest{SourceProtocol: job.owner.SourceProtocol, SourceAdapter: job.owner.SourceAdapter, TargetType: job.target.Type, TargetID: job.target.ID, Message: rayleabot.MessageOut{Segments: segments}})
		return err
	}
	result, err := a.stepGachaLink(ctx, job, time.Now().Add(40*time.Second))
	if err == nil && result == nil {
		a.LinkJobs.put(ref, job)
		return event.Result(map[string]any{"handled": true})
	}
	_, _ = event.Actions().SchedulerDelete(ctx, ref)
	if err != nil {
		_ = send(rayleabot.Text(a.gachaLinkFailure(job, err)))
	} else {
		_ = a.sendGachaLinkResult(ctx, event, job, *result, send)
	}
	return event.Result(map[string]any{"handled": true})
}

// sendHere sends into the chat of the event.
func (a *App) sendHere(event *rayleabot.EventContext) func(...rayleabot.Segment) error {
	return func(segments ...rayleabot.Segment) error {
		return event.Send(event.Event.Target.Type, event.Event.Target.ID, segments...)
	}
}

// sendGachaLinkResult answers a finished fetch: after a pasted link, Yunzai's
// report of each pool and its gacha-log page of the character event warp;
// after 更新跃迁记录, StarRail-plugin's list of analysis commands.
func (a *App) sendGachaLinkResult(ctx context.Context, event *rayleabot.EventContext, job *gachaLinkJob, result gacha.ImportResult, send func(...rayleabot.Segment) error) error {
	prefix := a.Game.Prefix
	if job.update {
		return send(rayleabot.Text("跃迁数据获取成功，你可以使用：\n" + prefix + "跃迁分析\n" + prefix + "角色分析\n" + prefix + "光锥分析\n" + prefix + "常驻分析\n查看具体的跃迁数据"))
	}
	archive, err := a.Gacha.Read(result.UID, result.Region)
	if err != nil {
		return send(rayleabot.Text(friendlyError(err)))
	}
	if err := send(rayleabot.Text(gachaLinkSummary(prefix, job.before, gachaPoolCounts(archive)))); err != nil {
		return err
	}
	view := GachaView(a.Game, archive)
	if a.gacha != nil {
		if drawn, ok := a.gacha(a.imageContext(ctx), GachaImage{UID: result.UID, Role: Role{Game: a.Game.ID, UID: result.UID, Region: result.Region}, Command: "gacha-log", Archive: archive}); ok {
			view.Image = &drawn
		}
	}
	if image := a.renderView(ctx, event, view); image != "" {
		return send(rayleabot.Image(image))
	}
	return send(rayleabot.Text(view.Text()))
}

// gachaLinkSummary is Yunzai's reply after a link: the records each pool
// gained, then the record commands to try.
func gachaLinkSummary(prefix string, before, after map[string]int) string {
	lines := []string{}
	for _, pool := range gachaLinkPools {
		if after[pool[0]] > 0 {
			lines = append(lines, "["+pool[1]+"]记录获取成功，更新"+strconv.Itoa(after[pool[0]]-before[pool[0]])+"条")
		}
	}
	lines = append(lines, "", "抽卡记录更新完成，您还可回复", "【"+prefix+"全部记录】统计全部抽卡数据", "【"+prefix+"光锥记录】统计星铁光锥池数据", "【"+prefix+"角色统计】按卡池统计数据")
	return strings.Join(lines, "\n")
}

// gachaLinkCommand is StarRail-plugin's 抽卡链接: the link is taken in private
// chat, and pasting it anywhere is also read.
func (a *App) gachaLinkCommand(event *rayleabot.EventContext) error {
	if event.Event.Target.Type == "group" {
		return event.SendText("请私聊绑定")
	}
	return event.SendText("请发送抽卡链接")
}

// gachaPoolCounts counts an archive's records by pool.
func gachaPoolCounts(archive gacha.Archive) map[string]int {
	counts := map[string]int{}
	for _, record := range archive.Records {
		counts[record.GachaType]++
	}
	return counts
}

// chatArchive is the archive a record command reads: as upstream, the log of
// the UID named or of the UID in use, bound with or without an account.
func (a *App) chatArchive(ctx context.Context, event *rayleabot.EventContext, uid string) (gacha.Archive, Role, error) {
	owner, err := a.panelOwner(ctx, event, uid)
	if err != nil {
		return gacha.Archive{}, Role{}, err
	}
	role := owner.Role
	if !owner.Owned {
		role = Role{Game: a.Game.ID, UID: owner.UID}
		if summaries, err := a.Gacha.List(); err == nil {
			for _, summary := range summaries {
				if summary.UID == owner.UID {
					role.Region = summary.Region
					break
				}
			}
		}
	}
	archive, err := a.Gacha.Read(role.UID, role.Region)
	if err != nil {
		return gacha.Archive{}, role, a.noGachaRecords()
	}
	return archive, role, nil
}

// noGachaRecords is Yunzai's reply for a UID without warp records.
func (a *App) noGachaRecords() error {
	return gameError("archive_missing", "暂无抽卡记录\n"+a.Game.Prefix+"记录帮助，查看配置说明")
}

func (a *App) archiveOrEmpty(uid, region string) gacha.Archive {
	archive, _ := a.Gacha.Read(uid, region)
	return archive
}

// useLinkUID makes the link's UID the sender's current UID, as upstream sets
// it after a link; without the account plugin the link still works.
func (a *App) useLinkUID(ctx context.Context, event *rayleabot.EventContext, uid string) {
	client := a.accountClient(event)
	if listed, err := client.List(ctx, 0); err == nil {
		_ = useUID(ctx, client, listed, a.Game.ID, uid)
	}
}

// syncError gives the sync engine's failures their public codes.
func syncError(err error) error {
	switch {
	case errors.Is(err, gacha.ErrSync):
		return gameError("sync_expired", "同步任务已取消或过期，请重新开始。")
	case errors.Is(err, gacha.ErrConflict):
		return gameError("sync_conflict", "档案已变更或记录冲突，请重新开始同步；现有档案已保留。")
	case errors.Is(err, gacha.ErrInvalid):
		return gameError("sync_invalid", "官方记录结构或分页异常，现有档案已保留。")
	}
	return err
}
