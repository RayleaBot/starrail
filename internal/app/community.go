package app

import (
	"context"
	"fmt"
	"strconv"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func (a *App) communitySign(ctx context.Context, client AccountsClient, input map[string]any) (map[string]any, error) {
	if input["confirm"] != true {
		return nil, gameError("input_invalid", "请明确确认本次社区签到。")
	}
	result, err := client.ExecuteConfirmed(ctx, Selection{AccountRef: asText(input["account_ref"])}, a.Game.ID+".community_sign", nil)
	if err != nil {
		return nil, err
	}
	view := View{Title: a.Game.Name + "米游社签到", Note: "米游币与米游币任务由米游社账号插件统计。"}
	if result.Data["accepted"] == true {
		view.Rows = append(view.Rows, Row{Label: "操作结果", Value: "官方已接受本次社区签到"})
	}
	return map[string]any{"view": view}, nil
}

func (a *App) communityCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	index := 1
	if len(args) > 1 {
		return event.SendText("请提供可选账号序号，例如 1。")
	}
	if len(args) == 1 {
		var err error
		index, err = strconv.Atoi(args[0])
		if err != nil || index < 1 {
			return event.SendText("账号序号应为正整数。")
		}
	}
	client := a.accountClient(event)
	list, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if index > len(list.Items) {
		return event.SendText("没有对应账号，请先扫码登录或查看账号列表。")
	}
	account := list.Items[index-1]
	result, err := a.communitySign(ctx, client, map[string]any{"account_ref": account.Ref, "confirm": true})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	view := result["view"].(View)
	view.Subtitle = fmt.Sprintf("账号 %d · %s", index, account.AccountLabel)
	return a.sendView(ctx, event, view)
}
