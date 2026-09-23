package app

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// challengeCommands are the operations StarRail-plugin's challenge and rogue
// commands run; they read their words anywhere in the message.
var challengeCommands = []string{"starrail.abyss", "starrail.abyss_latest", "starrail.challenge", "starrail.story", "starrail.boss", "starrail.peak", "starrail.rogue"}

// detailedRecords are the challenge records StarRail-plugin asks with
// need_all unless the message says 简易.
var detailedRecords = []string{"starrail.challenge", "starrail.story", "starrail.boss", "starrail.peak"}

// challengeInput reads a challenge or rogue command the way StarRail-plugin's
// queryChallenge and rogue do. They look for their words anywhere in the
// message, so a word may come before or after the command's name or stand
// apart from it: 上期 asks the last period and, for Anomaly Arbitration, 往期
// the three recent runs, which upstream draws side by side even beside 上期;
// 简易 asks a challenge's simple record, without need_all. Otherwise the
// Simulated Universe asks for the current period, as upstream does. A word
// standing apart must be one the command's trigger takes after its name; a
// period may also be given as its number, as on the query page, and the UID
// follows.
func challengeInput(operation Operation, word string, args []string) (map[string]any, string, error) {
	words := []string{"本期", "上期", "简易"}
	switch operation.Name {
	case "starrail.peak":
		words = append(words, "往期")
	case "starrail.rogue":
		words = words[:2]
	}
	text, rest := word, []string{}
	for _, arg := range args {
		if slices.Contains(words, arg) {
			text += arg
		} else {
			rest = append(rest, arg)
		}
	}
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
	// Without a period number, as in "深渊100000001", the UID comes first.
	if len(rest) > 0 && operation.Input != "none" && !uidPattern.MatchString(rest[0]) {
		maximum := 2
		if operation.Input != "period" {
			maximum = 3
		}
		period, err := strconv.Atoi(rest[0])
		if err != nil || period < 1 || period > maximum {
			return nil, "", gameError("input_invalid", fmt.Sprintf("期数使用 1 至 %d。", maximum))
		}
		input["schedule_type"] = period
		rest = rest[1:]
	}
	uid := ""
	if len(rest) > 0 {
		uid = rest[0]
	}
	return input, uid, nil
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
