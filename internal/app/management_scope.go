package app

import (
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// The action decides whether an account is required, even when a caller omits
// page_id. Public UID archives, reference data and plugin settings are global.
func accountManagementAction(action string) bool {
	switch action {
	case "query", "panel.score", "growth.prepare", "growth.compute", "build.prepare", "build.compare", "redeem.run", "cloud.ocr.compare", "challenge.list", "challenge.clear":
		return true
	case "billing.schema":
		return false
	}
	for _, prefix := range []string{"accounts.", "groups.", "content.subscription.", "reminder.", "challenge.reminder.", "signin.", "monthly.", "community.", "billing.", "panel.history.", "gacha.sync.", "gacha.task.", "cloud.archive."} {
		if strings.HasPrefix(action, prefix) {
			return true
		}
	}
	return false
}

func managementAccount(event *rayleabot.EventContext) (rayleabot.BotAccount, error) {
	bot, ok := event.Event.ManagementBot()
	if !ok {
		return bot, &rayleabot.ActionError{Code: "platform.invalid_request", Message: "请先选择机器人账号。"}
	}
	return bot, nil
}

func accountMismatch() error {
	return &rayleabot.ActionError{Code: "platform.invalid_request", Message: "记录不属于当前机器人账号。"}
}

func belongsToBot(owner Subject, bot rayleabot.BotAccount) bool {
	return owner.SourceProtocol == bot.SourceProtocol && owner.BotID == bot.BotID
}

func visibleOwner(event *rayleabot.EventContext, owner Subject) bool {
	if event.Event.EventType != "management.action" {
		return owner == chatOwner(event)
	}
	bot, ok := event.Event.ManagementBot()
	return ok && belongsToBot(owner, bot)
}

func checkManagementInput(bot rayleabot.BotAccount, input map[string]any) error {
	if protocol, exists := input["source_protocol"]; exists && protocol != bot.SourceProtocol {
		return accountMismatch()
	}
	if id, exists := input["bot_id"]; exists && id != bot.BotID {
		return accountMismatch()
	}
	for _, key := range []string{"owner", "subject", "scope", "bot"} {
		if raw, exists := input[key]; exists {
			var fields map[string]any
			if decodeObject(raw, &fields) != nil || fields == nil {
				return accountMismatch()
			}
			if err := checkManagementInput(bot, fields); err != nil {
				return err
			}
		}
	}
	return nil
}

func managementGroupInput(bot rayleabot.BotAccount, input map[string]any) (map[string]any, error) {
	var scope GroupScope
	if raw, exists := input["scope"]; exists {
		if err := decodeObject(raw, &scope); err != nil {
			return nil, gameError("input_invalid", "群身份无效。")
		}
	} else {
		scope.GroupID = asText(input["group_id"])
	}
	scope.Protocol, scope.BotID = bot.SourceProtocol, bot.BotID
	if input == nil {
		input = map[string]any{}
	}
	input["scope"] = scope
	return input, nil
}
