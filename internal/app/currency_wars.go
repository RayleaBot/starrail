package app

import (
	"context"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// currencyWarsNumerals are the numerals 货币战绩 may end with.
var currencyWarsNumerals = []string{"一", "二", "三", "四", "五", "六", "七", "八", "九", "十"}

// CurrencyWarsRecord is the record 货币战绩 names, as StarRail-plugin's
// grid_fight_archive reads the numeral that ends the word: 货币战绩三 is the
// third most recent, and a word without one the latest.
func CurrencyWarsRecord(word string) int {
	for index, numeral := range currencyWarsNumerals {
		if strings.HasSuffix(word, numeral) {
			return index + 1
		}
	}
	return 1
}

// currencyWarsArchive is StarRail-plugin's 货币战绩: the record the word names,
// drawn on its gridFightArchive page, or upstream's reply when there are
// fewer records.
func (a *App) currencyWarsArchive(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	if len(args) > 1 {
		return event.SendText("请提供至多一个 UID。")
	}
	uid := ""
	if len(args) == 1 {
		uid = args[0]
	}
	client := a.accountClient(event)
	listed, err := client.List(ctx, 0)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	choice, _, err := Choose(listed, a.Game.ID, uid)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	operation, _ := a.operation(a.Game.ID + ".currency_wars")
	result, err := client.Execute(ctx, choice, operation.Name, map[string]any{})
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	word := event.Event.Command()
	if index := CurrencyWarsRecord(word); len(asList(result.Data["grid_fight_archive_list"])) < index {
		return event.SendText("未找到货币战争第" + strconv.Itoa(index) + "条记录")
	}
	view := BusinessView(a.Game, operation, result, a.Catalog)
	view.Image = a.featureImage(ctx, client, choice, a.Game.ID+".currency_wars_archive", word, nil, result)
	return a.sendView(ctx, event, view)
}
