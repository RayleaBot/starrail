package showcase_test

import (
	"errors"
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-starrail/internal/assets"
	"github.com/RayleaBot/plugin-starrail/internal/showcase"
)

// The fixture follows Mihomo's parsed answer with one relic; its values are
// made up.
const mihomoFixture = `{"player":{"uid":"800000001","nickname":"开拓者","level":70},"characters":[{"id":"1413","rank":3,"level":80,"promotion":6,"enhanced":false,
"element":{"id":"Ice"},"skill_trees":[{"id":"1413001","level":6},{"id":"1413002","level":10},{"id":"1413101","level":1},{"id":"1413201","level":0},{"id":"1413301","level":6}],
"light_cone":{"id":"23049","name":"光锥","rarity":5,"rank":1,"level":80,"promotion":6,"attributes":[{"field":"hp","name":"基础生命值","value":1164.24}],"properties":[]},
"relics":[{"id":"61271","name":"救世主的登程风帽","type":1,"set_name":"再创天地的救世主","rarity":5,"level":15,
"main_affix":{"type":"HPDelta","field":"hp","name":"生命值","value":705.6,"percent":false},
"sub_affix":[{"type":"SpeedDelta","field":"spd","name":"速度","value":4.6,"percent":false,"count":2},{"type":"CriticalDamageBase","field":"crit_dmg","name":"暴击伤害","value":0.18144,"percent":true,"count":3}]}],
"attributes":[{"field":"hp","value":2483.712},{"field":"spd","value":99},{"field":"crit_rate","value":0.05},{"field":"crit_dmg","value":0.5}],
"additions":[{"field":"hp","value":5125.01},{"field":"spd","value":4.6},{"field":"crit_rate","value":0.6558},{"field":"sp_rate","value":0.1944}]}]}`

func TestParseReadsMihomoLikeMiao(t *testing.T) {
	app, err := gamekit.New(assets.Kit(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	profile, err := showcase.Parse(t.Context(), app.Game, app.Catalog, []byte(mihomoFixture))
	if err != nil || profile.Nickname != "开拓者" || len(profile.Panels) != 1 {
		t.Fatal(profile, err)
	}
	panel := profile.Panels[0]
	if panel.Level != 80 || panel.Rank != 3 || panel.Element != "ice" || panel.Weapon == nil || panel.Weapon.Refinement != 1 {
		t.Fatalf("identity = %+v", panel)
	}
	// Properties add the traces and relics to the base, shown as the game
	// cuts them; energy regeneration counts from 100%.
	stats := map[string]gamekit.PanelStat{}
	for _, stat := range panel.Stats {
		stats[stat.ID] = stat
	}
	if stats["1"].Value != "7608" || stats["1"].Base != "2483" || stats["4"].Value != "103.6" || stats["5"].Value != "70.5%" || stats["9"].Value != "119.4%" || stats["16"].Value != "0.0%" {
		t.Fatalf("stats = %+v", stats)
	}
	piece := panel.Equipment[0]
	if piece.Slot != 1 || piece.SetName != "再创天地的救世主" || piece.Main[0].Key != "hpPlus" || piece.Main[0].Value != "705" {
		t.Fatalf("relic = %+v", piece)
	}
	if piece.Sub[0].Key != "speed" || piece.Sub[0].Value != "4.6" || piece.Sub[1].Value != "18.1%" || piece.Sub[1].Times != 2 {
		t.Fatalf("substats = %+v", piece.Sub)
	}
	// Eidolon three adds two skill levels to 长夜月's skill and one to her
	// basic attack; activated traces are traces, the rest are left out.
	levels, traces := map[string]int{}, []string{}
	for _, skill := range panel.Skills {
		if skill.PointType == 2 {
			levels[skill.Kind] = skill.Level
		} else {
			traces = append(traces, skill.ID)
		}
	}
	if levels["普攻"] != 7 || levels["战技"] != 12 || levels["忆灵技"] != 6 || len(traces) != 1 || traces[0] != "1413101" {
		t.Fatalf("talents = %v, traces = %v", levels, traces)
	}
	if _, err = showcase.Parse(t.Context(), app.Game, app.Catalog, []byte(`{"player":{"uid":"1"},"characters":[]}`)); !errors.Is(err, gamekit.ErrShowcaseEmpty) {
		t.Fatal(err)
	}
}
