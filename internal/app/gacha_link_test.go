package app

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestGachaLinkParsesStarRailLinksOnly(t *testing.T) {
	link, ok, err := parseGachaLink("https://webstatic.mihoyo.com/hkrpg/event/e20211215gacha-v2/index.html?authkey_ver=1&region=prod_gf_cn&authkey=abcdefgh%2Bij&game_biz=hkrpg_cn#/log")
	if !ok || err != nil || link.key != "abcdefgh+ij" || link.region != "prod_gf_cn" || link.biz != "hkrpg_cn" {
		t.Fatal(link, ok, err)
	}
	for _, text := range []string{
		"https://webstatic.mihoyo.com/hk4e/event/e20190909gacha-v3/index.html?authkey=abcdefghij",
		"https://webstatic.mihoyo.com/nap/event/e20230424gacha/index.html?authkey=abcdefghij&game_biz=nap_cn",
		"https://webstatic.mihoyo.com/csc-service-center-fe/index.html?authkey=abcdefghij&game_biz=hkrpg_cn",
		"跃迁记录",
	} {
		if _, ok, _ := parseGachaLink(text); ok {
			t.Fatal("claimed:", text)
		}
	}
}

// warpHistory serves the official warp history for UID 100000001: records
// character event warps (21 when zero) and one collaboration warp on its own
// page, answering an authkey other than abcdefghij with -101. deadlines are
// the deadlines of the requests.
type warpHistory struct {
	records   int
	deadlines []time.Time
}

func (w *warpHistory) RoundTrip(request *http.Request) (*http.Response, error) {
	deadline, _ := request.Context().Deadline()
	w.deadlines = append(w.deadlines, deadline)
	query := request.URL.Query()
	pool := query.Get("gacha_type")
	body := map[string]any{"retcode": 0, "message": ""}
	collaboration := strings.HasSuffix(request.URL.Path, "/getLdGachaLog")
	if query.Get("authkey") != "abcdefghij" || collaboration != (pool == "21" || pool == "22") {
		body["retcode"] = -101
	} else {
		records := []any{}
		count, base := map[string]int{"11": cmp.Or(w.records, 21), "21": 1}[pool], map[string]int{"11": 5000, "21": 4000}[pool]
		first := 1
		if end, _ := strconv.Atoi(query.Get("end_id")); end != 0 {
			first = base - end + 1
		}
		size, _ := strconv.Atoi(query.Get("size"))
		for n := first; n <= count && len(records) < size; n++ {
			at := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC).Add(-time.Duration(n) * time.Minute)
			records = append(records, map[string]any{"uid": "100000001", "gacha_id": "2003", "gacha_type": pool, "item_id": "1308", "count": "1", "time": at.Format(time.DateTime), "name": "黄泉", "item_type": "角色", "rank_type": "5", "id": strconv.Itoa(base - n)})
		}
		body["data"] = map[string]any{"list": records, "region": "prod_gf_cn", "region_time_zone": 8}
	}
	raw, _ := json.Marshal(body)
	recorder := httptest.NewRecorder()
	_, _ = recorder.Write(raw)
	return recorder.Result(), nil
}

func TestGachaLinkFetchesTheWholeWarpHistory(t *testing.T) {
	a := pluginApp(t)
	a.LinkHTTP = &http.Client{Transport: &warpHistory{}}
	link := gachaLink{key: "abcdefghij"}
	uid, err := a.checkGachaLink(context.Background(), &link, 80)
	if err != nil || uid != "100000001" || link.region != "prod_gf_cn" {
		t.Fatal(uid, link.region, err)
	}
	result, err := a.readLink(context.Background(), link, uid)
	if err != nil || result == nil || result.Added != 22 {
		t.Fatal(result, err)
	}
	archive, _ := a.Gacha.Read(uid, link.region)
	summary := gachaLinkSummary("*", map[string]int{}, gachaPoolCounts(archive))
	if !strings.HasPrefix(summary, "[角色]记录获取成功，更新21条\n[角色联动]记录获取成功，更新1条\n\n抽卡记录更新完成") {
		t.Fatal(summary)
	}
}

func TestGachaLinkUpdatesAUIDAtMostHourly(t *testing.T) {
	var jobs gachaLinks
	if !jobs.markUpdated("100000001").IsZero() {
		t.Fatal("first update refused")
	}
	if jobs.markUpdated("100000001").IsZero() {
		t.Fatal("second update within the hour allowed")
	}
	if !jobs.markUpdated("100000002").IsZero() {
		t.Fatal("another UID refused")
	}
	jobs.forgetUpdate("100000001")
	if !jobs.markUpdated("100000001").IsZero() {
		t.Fatal("an update that did not happen still counts")
	}
	owner := Subject{SourceProtocol: "onebot11", BotID: "bot", ActorID: "user"}
	jobs.own(owner, gachaLink{key: "abcdefghij", region: "prod_gf_cn"}, "100000001")
	if owned, ok := jobs.owner(owner); !ok || owned.uid != "100000001" {
		t.Fatal("pasted link not kept for 更新跃迁记录")
	}
	if _, ok := jobs.owner(Subject{ActorID: "other"}); ok {
		t.Fatal("another sender got the link")
	}
}

// warpLink is a link to warpHistory.
const warpLink = "https://webstatic.mihoyo.com/hkrpg/event/e20211215gacha-v2/index.html?authkey_ver=1&region=prod_gf_cn&authkey=abcdefghij&game_biz=hkrpg_cn#/log"

// linkHost is the plugin reading history on the SDK host without the account
// plugin, its pages paced by a clock that does not wait.
func linkHost(t *testing.T, history *warpHistory) (*App, *sdkHost) {
	t.Helper()
	a := pluginApp(t)
	a.clock = &fakeClock{at: time.Now()}
	a.LinkHTTP = &http.Client{Transport: history}
	return a, newSDKHost(t, a, func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
		return nil, "plugin.service_unavailable"
	})
}

// sentTexts are the texts of the messages sent into target, which a detached
// event sends all its replies as.
func sentTexts(t *testing.T, h *sdkHost, targetType, targetID string) []string {
	t.Helper()
	texts := []string{}
	for _, sent := range h.sent {
		if sent.TargetType != targetType || sent.TargetID != targetID {
			t.Fatalf("answered in %+v", sent)
		}
		texts = append(texts, sentText(sent.Message))
	}
	return texts
}

// A link with many pages is read in its own event moved to the background:
// every page under the background deadline, then Yunzai's answers in order
// in the chat, and in a group the request to recall the link, mentioning the
// sender.
func TestGachaLinkWithManyPagesReadsInOneBackgroundEvent(t *testing.T) {
	history := &warpHistory{records: 1000}
	a, host := linkHost(t, history)
	sent := time.Now()
	end, _ := host.groupMessage(warpLink, "")
	if end["type"] != "result" || len(host.detached) != 1 || len(host.created) != 0 {
		t.Fatalf("the link ended with %v after %d detaches and jobs %v", end, len(host.detached), host.created)
	}
	// Fifty pages and more of the character event warp alone.
	if len(history.deadlines) < 50 {
		t.Fatalf("read %d pages", len(history.deadlines))
	}
	for _, deadline := range history.deadlines {
		if deadline.Before(sent.Add(eventTimeout)) {
			t.Fatalf("a page was read under the event deadline %v", deadline)
		}
	}
	texts := sentTexts(t, host, "group", "g")
	if len(texts) != 4 || texts[0] != "链接发送成功，数据获取中……" || !strings.HasPrefix(texts[1], "[角色]记录获取成功，更新1000条\n[角色联动]记录获取成功，更新1条") || !strings.Contains(texts[2], "100000001") || texts[3] != "\n已收到链接，请撤回" {
		t.Fatalf("answered %q", texts)
	}
	if mention := host.sent[3].Message.Segments[0]; mention.Type != "at" || asText(mention.Data["user_id"]) != "u" {
		t.Fatalf("asked to recall with %+v", host.sent[3].Message)
	}
	if archive, err := a.Gacha.Read("100000001", "prod_gf_cn"); err != nil || len(archive.Records) != 1001 {
		t.Fatalf("kept %d records, %v", len(archive.Records), err)
	}
}

// 更新跃迁记录 reads the sender's last link again in a background event with
// StarRail-plugin's replies; a second update within the hour is refused, and
// an update whose link no longer reads answers the expiry and does not count.
func TestGachaLinkUpdateReadsTheLinkAgain(t *testing.T) {
	history := &warpHistory{}
	_, host := linkHost(t, history)
	host.message(warpLink, "")
	host.sent = nil
	end, _ := host.message("*更新跃迁记录", "更新跃迁记录")
	texts := sentTexts(t, host, "private", "u")
	if end["type"] != "result" || len(host.detached) != 2 || len(texts) != 2 || texts[0] != "正在获取[100000001]的跃迁数据..." || texts[1] != "跃迁数据获取成功，你可以使用：\n*跃迁分析\n*角色分析\n*光锥分析\n*常驻分析\n查看具体的跃迁数据" {
		t.Fatalf("the update ended with %v, answered %q", end, texts)
	}
	end, _ = host.message("*更新抽卡记录", "更新抽卡记录")
	if !strings.HasPrefix(terminalText(end), "[100000001]近期已经更新过数据了，上次更新时间：") || len(host.detached) != 2 {
		t.Fatalf("the update within the hour ended with %v", end)
	}
	a2, host2 := linkHost(t, history)
	host2.message(warpLink, "")
	user := Subject{SourceProtocol: "onebot11", BotID: "bot", ActorID: "u"}
	owned, _ := a2.Links.owner(user)
	owned.link.key = "expired1"
	a2.Links.own(user, owned.link, owned.uid)
	host2.sent = nil
	host2.message("*更新跃迁记录", "更新跃迁记录")
	if texts := sentTexts(t, host2, "private", "u"); len(texts) != 2 || texts[1] != "抽卡链接已过期，请重新获取并绑定\n输入【*抽卡帮助】以查看抽卡链接获取方法" {
		t.Fatalf("the expired update answered %q", texts)
	}
	if !a2.Links.markUpdated(owned.uid).IsZero() {
		t.Fatal("the failed update still counts")
	}
}

// A link or update the host would not move to the background, as the plugin
// holds its limit of background events, says so in the foreground without
// reading, and the refused update does not count against the hour.
func TestGachaLinkRefusedByTheHostRepliesClearly(t *testing.T) {
	history := &warpHistory{}
	a, host := linkHost(t, history)
	host.busy = true
	end, _ := host.groupMessage(warpLink, "")
	if end["action"] != "message.send" || terminalText(end) != "后台处理的任务已达上限，请稍后再试。" || len(history.deadlines) != 0 || len(host.sent) != 0 {
		t.Fatalf("the refused link ended with %v after %d reads, sent %v", end, len(history.deadlines), host.sent)
	}
	host.busy = false
	host.message(warpLink, "")
	host.busy = true
	if end, _ := host.message("*更新跃迁记录", "更新跃迁记录"); terminalText(end) != "后台处理的任务已达上限，请稍后再试。" {
		t.Fatalf("the refused update ended with %v", end)
	}
	if !a.Links.markUpdated("100000001").IsZero() {
		t.Fatal("the refused update counts against the hour")
	}
}
