package app

import (
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// StarRail-plugin's own panel service (apps/panel.js) answers beside miao's
// panels, the way Miao-Yunzai routes the words with MiaoOrStarrail's default
// settings (miao on, StarRail-plugin off): MiaoOrStarrail hands the words it
// takes to miao, and panel.js answers the ones it leaves. The service is
// one of the APIs its defSet/panelApi.yaml lists, which the panel_api
// setting picks.

// panelAPI is the API the panel_api setting picks, counted from 1 as
// upstream's default; a setting out of range picks the first.
func (a *App) panelAPI(event *rayleabot.EventContext) (int, string) {
	index := settings(event).PanelAPI
	if index < 1 || index > len(a.Game.PanelAPIs) {
		index = 1
	}
	return index, a.Game.PanelAPIs[index-1]
}

// apiHost is how upstream names an API in replies and pages: the host of its
// address.
func apiHost(api string) string {
	parts := strings.Split(api, "/")
	if len(parts) < 3 {
		return api
	}
	return parts[2]
}

// apiIndexWord is upstream's reading of 切换面板API: the first number in the
// message.
var apiIndexWord = regexp.MustCompile(`[1-9][0-9]*`)

// apiCommand answers 切换面板API and API列表, which upstream leaves to the
// bot's masters.
func (a *App) apiCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	master := slices.Contains(event.SuperAdmins, event.Event.Actor.ID)
	if command == "api-list" {
		if !master {
			return event.SendText("仅限主人可以查看API列表")
		}
		lines := []string{"API列表："}
		for i, api := range a.Game.PanelAPIs {
			lines = append(lines, strconv.Itoa(i+1)+"："+apiHost(api))
		}
		index, api := a.panelAPI(event)
		return event.SendText(strings.Join(lines, "\n") + "\n当前API：\n" + strconv.Itoa(index) + "：" + apiHost(api))
	}
	word := event.Event.Command()
	if len(args) == 0 && (word == "设置面板" || word == "切换面板") {
		// MiaoOrStarrail takes the bare word ending in 面板 first and hands it
		// to miao, which finds no character in it.
		return event.Result(map[string]any{"handled": false})
	}
	if !master {
		return event.SendText("仅限主人可以切换API")
	}
	index, err := strconv.Atoi(apiIndexWord.FindString(strings.Join(append([]string{word}, args...), " ")))
	if err != nil || index > len(a.Game.PanelAPIs) {
		return event.SendText("请输入正确的API序号")
	}
	if _, err = event.Actions().ConfigWrite(ctx, map[string]any{"panel_api": index}); err != nil {
		return event.SendText("切换API失败，请前往控制台查看报错！")
	}
	return event.SendText("切换API成功，当前API：" + apiHost(a.Game.PanelAPIs[index-1]))
}
