package app

import (
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// 换装 words that end in a panel, 遗器 or 伤害 word reach those commands
// first; as in miao, they answer as 面板换装 does.
func TestPanelWordsHandChangesToPanelChange(t *testing.T) {
	host := newSDKHost(t, pluginApp(t), func(rayleabot.ServiceCallRequest, bool) (map[string]any, string) {
		return nil, "plugin.service_unavailable"
	})
	end, _ := host.message("*希儿换满命", "希儿换满命")
	want := terminalText(end)
	if want == "" {
		t.Fatalf("面板换装 answered %+v", end)
	}
	for _, word := range []string{"希儿换银狼遗器", "希儿换银狼圣遗物", "希儿换满命面板", "希儿换满命详情", "希儿换满命伤害", "希儿换满命换满行迹"} {
		if end, _ := host.message("*"+word, word); terminalText(end) != want {
			t.Errorf("%s answered %q, want %q", word, terminalText(end), want)
		}
	}
}
