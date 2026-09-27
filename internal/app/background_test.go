package app

import (
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// Chat commands whose work can take longer than an event move their event to
// the background before the work and answer there; when the host refuses, as
// the plugin holds its limit of background events, they say so at once.
func TestLongCommandsAnswerInTheBackground(t *testing.T) {
	noAccounts := func(request rayleabot.ServiceCallRequest, _ bool) (map[string]any, string) {
		if request.Method != "list" {
			t.Errorf("unexpected %s", request.Method)
		}
		return map[string]any{"items": []any{}, "default_roles": map[string]any{}, "uid_bindings": map[string]any{}}, ""
	}
	for _, command := range []struct {
		text, word string
		replies    []string
	}{
		{"*推送公告", "推送公告", []string{}},
		{"*刷新排名", "刷新排名", []string{"面板数据刷新中，等待时间可能较长，请耐心等待...", "本群排名已刷新，共刷新0个UID数据..."}},
		{"*星琼任务", "星琼任务", []string{"开始任务：保存星琼数据，完成前请勿重复执行", "开拓月历ck：0个\n预计需要：\n完成时间：", "星琼任务完成"}},
	} {
		t.Run(command.word, func(t *testing.T) {
			host := newSDKHost(t, pluginApp(t), noAccounts)
			end, actions := host.groupMessage(command.text, command.word)
			if end["type"] != "result" || len(actions) == 0 || actions[0].Name != "event.detach" {
				t.Fatalf("ended with %v after %v", end, actions)
			}
			texts := sentTexts(t, host, "group", "g")
			if len(texts) != len(command.replies) {
				t.Fatalf("answered %q", texts)
			}
			for i, text := range texts {
				if !strings.HasPrefix(text, command.replies[i]) {
					t.Fatalf("answered %q", texts)
				}
			}
			host.busy = true
			if end, _ := host.groupMessage(command.text, command.word); terminalText(end) != "后台处理的任务已达上限，请稍后再试。" || len(host.sent) != len(texts) {
				t.Fatalf("refused, ended with %v and sent %v", end, host.sent)
			}
		})
	}
}
