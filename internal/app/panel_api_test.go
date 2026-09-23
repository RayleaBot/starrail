package app

import (
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// panel_api counts from 1 as upstream's panelApi.yaml default; a value the
// list does not have reads as the first service.
func TestPanelAPISettingPicksAService(t *testing.T) {
	a := &App{Game: Game{PanelAPIs: []string{"https://one.example/v1/info/", "https://two.example/v1/info/"}}}
	for setting, want := range map[any]int{nil: 1, 2: 2, 0: 1, 3: 1, "x": 1} {
		event := &rayleabot.EventContext{Config: map[string]any{}}
		if setting != nil {
			event.Config["panel_api"] = setting
		}
		if index, api := a.panelAPI(event); index != want || apiHost(api) != []string{"one.example", "two.example"}[want-1] {
			t.Errorf("panel_api %v picked %d %s, want %d", setting, index, api, want)
		}
	}
}
