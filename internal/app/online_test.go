package app

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 在线时长 follows the official list's record-ID cursor page by page: the
// power records of earlier days arrive only through next_end_id.
func TestOnlineTimeFollowsTheRecordCursor(t *testing.T) {
	var mu sync.Mutex
	cursors := []string{}
	role := map[string]any{"ref": "role", "game": "starrail", "uid": "100000001", "region": "prod_gf_cn", "nickname": "开拓者", "level": 70}
	records := func(day string, from int) []any {
		list := []any{}
		for i := range 20 {
			list = append(list, map[string]any{"id": fmt.Sprint(from - i), "time": fmt.Sprintf("%s 12:%02d:00", day, 59-i*2), "action": onlineAction, "add_num": "1"})
		}
		return list
	}
	a := pluginApp(t)
	// The reply lists the last seven days up to today, the newest records' day.
	a.clock = &fakeClock{at: time.Date(2026, 9, 28, 21, 0, 0, 0, chinaZone)}
	host := newSDKHost(t, a, func(request rayleabot.ServiceCallRequest, _ bool) (map[string]any, string) {
		switch request.Method {
		case "list":
			owner := map[string]any{"source_protocol": "onebot11", "source_adapter": "a", "bot_id": "bot", "actor_id": "u"}
			return map[string]any{"items": []any{map[string]any{"ref": "account", "owner": owner, "account_label": "账号", "roles": []any{role}, "status": "valid"}}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
		case "execute":
			input := asObject(request.Params["input"])
			mu.Lock()
			cursors = append(cursors, asText(input["end_id"]))
			mu.Unlock()
			data := map[string]any{"items": records("2026-09-28", 140), "has_more": true, "next_end_id": "121"}
			if asText(input["end_id"]) == "121" {
				data = map[string]any{"items": records("2026-09-27", 120), "has_more": false, "next_end_id": ""}
			}
			return map[string]any{"operation": "starrail.billing", "role": role, "data": data}, ""
		}
		return nil, "plugin.method_not_found"
	})
	end, _ := host.groupMessage("*在线时长", "在线时长")
	text := terminalText(end)
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(cursors, ",") != "0,121" {
		t.Fatalf("cursors %v", cursors)
	}
	if !strings.Contains(text, "2026-09-27") || !strings.Contains(text, "2026-09-28") {
		t.Fatalf("reply %q", text)
	}
}
