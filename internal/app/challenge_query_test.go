package app

import (
	"encoding/json"
	"testing"
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
		{"starrail.abyss", "深渊", []string{"2", "100000001"}, 2, false, "100000001"},
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
		{"starrail.rogue", "宇宙", []string{"3", "100000001"}, 3, false, "100000001"},
	} {
		operation, _ := a.operation(tc.operation)
		input, uid, err := challengeInput(operation, tc.word, tc.args)
		if err != nil {
			t.Errorf("%s %v: %v", tc.word, tc.args, err)
			continue
		}
		_, simple := input["need_all"]
		if input["schedule_type"] != nil && input["schedule_type"] != tc.period || input["schedule_type"] == nil && tc.period != 0 || simple != tc.simple || simple && input["need_all"] != false || uid != tc.uid {
			t.Errorf("%s %v = %v %q, want period %d simple %v uid %q", tc.word, tc.args, input, uid, tc.period, tc.simple, tc.uid)
		}
	}
	// A word standing apart must be one the command's trigger takes after
	// its name: 往期 is Anomaly Arbitration's alone.
	for _, tc := range []struct {
		operation string
		args      []string
	}{{"starrail.abyss", []string{"往期"}}, {"starrail.challenge", []string{"3"}}, {"starrail.rogue", []string{"简易"}}, {"starrail.peak", []string{"4"}}} {
		operation, _ := a.operation(tc.operation)
		if input, _, err := challengeInput(operation, operation.Command, tc.args); err == nil {
			t.Errorf("%s %v accepted: %v", tc.operation, tc.args, input)
		}
	}
}
