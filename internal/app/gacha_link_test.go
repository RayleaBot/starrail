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
	var jobs gachaLinkJobs
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
	end, _ := host.groupMessage("https://webstatic.mihoyo.com/hkrpg/event/e20211215gacha-v2/index.html?authkey_ver=1&region=prod_gf_cn&authkey=abcdefghij&game_biz=hkrpg_cn#/log", "")
	if end["type"] != "result" {
		t.Fatalf("the link event ended with %v", end)
	}
	texts := []string{}
	for _, sent := range host.sent {
		if sent.TargetType != "group" || sent.TargetID != "g" {
			t.Fatalf("answered in %+v", sent)
		}
		texts = append(texts, sentText(sent.Message))
	}
	if len(texts) != 4 || texts[0] != "链接发送成功，数据获取中……" || !strings.HasPrefix(texts[1], "[角色]记录获取成功，更新21条") || !strings.Contains(texts[2], "100000001") || texts[3] != "已收到链接，请撤回" {
		t.Fatalf("answered %q", texts)
	}
}
