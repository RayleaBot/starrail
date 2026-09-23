package app

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

// The published record API exposes rounded values and enhancement counts, not
// the original roll order. Restore the closest values in the pinned tables.
func cloudRolls(g Game, star int, stat PanelStat, times any) ([]any, error) {
	d := *g.Data.CloudGear
	key := stat.Key
	key = mapCloudStat(key)

	value, ok := buildStatNumber(stat.Value)
	if !ok {
		return nil, gameError("cloud_invalid", "官方装备词条数值不可还原。")
	}
	tolerance := 0.051
	if !strings.HasSuffix(strings.TrimSpace(stat.Value), "%") {
		tolerance = 0.51
	}
	if key == "speed" && !strings.Contains(stat.Value, ".") {
		tolerance = 0.999
	}
	count, hasCount := cloudInteger(times, 0, 6)
	counts := []int{1, 2, 3, 4, 5, 6}
	if hasCount {
		if count >= 1 && count <= 6 {
			counts = []int{count}
		}
	}
	best := math.Inf(1)
	var found []any
	table := d.Meta.StarData[strconv.Itoa(star)]
	for id, attr := range table.Sub {
		if attr.Key != key {
			continue
		}
		for _, n := range counts {
			for step := 0; step <= n*2; step++ {
				v := attr.Base*float64(n) + attr.Step*float64(step)
				diff := math.Abs(v - value)
				if key == "speed" && !strings.Contains(stat.Value, ".") && v < value {
					continue
				}
				if diff < best {
					best = diff
					found = []any{id + "," + strconv.Itoa(n) + "," + strconv.Itoa(step)}
				}
			}
		}
	}

	if len(found) == 0 || best > tolerance {
		return nil, gameError("cloud_invalid", "官方显示值与固定词条表无法合理对应，未导出此角色。")
	}
	return found, nil
}
func mapCloudStat(key string) string {
	if value := map[string]string{"physical": "phy", "lightning": "elec"}[key]; value != "" {
		return value
	}
	return key
}

func officialCloudGear(g Game, gear PanelEquipment, raw map[string]any) (map[string]any, error) {
	if !gear.Complete || len(gear.Main) != 1 {
		return nil, gameError("cloud_invalid", "官方装备资料不完整，无法导出。")
	}
	star, err := strconv.Atoi(gear.Rarity)
	if err != nil || star < 1 || star > 5 {
		return nil, gameError("cloud_invalid", "官方装备品质无效。")
	}
	d := *g.Data.CloudGear
	itemKey := gear.ID
	item, ok := d.Items[itemKey]
	if !ok || item.Slot != gear.Slot {
		return nil, gameError("cloud_invalid", "固定资料未收录此装备，无法可靠导出。")
	}
	out := map[string]any{"level": gear.Level, "star": star}
	out["id"] = gear.ID
	mainID := ""
	for id, key := range d.Meta.MainIdx[strconv.Itoa(gear.Slot)] {
		if key == mapCloudStat(gear.Main[0].Key) {
			mainID = id
			break
		}
	}
	if mainID == "" {
		return nil, gameError("cloud_invalid", "官方主词条无法映射。")
	}
	out["mainId"] = mainID

	listKey := "properties"
	rows := asList(raw[listKey])
	attrs := []any{}
	for _, stat := range gear.Sub {
		var times any
		for _, v := range rows {
			r := asObject(v)
			if asText(r["property_type"]) == stat.ID {
				times = r["times"]
				break
			}
		}
		rolls, err := cloudRolls(g, star, stat, times)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, rolls...)
	}
	out["attrIds"] = attrs
	return out, nil
}
func resolveCloudPromotions(ctx context.Context, engine *reference.Engine, record reference.Character, profile BuildProfile) (int, int, error) {
	if profile.Promote != nil && profile.Weapon.Promote != nil {
		return *profile.Promote, *profile.Weapon.Promote, nil
	}
	var input map[string]any
	_ = decodeObject(profile, &input)
	input["resolve_identity"] = true
	raw, err := engine.Run(ctx, record, input)
	if err != nil {
		return 0, 0, gameError("cloud_invalid", "无法确认角色或武器突破阶段，未导出。")
	}
	var out struct {
		Promote int `json:"promote"`
		Weapon  int `json:"weapon_promote"`
	}
	if json.Unmarshal(raw, &out) != nil {
		return 0, 0, gameError("cloud_invalid", "突破阶段还原结果无效。")
	}
	return out.Promote, out.Weapon, nil
}
func originalCloudTalents(record reference.Character, profile BuildProfile) map[string]int {
	out := map[string]int{}
	cons := asObject(record.Data["talentCons"])
	for key, level := range profile.Talents {
		step := map[string]int{"a": 1, "e": 2, "q": 2, "t": 2, "me": 1, "mt": 1, "xe": 1}[key]
		targets := asList(cons[key])
		if targets == nil {
			targets = []any{cons[key]}
		}
		for _, raw := range targets {
			target, ok := cloudInteger(raw, 1, 6)
			if ok && profile.Rank >= target {
				level -= step
			}
		}
		out[key] = max(1, level)
	}
	return out
}
func officialCloudAvatar(ctx context.Context, g Game, panel CharacterPanel, raw map[string]any) (string, json.RawMessage, error) {
	game, engine := g.ID, g.Calc
	if g.Data == nil || g.Data.CloudGear == nil {
		return "", nil, gameError("cloud_invalid", "固定装备资料无法读取。")
	}
	record, err := findBuildCharacter(engine, panel)
	if err != nil {
		return "", nil, err
	}
	var _ string = game
	profile, err := buildProfile(panel, record)
	if err != nil {
		return "", nil, err
	}
	promote, wp, err := resolveCloudPromotions(ctx, engine, record, profile)
	if err != nil {
		return "", nil, err
	}
	id := record.ID
	now := time.Now().UnixMilli()
	avatar := map[string]any{"_time": now, "_update": now, "_talent": now, "id": id, "name": record.Name, "elem": record.Element, "level": panel.Level, "promote": promote, "cons": panel.Rank, "talent": originalCloudTalents(record, profile), "trees": profile.Trees, "_source": "share", "artis": map[string]any{}}
	if panel.Weapon == nil {
		avatar["weapon"] = nil
	} else {
		weapon := map[string]any{"level": profile.Weapon.Level, "promote": wp, "affix": profile.Weapon.Refinement}
		weapon["id"] = profile.Weapon.ID

		avatar["weapon"] = weapon
	}
	rawGear := map[int]map[string]any{}
	for _, field := range []string{"relics", "ornaments"} {
		for _, v := range asList(raw[field]) {
			gear := asObject(v)
			rawGear[number(gear["pos"])] = gear
		}
	}
	for _, gear := range panel.Equipment {
		converted, err := officialCloudGear(g, gear, rawGear[gear.Slot])
		if err != nil {
			return "", nil, err
		}
		avatar["artis"].(map[string]any)[strconv.Itoa(gear.Slot)] = converted
	}
	// Normalize through JSON so exported slices use the same representation as imported files.
	var data map[string]any
	_ = decodeObject(avatar, &data)
	return cleanCloudAvatar(data)
}
func (a *App) captureCloudPanel(ctx context.Context, client AccountsClient, choice Selection, role Role, archive CloudArchive, id string) (map[string]any, error) {
	if _, err := strconv.Atoi(id); err != nil {
		if entry, ok := a.Catalog.Resolve(id, "character", nil); ok {
			id = entry.ID
		}
	}
	if !cloudIDs([]string{id}, 1, false) {
		return nil, gameError("input_invalid", "请选择角色名称或编号。")
	}
	result, err := client.Execute(ctx, choice, a.Game.ID+".character", map[string]any{"character_ids": []any{id}})
	if err != nil {
		return nil, err
	}
	if result.Role.Ref != role.Ref || result.Role.UID != role.UID || result.Role.Region != role.Region {
		return nil, gameError("role_missing", "角色身份已变化，请重新选择。")
	}
	var panel *CharacterPanel
	for _, p := range NormalizePanels(result, a.Catalog) {
		if p.ID == id {
			copy := p
			panel = &copy
			break
		}
	}
	if panel == nil {
		return nil, gameError("character_missing", "官方未返回此角色。")
	}
	var raw map[string]any
	for _, v := range asList(result.Data["avatar_list"]) {
		if item := asObject(v); asText(item["id"]) == id {
			raw = item
			break
		}
	}
	exportID, data, err := officialCloudAvatar(ctx, a.Game, *panel, raw)
	if err != nil {
		return nil, err
	}
	if err = a.CloudArchive.Update(client.Provider, choice, archive.Revision, role, map[string]json.RawMessage{exportID: data}, false); err != nil {
		return nil, err
	}
	return map[string]any{"captured": exportID, "note": "已保存官方面板的交换副本。强化档位按显示值还原，存在舍入误差，不代表真实升级顺序；尚未上传云服务。"}, nil
}
