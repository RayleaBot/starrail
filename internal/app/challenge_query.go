package app

import (
	"context"
	"maps"
	"regexp"
	"slices"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// challengeCommands are the operations StarRail-plugin's challenge and rogue
// commands run; they read their words anywhere in the message.
var challengeCommands = []string{"starrail.abyss", "starrail.abyss_latest", "starrail.challenge", "starrail.story", "starrail.boss", "starrail.peak", "starrail.rogue"}

// detailedRecords are the challenge records StarRail-plugin asks with
// need_all unless the message says 简易.
var detailedRecords = []string{"starrail.challenge", "starrail.story", "starrail.boss", "starrail.peak"}

// wordEndedCommands are the queries whose upstream rules end with the word:
// StarRail-plugin's 体力 (Yunzai's too, with 树脂 and 查询体力), 月历 and its
// other words, and 货币战争, and miao's 练度统计. A message with anything
// after the word matches none of them, so it is left to other plugins.
var wordEndedCommands = []string{"note", "monthly", "currency_wars", "training"}

// firstNumber is how StarRail-plugin reads the UID (userUid, and hkrpg's
// card): the first number anywhere in the message, a UID or not.
var firstNumber = regexp.MustCompile(`[0-9]+`)

// operationInput reads a query command's words as its upstream rule reads the
// message, and whether the rule takes the message at all: StarRail-plugin's
// challenge and rogue rules as challengeInput reads them, rules that end with
// the word only the word itself, and its card and other rogue rules anything
// after the word, the message's first number being the UID. 跃迁摘要, which
// no upstream plugin has, reads its pool and UID.
func (a *App) operationInput(operation Operation, command, word string, args []string) (map[string]any, string, bool, error) {
	switch {
	case slices.Contains(challengeCommands, operation.Name):
		input, uid := challengeInput(operation, word, args)
		return input, uid, true, nil
	case slices.Contains(wordEndedCommands, command):
		return map[string]any{}, "", len(args) == 0, nil
	case operation.Input == "starrail_pool":
		input, uid, err := a.commandInput(operation, args, nil)
		return input, uid, true, err
	}
	return map[string]any{}, firstNumber.FindString(strings.Join(args, " ")), true, nil
}

// challengeInput reads a challenge or rogue command the way StarRail-plugin's
// queryChallenge and rogue do. Their rules take anything after the word, and
// they look for their words anywhere in the message: 上期 asks the last period
// and, for Anomaly Arbitration, 往期 the three recent runs, which upstream
// draws side by side even beside 上期; 简易 asks a challenge's simple record,
// without need_all. Other words are passed over, and the Simulated Universe
// otherwise asks for the current period, as upstream does. The UID is the
// first number in the message; unlike on the query page, a number is never a
// period.
func challengeInput(operation Operation, word string, args []string) (map[string]any, string) {
	text := strings.Join(append([]string{word}, args...), " ")
	input := map[string]any{}
	switch {
	case operation.Name == "starrail.peak" && strings.Contains(text, "往期"):
		input["schedule_type"] = 3
	case strings.Contains(text, "上期"):
		input["schedule_type"] = 2
	case operation.Name == "starrail.rogue":
		input["schedule_type"] = 1
	}
	if operation.Name != "starrail.rogue" && strings.Contains(text, "简易") {
		input["need_all"] = false
	}
	return input, firstNumber.FindString(text)
}

// queryRecord runs a query the way StarRail-plugin's queryChallenge asks for a
// challenge record: unless the message said 简易, the detailed record comes
// first and, when it fails, failed hears why before the simple record is
// asked, which is drawn the same way. Other queries run as they are.
func queryRecord(ctx context.Context, client AccountsClient, choice Selection, operation string, input map[string]any, failed func(error)) (QueryResult, error) {
	result, err := client.Execute(ctx, choice, operation, input)
	if err == nil || !slices.Contains(detailedRecords, operation) || input["need_all"] == false {
		return result, err
	}
	failed(err)
	simple := map[string]any{}
	maps.Copy(simple, input)
	simple["need_all"] = false
	return client.Execute(ctx, choice, operation, simple)
}

// query runs a chat command's query. Before a challenge record falls back to
// the simple one, the reply says why the detailed one failed, as upstream's
// checkCode replies it.
func (a *App) query(ctx context.Context, event *rayleabot.EventContext, choice Selection, operation string, input map[string]any) (QueryResult, error) {
	return queryRecord(ctx, a.accountClient(event), choice, operation, input, func(err error) { notice(ctx, event, friendlyError(err)) })
}
