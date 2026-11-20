package app

import (
	"context"
	"encoding/json"

	"github.com/RayleaBot/starrail/internal/reference"
)

// ProfileNumbers holds the formatted static attributes used by the character card.
type ProfileNumbers struct {
	Stats  map[string]PanelStat `json:"stats"`
	Weapon []PanelStat          `json:"weapon"`
}

func (g Game) profileNumbers(ctx context.Context, panel CharacterPanel, record reference.Character) ProfileNumbers {
	if g.Calc == nil || record.ID == "" || panel.Level < 1 || (panel.Weapon == nil && !panel.WeaponKnown) {
		return ProfileNumbers{}
	}
	weapon := BuildWeapon{Refinement: 1}
	if w := panel.Weapon; w != nil {
		weapon = BuildWeapon{ID: w.ID, Level: w.Level, Promote: w.Promote, Refinement: w.Refinement}
	}
	gear, err := buildGearSet(panel)
	if err != nil {
		gear = []BuildGear{}
	} else {
		relics := miaoRelics(ctx, g, panel)
		for i, piece := range gear {
			if exact, ok := relics[piece.Slot]; ok {
				gear[i].Main, gear[i].Sub = exact.Main, exact.Sub
			}
		}
	}
	input := map[string]any{"level": panel.Level, "promote": panel.Promote, "rank": panel.Rank, "weapon": weapon,
		"equipment": gear, "trees": activeTrees(panel, record), "observed": observedAttributes(panel), "complete": err == nil && panel.RankKnown}
	raw, err := g.Calc.Change(ctx, record, input)
	var result ProfileNumbers
	if err != nil || json.Unmarshal(raw, &result) != nil {
		return ProfileNumbers{}
	}
	return result
}
