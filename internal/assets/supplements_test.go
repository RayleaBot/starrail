package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

func supplementalProfile(t *testing.T, key string, rank int) (reference.Character, map[string]any) {
	t.Helper()
	metadata := calcEngine(t).Metadata()
	var record reference.Character
	for _, r := range metadata.Characters {
		if r.Key == key {
			record = r
		}
	}
	if record.Key == "" {
		t.Fatal("missing supplemental character")
	}
	levelData := record.Data["attr"].(map[string]any)["6"].(map[string]any)
	base := levelData["attrs"].(map[string]any)
	grow := levelData["grow"].(map[string]any)
	attributes := map[string]any{"hp": 10000.0, "atk": 1000.0, "def": 1000.0, "speed": 100.0, "cpct": 0.0, "cdmg": 100.0, "dmg": 0.0, "heal": 0.0, "recharge": 100.0, "effPct": 100.0, "effDef": 0.0, "stance": 0.0, "joy": 0.0}
	for _, key := range []string{"hp", "atk", "def", "speed"} {
		n := base[key].(float64)
		if increase, ok := grow[key].(float64); ok {
			n += increase * 79
		}
		attributes[key+"Base"] = n
	}
	trace := "1" + record.ID + "103"
	return record, map[string]any{"level": 80, "promote": 6, "rank": rank, "talents": map[string]any{"a": 6, "e": 10, "q": 10, "t": 10}, "trees": []any{trace}, "weapon": map[string]any{"id": "", "level": 0, "promote": 0, "refinement": 1}, "equipment": []any{}, "attributes": attributes, "enemy_level": 80}
}
func TestEnhancedSupplementalFormulaDefinitions(t *testing.T) {
	for _, key := range []string{"sr_2306", "sr_2307"} {
		for _, rank := range []int{0, 6} {
			t.Run(fmt.Sprintf("%s/%d", key, rank), func(t *testing.T) {
				record, input := supplementalProfile(t, key, rank)
				raw, err := calcEngine(t).Run(context.Background(), record, input)
				if err != nil {
					t.Fatal(err)
				}
				var result struct {
					Baseline struct {
						Results []struct {
							Title    string   `json:"title"`
							Expected *float64 `json:"expected"`
							Text     string   `json:"text"`
						} `json:"results"`
					} `json:"baseline"`
				}
				if json.Unmarshal(raw, &result) != nil {
					t.Fatal("invalid result")
				}
				expected := map[int]float64{}
				if key == "sr_2306" {
					baseAtk := input["attributes"].(map[string]any)["atkBase"].(float64)
					atkPct, defence := 0.45, 0.5
					if rank == 6 {
						atkPct, defence = 0.85, 1/1.68
					}
					expected[0] = (1000 + baseAtk*atkPct) * 1.32 * defence * 0.9
					text := "69.0%"
					if rank == 6 {
						text = "99.0%"
					}
					if result.Baseline.Results[1].Text != text {
						t.Fatal("incorrect teammate critical-damage bonus")
					}
					if result.Baseline.Results[2].Text != "32.0%" {
						t.Fatal("talent/Cipher vulnerability did not use four layers")
					}
				} else {
					resistance, vulnerability := 1.0, 1.25
					if rank == 6 {
						resistance, vulnerability = 1.25, 1.45
					}
					expected[2] = 1000 * 1.2 * 1.6 * vulnerability / 1.792 * resistance * 0.9
					expected[5] = 1000 * (2.4 + 0.12*30) * 1.6 * vulnerability / 1.592 * resistance * 0.9
				}
				for index, want := range expected {
					actual := result.Baseline.Results[index].Expected
					if actual == nil || math.Abs(*actual-want) > 1e-7*math.Max(1, want) {
						t.Fatalf("scenario %d: got %v want %g", index, actual, want)
					}
				}
			})
		}
	}
}
