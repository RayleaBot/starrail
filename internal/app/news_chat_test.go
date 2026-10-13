package app

import (
	"context"
	"net/http"
	"testing"
)

func TestSearchLeavesSignInWords(t *testing.T) {
	a := pluginApp(t)
	a.Content.HTTP = doerFunc(func(*http.Request) (*http.Response, error) {
		t.Error("sign-in words made a content request")
		return nil, context.Canceled
	})
	host := newSDKHost(t, a, nil)
	end, _ := host.message("*米游社每日签到", "米游社每日签到")
	if asObject(end["data"])["handled"] != false || len(host.sent) != 0 {
		t.Fatal(end, host.sent)
	}
}
