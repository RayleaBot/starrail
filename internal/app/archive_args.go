package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// ArchiveArgs follows StarRail-plugin's parseArchiveArgs: one or two digits
// select a record, a longer number is a UID; two numbers mean index and UID.
func ArchiveArgs(text string) (index int, uid string) {
	index = 1
	numbers := firstNumber.FindAllString(text, -1)
	if len(numbers) > 1 {
		index, _ = strconv.Atoi(numbers[0])
		uid = numbers[1]
	} else if len(numbers) == 1 {
		if len(numbers[0]) <= 2 {
			index, _ = strconv.Atoi(numbers[0])
		} else {
			uid = numbers[0]
		}
	}
	return max(1, index), uid
}

func (a *App) archiveCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	if len(args) > 1 || len(args) == 1 && (args[0] == "" || firstNumber.FindString(args[0]) != args[0]) {
		return event.Result(map[string]any{"handled": false})
	}
	text := strings.Join(append([]string{event.Event.Command()}, args...), " ")
	index, uid := ArchiveArgs(text)
	listed, err := a.accountClient(event).List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	name := map[string]string{"divergent-normal": "divergent_normal", "divergent-week": "divergent_week", "currency-wars-archive": "currency_wars"}[command]
	operation, _ := a.operation(a.Game.ID + "." + name)
	query, textOperation := a.routedQuery(operation, operation.Name)
	result, err := a.query(ctx, event, choice, query, map[string]any{})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	imageKey := operation.Name
	switch command {
	case "divergent-normal":
		count := len(asList(asObject(result.Data["normal_detail"])["records"]))
		if index > count {
			return event.SendText(fmt.Sprintf("未找到差分宇宙·常规演算第%d条记录：共%d条记录", index, count))
		}
	case "divergent-week":
		key, period := "cur_week_detail", "本期"
		if strings.Contains(text, "上期") || strings.Contains(text, "上周") {
			key, period = "last_week_detail", "上期"
		}
		count := len(asList(asObject(result.Data[key])["records"]))
		if index > count {
			return event.SendText(fmt.Sprintf("未找到差分宇宙·周期演算%s第%d条记录：%s共有%d条记录", period, index, period, count))
		}
	case "currency-wars-archive":
		if index > len(asList(result.Data["grid_fight_archive_list"])) {
			return event.SendText(fmt.Sprintf("未找到货币战争第%d条记录", index))
		}
		imageKey = a.Game.ID + ".currency_wars_archive"
	}
	view := BusinessView(a.Game, textOperation, result, a.Catalog)
	view.Image = a.featureImage(ctx, event, choice, imageKey, text, nil, result)
	return a.sendView(ctx, event, view)
}
