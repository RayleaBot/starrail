package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestChallengeInputFollowsQueryChallenge(t *testing.T) {
	var game Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &game); err != nil {
		t.Fatal(err)
	}
	a := &App{Game: game}
	for _, tc := range []struct {
		operation, word string
		args            []string
		// period is the schedule_type asked, 0 for none; simple leaves
		// need_all out.
		period int
		simple bool
		uid    string
	}{
		{"starrail.challenge", "忘却之庭", nil, 0, false, ""},
		// 上期 and 简易 count anywhere in the message: before or after the
		// name, or apart from it, and around the UID.
		{"starrail.challenge", "上期忘却之庭", nil, 2, false, ""},
		{"starrail.challenge", "混沌上期", nil, 2, false, ""},
		{"starrail.story", "简易虚构叙事", nil, 0, true, ""},
		{"starrail.boss", "末日幻影简易上期100000001", []string{"100000001"}, 2, true, "100000001"},
		{"starrail.boss", "本期末日", nil, 0, false, ""},
		{"starrail.abyss", "深渊", []string{"上期"}, 2, false, ""},
		{"starrail.abyss", "深渊", []string{"简易", "100000001"}, 0, true, "100000001"},
		{"starrail.abyss", "深渊", []string{"100000001", "上期"}, 2, false, "100000001"},
		{"starrail.abyss", "上期深渊简易", nil, 2, true, ""},
		// The first number in the message is the UID, as upstream reads it,
		// never a period; other words are passed over, and the words count
		// wherever they stand.
		{"starrail.abyss", "深渊", []string{"2", "100000001"}, 0, false, "2"},
		{"starrail.abyss", "深渊", []string{"上期", "2"}, 2, false, "2"},
		{"starrail.abyss", "深渊", []string{"往期"}, 0, false, ""},
		{"starrail.abyss", "深渊", []string{"看看上期的", "a100000001"}, 2, false, "100000001"},
		{"starrail.abyss", "深渊100000001", []string{"100000001", "简易"}, 0, true, "100000001"},
		// 最新深渊 runs queryChallenge too.
		{"starrail.abyss_latest", "最新深渊上期", nil, 2, false, ""},
		{"starrail.abyss_latest", "最新简易深渊", []string{"100000001"}, 0, true, "100000001"},
		// Anomaly Arbitration's 往期 asks the three recent runs, even beside
		// 上期.
		{"starrail.peak", "异相仲裁往期", nil, 3, false, ""},
		{"starrail.peak", "上期异相仲裁往期", nil, 3, false, ""},
		{"starrail.peak", "仲裁上期", nil, 2, false, ""},
		{"starrail.peak", "简易异相", []string{"往期"}, 3, true, ""},
		// The rogue command asks for the current period unless 上期, and has
		// no simple record.
		{"starrail.rogue", "模拟宇宙", nil, 1, false, ""},
		{"starrail.rogue", "宇宙上期", nil, 2, false, ""},
		{"starrail.rogue", "宇宙", []string{"3", "100000001"}, 1, false, "3"},
		{"starrail.rogue", "宇宙", []string{"简易"}, 1, false, ""},
	} {
		operation, _ := a.operation(tc.operation)
		input, uid := challengeInput(operation, tc.word, tc.args)
		_, simple := input["need_all"]
		if input["schedule_type"] != nil && input["schedule_type"] != tc.period || input["schedule_type"] == nil && tc.period != 0 || simple != tc.simple || simple && input["need_all"] != false || uid != tc.uid {
			t.Errorf("%s %v = %v %q, want period %d simple %v uid %q", tc.word, tc.args, input, uid, tc.period, tc.simple, tc.uid)
		}
	}
}

// A query command takes what follows its word as its upstream rule does.
func TestOperationInputFollowsUpstreamRules(t *testing.T) {
	var game Game
	if err := json.Unmarshal(pluginFile(t, "internal/assets/game.json"), &game); err != nil {
		t.Fatal(err)
	}
	a := &App{Game: game}
	for _, tc := range []struct {
		command, word string
		args          []string
		uid           string
		ours          bool
	}{
		// StarRail-plugin's card and rogue rules take anything after the
		// word and read the first number as the UID.
		{"profile", "卡片", []string{"看看", "100000001"}, "100000001", true},
		{"divergent-week", "本期演算", []string{"上周", "12"}, "12", true},
		{"swarm", "蝗灾", []string{"简易"}, "", true},
		// Rules that end with the word take nothing after it.
		{"note", "体力", nil, "", true},
		{"note", "体力", []string{"100000001"}, "", false},
		{"monthly", "月历", []string{"202609"}, "", false},
		{"currency_wars", "货币战争", []string{"一"}, "", false},
		{"training", "练度统计", []string{"100000001"}, "", false},
		{"abyss", "深渊", []string{"往期"}, "", true},
	} {
		operation, ok := a.operation("starrail." + strings.ReplaceAll(tc.command, "-", "_"))
		if !ok {
			t.Fatalf("no operation for %s", tc.command)
		}
		_, uid, ours, err := a.operationInput(operation, tc.command, tc.word, tc.args)
		if err != nil || uid != tc.uid || ours != tc.ours {
			t.Errorf("%s %v = %q %v %v", tc.word, tc.args, uid, ours, err)
		}
	}
}

// recordCaller answers challenge records as the accounts plugin does, failing
// the detailed ones while detailed is set.
type recordCaller struct {
	detailed error
	simple   error
	asked    []string
}

func (c *recordCaller) CallService(_ context.Context, req rayleabot.ServiceCallRequest, out any) error {
	input, _ := req.Params["input"].(map[string]any)
	kind := "detailed"
	if input["need_all"] == false {
		kind = "simple"
	}
	c.asked = append(c.asked, asText(req.Params["operation"])+"/"+kind)
	failure := c.detailed
	if kind == "simple" {
		failure = c.simple
	}
	if failure != nil {
		return failure
	}
	return decodeObject(QueryResult{Data: map[string]any{"kind": kind, "schedule_type": input["schedule_type"]}}, out)
}

func TestChallengeRecordFallsBackToSimple(t *testing.T) {
	challenge := &rayleabot.ActionError{Code: "plugin.upstream_challenge_required", Message: "米游社要求完成验证，请在官方应用处理后重试。"}
	run := func(caller *recordCaller, operation string, input map[string]any) (QueryResult, []string, error) {
		failures := []string{}
		result, err := queryRecord(t.Context(), AccountsClient{Caller: caller, Game: "starrail"}, Selection{"account", "role"}, operation, input, func(err error) {
			failures = append(failures, PublicError(err).Code)
		})
		return result, failures, err
	}

	// Upstream asks the detailed record first and, when it fails, replies
	// why and draws the simple record of the same period.
	caller := &recordCaller{detailed: challenge}
	result, failures, err := run(caller, "starrail.boss", map[string]any{"schedule_type": 2})
	if err != nil || result.Data["kind"] != "simple" || asText(result.Data["schedule_type"]) != "2" || strings.Join(caller.asked, ",") != "starrail.boss/detailed,starrail.boss/simple" || strings.Join(failures, ",") != challenge.Code {
		t.Errorf("fallback = %v %v %v %v", result.Data, caller.asked, failures, err)
	}
	// A detailed record that answers is drawn as it is.
	caller = &recordCaller{}
	if result, failures, err = run(caller, "starrail.peak", map[string]any{}); err != nil || result.Data["kind"] != "detailed" || len(caller.asked) != 1 || len(failures) != 0 {
		t.Errorf("detailed = %v %v %v %v", result.Data, caller.asked, failures, err)
	}
	// 简易 asks the simple record alone.
	caller = &recordCaller{detailed: challenge}
	if result, failures, err = run(caller, "starrail.challenge", map[string]any{"need_all": false}); err != nil || result.Data["kind"] != "simple" || strings.Join(caller.asked, ",") != "starrail.challenge/simple" || len(failures) != 0 {
		t.Errorf("simple = %v %v %v %v", result.Data, caller.asked, failures, err)
	}
	// When the simple record fails too, both failures are told, as
	// upstream's checkCode replies each.
	caller = &recordCaller{detailed: challenge, simple: &rayleabot.ActionError{Code: "plugin.upstream_rate_limited", Message: "频繁"}}
	if _, failures, err = run(caller, "starrail.story", nil); PublicError(err).Code != "plugin.upstream_rate_limited" || len(caller.asked) != 2 || strings.Join(failures, ",") != challenge.Code {
		t.Errorf("both failed = %v %v %v", caller.asked, failures, err)
	}
	// Other records have no simple form.
	caller = &recordCaller{detailed: challenge}
	if _, failures, err = run(caller, "starrail.rogue", map[string]any{"schedule_type": 1}); err == nil || len(caller.asked) != 1 || len(failures) != 0 {
		t.Errorf("rogue = %v %v %v", caller.asked, failures, err)
	}
}
