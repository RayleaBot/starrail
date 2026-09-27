package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
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

// warpHistory serves the official warp history for UID 100000001: 21
// character event warps, and one collaboration warp on its own page.
type warpHistory struct{}

func (warpHistory) RoundTrip(request *http.Request) (*http.Response, error) {
	query := request.URL.Query()
	pool := query.Get("gacha_type")
	body := map[string]any{"retcode": 0, "message": ""}
	collaboration := strings.HasSuffix(request.URL.Path, "/getLdGachaLog")
	if query.Get("authkey") != "abcdefghij" || collaboration != (pool == "21" || pool == "22") {
		body["retcode"] = -101
	} else {
		records := []any{}
		count, base := map[string]int{"11": 21, "21": 1}[pool], map[string]int{"11": 5000, "21": 4000}[pool]
		first := 1
		if end, _ := strconv.Atoi(query.Get("end_id")); end != 0 {
			first = base - end + 1
		}
		size, _ := strconv.Atoi(query.Get("size"))
		for n := first; n <= count && len(records) < size; n++ {
			records = append(records, map[string]any{"uid": "100000001", "gacha_id": "2003", "gacha_type": pool, "item_id": "1308", "count": "1", "time": fmt.Sprintf("2024-01-01 00:00:%02d", 59-n), "name": "黄泉", "item_type": "角色", "rank_type": "5", "id": strconv.Itoa(base - n)})
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
	a.LinkHTTP = &http.Client{Transport: warpHistory{}}
	link := gachaLink{key: "abcdefghij"}
	uid, err := a.checkGachaLink(context.Background(), &link, 80)
	if err != nil || uid != "100000001" || link.region != "prod_gf_cn" {
		t.Fatal(uid, link.region, err)
	}
	info, err := a.Syncs.Start(a.Gacha, gacha.SyncChoice{Link: true}, uid, link.region, false)
	if err != nil {
		t.Fatal(err)
	}
	job := &gachaLinkJob{link: link, uid: uid, sync: info.Ref, before: map[string]int{}}
	result, err := a.stepGachaLink(context.Background(), job, time.Now().Add(time.Minute))
	if err != nil || result == nil || result.Added != 22 {
		t.Fatal(result, err)
	}
	archive, _ := a.Gacha.Read(uid, link.region)
	summary := gachaLinkSummary("*", job.before, gachaPoolCounts(archive))
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
	owner := Subject{SourceProtocol: "onebot11", SourceAdapter: "adapter", BotID: "bot", ActorID: "user"}
	jobs.own(owner, gachaLink{key: "abcdefghij", region: "prod_gf_cn"}, "100000001")
	if owned, ok := jobs.owner(owner); !ok || owned.uid != "100000001" {
		t.Fatal("pasted link not kept for 更新跃迁记录")
	}
	if _, ok := jobs.owner(Subject{ActorID: "other"}); ok {
		t.Fatal("another sender got the link")
	}
}

// A link read within its event answers there as Yunzai does: the notice, the
// records each pool gained, the record page and, in a group, the request to
// recall the link.
func TestGachaLinkReadWithinItsEventAnswersInFull(t *testing.T) {
	a := pluginApp(t)
	a.LinkHTTP = &http.Client{Transport: warpHistory{}}
	host := newSDKHost(t, a, func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
		return nil, "plugin.service_unavailable"
	})
	end, _ := host.groupMessage(warpLink, "")
	texts := []string{}
	for _, sent := range host.sent {
		if sent.TargetType != "group" || sent.TargetID != "g" {
			t.Fatalf("answered in %+v", sent)
		}
		texts = append(texts, sentText(sent.Message))
	}
	if data := asObject(end["data"]); end["action"] != "message.send" || data["target_type"] != "group" || data["target_id"] != "g" {
		t.Fatalf("the link event ended with %v", end)
	}
	texts = append(texts, terminalText(end))
	if len(texts) != 4 || texts[0] != "链接发送成功，数据获取中……" || !strings.HasPrefix(texts[1], "[角色]记录获取成功，更新21条") || !strings.Contains(texts[2], "100000001") || texts[3] != "已收到链接，请撤回" {
		t.Fatalf("answered %q", texts)
	}
}

// roundTripFunc answers a client's requests with a function.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// warpLink is a link to warpHistory.
const warpLink = "https://webstatic.mihoyo.com/hkrpg/event/e20211215gacha-v2/index.html?authkey_ver=1&region=prod_gf_cn&authkey=abcdefghij&game_biz=hkrpg_cn#/log"

// slowLinkOnHost sends warpLink in a private chat through the SDK, each page
// of the history taking ten seconds and slow answering a page instead when
// it returns true, and checks that the event handed the rest to a job, whose
// ID it returns.
func slowLinkOnHost(t *testing.T, slow func(clock *fakeClock) bool) (*fakeClock, *sdkHost, string) {
	t.Helper()
	a := pluginApp(t)
	clock := &fakeClock{at: time.Unix(1_800_000_000, 0)}
	a.clock = clock
	a.LinkHTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if slow != nil && slow(clock) {
			return nil, context.DeadlineExceeded
		}
		clock.set(clock.Now().Add(10 * time.Second))
		return warpHistory{}.RoundTrip(r)
	})}
	host := newSDKHost(t, a, func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
		return nil, "plugin.service_unavailable"
	})
	end, _ := host.message(warpLink, "")
	if terminalText(end) != "记录较多，将在后台继续获取，完成后在此回复。" || len(host.sent) != 1 || sentText(host.sent[0].Message) != "链接发送成功，数据获取中……" {
		t.Fatalf("the link event ended with %v after %v", end, host.sent)
	}
	return clock, host, host.job(gachaLinkTask)
}

// triggerUntilDone runs the job each minute after start, as the host's
// scheduler does, until the plugin deletes it.
func triggerUntilDone(t *testing.T, clock *fakeClock, host *sdkHost, ref string, start time.Time, limit int) {
	t.Helper()
	for minute := 1; len(host.deleted) == 0; minute++ {
		if minute > limit {
			t.Fatalf("%d triggers of %s did not finish it", limit, ref)
		}
		clock.set(start.Add(time.Duration(minute) * time.Minute))
		if end, _ := host.trigger(ref); end["type"] != "result" {
			t.Fatalf("trigger %d ended with %v", minute, end)
		}
	}
}

// checkLinkAnswer checks that the job ref was removed and the records were
// answered after the first message in the private chat of the link.
func checkLinkAnswer(t *testing.T, host *sdkHost, ref string) {
	t.Helper()
	if len(host.deleted) != 1 || host.deleted[0] != ref || len(host.jobs) != 0 || len(host.sent) != 3 {
		t.Fatalf("deleted %v, jobs %v, answered %v", host.deleted, host.jobs, host.sent)
	}
	for _, sent := range host.sent {
		if sent.SourceProtocol != "onebot11" || sent.SourceAdapter != "a" || sent.TargetType != "private" || sent.TargetID != "u" {
			t.Fatalf("answered in %+v", sent)
		}
	}
	if !strings.HasPrefix(sentText(host.sent[1].Message), "[角色]记录获取成功，更新21条") || !strings.Contains(sentText(host.sent[2].Message), "100000001") {
		t.Fatalf("answered %v", host.sent)
	}
}

// A link whose history takes longer than its event, run through the SDK as
// the host runs it: the triggers of its job carry the job's payload but not
// its task ID, fetch the rest within their events and answer in the private
// chat the link came from.
func TestGachaLinkFinishesOnTheHostsTriggers(t *testing.T) {
	clock, host, ref := slowLinkOnHost(t, nil)
	triggerUntilDone(t, clock, host, ref, time.Unix(1_800_000_000, 0), 5)
	checkLinkAnswer(t, host, ref)
}

// A trigger whose page is still open when the trigger runs out of time (the
// page answers only once the host has timed the event out) ends its event
// in time and leaves the job to the next trigger, which reads that page
// again and finishes.
func TestGachaLinkRecoversFromATriggerThatRanOutOfTime(t *testing.T) {
	stuck := false
	clock, host, ref := slowLinkOnHost(t, func(clock *fakeClock) bool {
		if stuck || clock.Now().Before(time.Unix(1_800_000_000, 0).Add(time.Minute)) {
			return false
		}
		stuck = true
		clock.set(clock.Now().Add(time.Minute))
		return true
	})
	start := time.Unix(1_800_000_000, 0)
	clock.set(start.Add(time.Minute))
	if end, _ := host.trigger(ref); end["type"] != "result" || len(host.deleted) != 0 || len(host.sent) != 1 {
		t.Fatalf("the trigger that ran out ended with %v, deleted %v, answered %v", end, host.deleted, host.sent)
	}
	triggerUntilDone(t, clock, host, ref, start.Add(2*time.Minute), 5)
	checkLinkAnswer(t, host, ref)
}

// A link still unread after its fifteen minutes answers with the failure and
// removes its job.
func TestGachaLinkPastItsLifetimeAnswersItsFailure(t *testing.T) {
	clock, host, ref := slowLinkOnHost(t, nil)
	clock.set(time.Unix(1_800_000_000, 0).Add(16 * time.Minute))
	if end, _ := host.trigger(ref); end["type"] != "result" {
		t.Fatalf("the trigger ended with %v", end)
	}
	if len(host.deleted) != 1 || host.deleted[0] != ref || len(host.sent) != 2 || sentText(host.sent[1].Message) != "后台读取超时，未能完成。" {
		t.Fatalf("deleted %v, answered %v", host.deleted, host.sent)
	}
}
