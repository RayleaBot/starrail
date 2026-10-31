package app

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

var chineseLivestreamCodes = []string{"永远不会抛弃你", "永远不会欺骗你", "星神阿哈祝福你"}

func publicCodesFixture(t *testing.T, codes []string) PublicContentClient {
	t.Helper()
	return PublicContentClient{HTTP: cloudDoer(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("public code query sent credentials")
		}
		var data any
		switch r.URL.Path {
		case "/apihub/api/home/new":
			data = map[string]any{"navigator": []any{map[string]any{"app_path": "https://webstatic.mihoyo.com/bbs/event/live/index.html?act_id=ABC123"}}}
		case "/event/miyolive/index":
			data = map[string]any{"live": map[string]any{"code_ver": "fixture-version", "title": "前瞻兑换码"}}
		case "/event/miyolive/refreshCode":
			items := make([]any, 0, len(codes))
			for _, code := range codes {
				items = append(items, map[string]any{"code": code, "title": "<p>星琼×100</p>", "to_get_time": "1793359800"})
			}
			data = map[string]any{"code_list": items}
		default:
			t.Fatalf("unexpected public endpoint: %s", r.URL.Path)
		}
		raw, err := json.Marshal(map[string]any{"retcode": 0, "data": data})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}
}

func TestPublicCodesRetainChineseAndMixedCodes(t *testing.T) {
	want := append(slices.Clone(chineseLivestreamCodes), "TEST1234", "开拓者2026-TEST", strings.Repeat("星", 64))
	input := append(slices.Clone(want), chineseLivestreamCodes[0], "", "<script>", "ABCD\nEFGH", "兑换\u200b口令", strings.Repeat("星", 65))
	result, err := publicCodesFixture(t, input).codes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	rows := result["items"].([]map[string]any)
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, asText(row["code"]))
		if row["reward"] != "星琼×100" || row["expires_at"] != "2026-10-31 23:59:59 UTC+8" || row["expiry_estimated"] != true {
			t.Fatalf("code metadata changed: %+v", row)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("returned codes = %q, want %q", got, want)
	}
}

func TestCodesCommandRepliesWithCopyableChineseCodes(t *testing.T) {
	for _, target := range []string{"private", "group"} {
		t.Run(target, func(t *testing.T) {
			a := pluginApp(t)
			a.Content = publicCodesFixture(t, chineseLivestreamCodes)
			host := newSDKHost(t, a, func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
				t.Fatal("public code query requested an account")
				return nil, ""
			})
			frame, actions := host.chat(map[string]any{"type": target, "id": "fixture"}, "*兑换码", "兑换码")
			var reply rayleabot.MessageOut
			if err := decodeObject(asObject(frame["data"])["message"], &reply); err != nil {
				t.Fatal(err)
			}
			if frame["action"] != "message.send" || len(reply.Segments) != 1 || reply.Segments[0].Type != "text" {
				t.Fatalf("expected one copyable text reply: %+v", frame)
			}
			text := terminalText(frame)
			for _, code := range chineseLivestreamCodes {
				if strings.Count(text, code) != 1 {
					t.Fatalf("missing or duplicated code %q in %q", code, text)
				}
			}
			for _, action := range actions {
				if action.Name == "render.image" {
					t.Fatal("code query requested an image")
				}
			}
		})
	}
}
