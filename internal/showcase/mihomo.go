// Package showcase reads Star Rail panels from Mihomo, the showcase service
// miao-plugin refreshes from without an account.
package showcase

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/reference"
)

// Source is Mihomo's parsed Star Rail API in Chinese. miao reads Mihomo's
// raw answer and adds up the properties itself; the parsed answer carries
// the same panel with its properties added up.
var Source = app.ShowcaseSource{Name: "Mihomo", URL: func(uid string) string { return "https://api.mihomo.me/sr_info_parsed/" + uid + "?lang=cn" }, Parse: Parse}

type mihomoAnswer struct {
	Player *struct {
		UID      string `json:"uid"`
		Nickname string `json:"nickname"`
		Level    int    `json:"level"`
	} `json:"player"`
	Characters []mihomoCharacter `json:"characters"`
}

type mihomoCharacter struct {
	ID         string              `json:"id"`
	Rank       int                 `json:"rank"`
	Level      int                 `json:"level"`
	Promotion  int                 `json:"promotion"`
	Enhanced   bool                `json:"enhanced"`
	Element    struct{ ID string } `json:"element"`
	SkillTrees []struct {
		ID    string `json:"id"`
		Level int    `json:"level"`
	} `json:"skill_trees"`
	LightCone *struct {
		ID         string       `json:"id"`
		Name       string       `json:"name"`
		Rarity     int          `json:"rarity"`
		Rank       int          `json:"rank"`
		Level      int          `json:"level"`
		Promotion  int          `json:"promotion"`
		Attributes []mihomoStat `json:"attributes"`
		Properties []mihomoStat `json:"properties"`
	} `json:"light_cone"`
	Relics []struct {
		ID        string       `json:"id"`
		Name      string       `json:"name"`
		Type      int          `json:"type"`
		SetName   string       `json:"set_name"`
		Rarity    int          `json:"rarity"`
		Level     int          `json:"level"`
		MainAffix mihomoStat   `json:"main_affix"`
		SubAffix  []mihomoStat `json:"sub_affix"`
	} `json:"relics"`
	Attributes []mihomoStat `json:"attributes"`
	Additions  []mihomoStat `json:"additions"`
}

type mihomoStat struct {
	Type    string  `json:"type"`
	Field   string  `json:"field"`
	Name    string  `json:"name"`
	Value   float64 `json:"value"`
	Percent bool    `json:"percent"`
	Count   int     `json:"count"`
}

// panelStats are the properties the official panel lists, by Mihomo's
// field, with the base the game counts from.
var panelStats = []struct {
	field, id, name string
	percent         bool
	base            float64
}{
	{"hp", "1", "生命值", false, 0}, {"atk", "2", "攻击力", false, 0}, {"def", "3", "防御力", false, 0}, {"spd", "4", "速度", false, 0},
	{"crit_rate", "5", "暴击率", true, 0}, {"crit_dmg", "6", "暴击伤害", true, 0}, {"heal_rate", "7", "治疗量加成", true, 0}, {"sp_rate", "9", "能量恢复效率", true, 1},
	{"effect_hit", "10", "效果命中", true, 0}, {"effect_res", "11", "效果抵抗", true, 0}, {"break_dmg", "58", "击破特攻", true, 0},
	{"physical_dmg", "12", "物理属性伤害提高", true, 0}, {"fire_dmg", "14", "火属性伤害提高", true, 0}, {"ice_dmg", "16", "冰属性伤害提高", true, 0},
	{"lightning_dmg", "18", "雷属性伤害提高", true, 0}, {"wind_dmg", "20", "风属性伤害提高", true, 0}, {"quantum_dmg", "22", "量子属性伤害提高", true, 0},
	{"imaginary_dmg", "24", "虚数属性伤害提高", true, 0},
}

// relicStats are the official relic property IDs and miao's attribute keys
// by Mihomo's property type.
var relicStats = map[string][2]string{
	"HPDelta": {"27", "hpPlus"}, "AttackDelta": {"29", "atkPlus"}, "DefenceDelta": {"31", "defPlus"}, "HPAddedRatio": {"32", "hp"}, "AttackAddedRatio": {"33", "atk"},
	"DefenceAddedRatio": {"34", "def"}, "SpeedDelta": {"51", "speed"}, "CriticalChanceBase": {"52", "cpct"}, "CriticalDamageBase": {"53", "cdmg"}, "SPRatioBase": {"54", "recharge"},
	"HealRatioBase": {"55", "heal"}, "StatusProbabilityBase": {"56", "effPct"}, "StatusResistanceBase": {"57", "effDef"}, "BreakDamageAddedRatioBase": {"58", "stance"},
	"PhysicalAddedRatio": {"12", "physical"}, "FireAddedRatio": {"14", "fire"}, "IceAddedRatio": {"16", "ice"}, "ThunderAddedRatio": {"18", "lightning"},
	"WindAddedRatio": {"20", "wind"}, "QuantumAddedRatio": {"22", "quantum"}, "ImaginaryAddedRatio": {"24", "imaginary"},
}

// elements are the official element names by Mihomo's.
var elements = map[string]string{"Physical": "physical", "Fire": "fire", "Ice": "ice", "Thunder": "lightning", "Wind": "wind", "Quantum": "quantum", "Imaginary": "imaginary"}

// skillKinds are the official skill names of miao's talent keys.
var skillKinds = map[string]string{"a": "普攻", "e": "战技", "q": "终结技", "t": "天赋", "z": "秘技", "xe": "欢愉技", "me": "忆灵技", "mt": "忆灵天赋"}

// skillSteps are the levels an eidolon adds to each talent, as miao counts
// them.
var skillSteps = map[string]int{"a": 1, "e": 2, "q": 2, "t": 2, "me": 1, "mt": 1, "xe": 1}

// Game values are shown the way the game shows them: whole numbers and
// percentages with one decimal, cut rather than rounded; speed keeps a
// decimal.
func display(value float64, percent bool, field string) string {
	if percent {
		return strconv.FormatFloat(math.Floor(value*1000+1e-6)/10, 'f', 1, 64) + "%"
	}
	if field == "spd" {
		return strconv.FormatFloat(math.Floor(value*10+1e-6)/10, 'f', -1, 64)
	}
	return strconv.FormatFloat(math.Floor(value+1e-6), 'f', 0, 64)
}

func relicStat(stat mihomoStat) (app.PanelStat, bool) {
	ids, ok := relicStats[stat.Type]
	if !ok {
		return app.PanelStat{}, false
	}
	return app.PanelStat{ID: ids[0], Key: ids[1], Name: stat.Name, Value: display(stat.Value, stat.Percent, stat.Field), Times: max(0, stat.Count-1)}, true
}

// Parse reads Mihomo's answer the way miao's HomoApi does: every character
// the player shows with its light cone, relics, traces and talents.
func Parse(_ context.Context, game app.Game, catalog app.Catalog, raw []byte) (app.ShowcaseProfile, error) {
	var answer mihomoAnswer
	if err := json.Unmarshal(raw, &answer); err != nil || answer.Player == nil {
		return app.ShowcaseProfile{}, &app.ShowcaseFailure{Status: 200}
	}
	profile := app.ShowcaseProfile{Nickname: answer.Player.Nickname, Level: answer.Player.Level, TTL: 5 * time.Minute}
	if len(answer.Characters) == 0 {
		return profile, app.ErrShowcaseEmpty
	}
	for _, character := range answer.Characters {
		if panel, ok := parseCharacter(game, catalog, character); ok {
			profile.Panels = append(profile.Panels, panel)
		}
	}
	return profile, nil
}

func parseCharacter(game app.Game, catalog app.Catalog, character mihomoCharacter) (app.CharacterPanel, bool) {
	entry, ok := catalog.Get(character.ID)
	record, found := calcRecord(game, character.ID)
	if !ok || !found {
		return app.CharacterPanel{}, false
	}
	promote := character.Promotion
	panel := app.CharacterPanel{ID: character.ID, Name: entry.Name, Level: character.Level, Promote: &promote, Rank: character.Rank, RankKnown: true, Element: elements[character.Element.ID], Source: "mihomo",
		WeaponKnown: true, EquipmentKnown: true, Stats: []app.PanelStat{}, Equipment: []app.PanelEquipment{}, Skills: []app.PanelSkill{}, Ranks: []app.PanelSkill{}}
	base, added := map[string]float64{}, map[string]float64{}
	for _, stat := range character.Attributes {
		base[stat.Field] += stat.Value
	}
	for _, stat := range character.Additions {
		added[stat.Field] += stat.Value
	}
	for _, stat := range panelStats {
		from := base[stat.field] + stat.base
		item := app.PanelStat{ID: stat.id, Name: stat.name, Value: display(from+added[stat.field], stat.percent, stat.field), Base: display(from, stat.percent, stat.field), Added: display(added[stat.field], stat.percent, stat.field)}
		panel.Stats = append(panel.Stats, item)
	}
	if cone := character.LightCone; cone != nil {
		promotion := cone.Promotion
		w := app.PanelEquipment{ID: cone.ID, Name: cone.Name, Level: cone.Level, Promote: &promotion, Rarity: strconv.Itoa(cone.Rarity), Refinement: cone.Rank, Main: []app.PanelStat{}, Sub: []app.PanelStat{}, Complete: true}
		if weapon, ok := catalog.Get(cone.ID); ok {
			w.Name = weapon.Name
		}
		for _, stat := range cone.Attributes {
			w.Main = append(w.Main, app.PanelStat{ID: stat.Field, Name: stat.Name, Value: display(stat.Value, stat.Percent, stat.Field)})
		}
		for _, stat := range cone.Properties {
			if item, ok := relicStat(stat); ok {
				w.Sub = append(w.Sub, item)
			}
		}
		panel.Weapon = &w
	}
	for _, relic := range character.Relics {
		piece := app.PanelEquipment{ID: relic.ID, Name: relic.Name, Slot: relic.Type, Level: relic.Level, Rarity: strconv.Itoa(relic.Rarity), SetName: relic.SetName, Main: []app.PanelStat{}, Sub: []app.PanelStat{}}
		if main, ok := relicStat(relic.MainAffix); ok {
			main.Times = 0
			piece.Main = append(piece.Main, main)
		}
		for _, sub := range relic.SubAffix {
			if stat, ok := relicStat(sub); ok {
				piece.Sub = append(piece.Sub, stat)
			}
		}
		piece.Complete = len(piece.Main) == 1
		panel.Equipment = append(panel.Equipment, piece)
	}
	slices.SortFunc(panel.Equipment, func(a, b app.PanelEquipment) int { return a.Slot - b.Slot })
	panel.Skills = skills(record, character)
	constellations, _ := record.Data["cons"].(map[string]any)
	for index := 1; index <= 6; index++ {
		item, _ := constellations[strconv.Itoa(index)].(map[string]any)
		name, _ := item["name"].(string)
		panel.Ranks = append(panel.Ranks, app.PanelSkill{Name: name, Active: index <= character.Rank})
	}
	return panel, true
}

// skills are the trace points: miao's getTalentKey names a point's talent
// (by the record's talentId, or by the point's last digits), whose level gets
// the eidolons' bonus; the other activated points are traces. An enhanced
// character's points start with 1, as the official panel gives them.
func skills(record reference.Character, character mihomoCharacter) []app.PanelSkill {
	talentID, _ := record.Data["talentId"].(map[string]any)
	cons, _ := record.Data["talentCons"].(map[string]any)
	result := []app.PanelSkill{}
	for _, point := range character.SkillTrees {
		id := point.ID
		if character.Enhanced && !strings.HasPrefix(id, "1"+character.ID) {
			id = "1" + id
		}
		key, _ := talentID[point.ID].(string)
		if key == "" {
			key = map[string]string{"001": "a", "002": "e", "003": "q", "004": "t", "007": "z", "301": "me", "302": "mt", "420": "xe"}[strings.TrimPrefix(strings.TrimPrefix(point.ID, "1"+character.ID), character.ID)]
		}
		if kind := skillKinds[key]; kind != "" {
			plus := 0
			switch need := cons[key].(type) {
			case float64:
				if need > 0 && float64(character.Rank) >= need {
					plus = skillSteps[key]
				}
			case []any:
				for _, level := range need {
					if value, ok := level.(float64); ok && float64(character.Rank) >= value {
						plus += skillSteps[key]
					}
				}
			}
			result = append(result, app.PanelSkill{Kind: kind, PointType: 2, ID: id, Level: point.Level + plus, ExtraLevel: plus, Active: point.Level > 0, Name: kind})
			continue
		}
		if point.Level > 0 {
			result = append(result, app.PanelSkill{PointType: 1, ID: id, Level: point.Level, Active: true, Name: "行迹 " + id})
		}
	}
	return result
}

// calcRecord is the character's upstream record.
func calcRecord(game app.Game, id string) (reference.Character, bool) {
	if game.Calc == nil {
		return reference.Character{}, false
	}
	for _, record := range game.Calc.Metadata().Characters {
		if record.ID == id {
			return record, true
		}
	}
	return reference.Character{}, false
}
