package app

import (
	"encoding/json"
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

func TestOfficialCloudRollRestoration(t *testing.T) {
	rolls, err := cloudRolls(testGame(t), 5, PanelStat{Key: "cpct", Value: "6.48%"}, 2)
	if err != nil || len(rolls) != 1 {
		t.Fatal(rolls, err)
	}
	if _, err = cloudRolls(testGame(t), 5, PanelStat{Key: "cpct", Value: "1000%"}, 1); err == nil {
		t.Fatal("unrepresentable stat exported")
	}
}
func TestCloudCapturePromotionResolutionUsesPinnedBaseAttributes(t *testing.T) {
	for _, key := range []string{"sr_1102"} {
		engine := calcEngine(t)
		metadata := engine.Metadata()
		var cases []struct {
			Key   string
			Input BuildProfile
		}
		if err := json.Unmarshal(pluginFile(t, "internal/assets/testdata/calc-vectors.json"), &cases); err != nil {
			t.Fatal(err)
		}
		var profile BuildProfile
		var record reference.Character
		for _, c := range cases {
			if c.Key == key && c.Input.Rank == 0 {
				profile = c.Input
				break
			}
		}
		for _, r := range metadata.Characters {
			if r.Key == key {
				record = r
				break
			}
		}
		profile.Promote = nil
		profile.Weapon.Promote = nil
		cp, wp, err := resolveCloudPromotions(t.Context(), engine, record, profile)
		if err != nil || cp != 6 || wp != 6 {
			t.Fatal(key, cp, wp, err)
		}
	}
}
