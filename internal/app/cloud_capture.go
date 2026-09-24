package app

import (
	"context"
	"encoding/json"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

// subAttr is the substat of a relic rarity's table with a key, as miao's
// lodash.findKey finds it: the first by ID.
func subAttr(g Game, star int, key string) (string, cloudAttribute, bool) {
	table := g.Data.CloudGear.Meta.StarData[strconv.Itoa(star)].Sub
	ids := []int{}
	for id, attr := range table {
		if n, err := strconv.Atoi(id); err == nil && attr.Key == mapCloudStat(key) {
			ids = append(ids, n)
		}
	}
	if len(ids) == 0 {
		return "", cloudAttribute{}, false
	}
	id := strconv.Itoa(slices.Min(ids))
	return id, table[id], true
}

// officialRolls is MysPanelHSRData's getArtifactAttrId: the roll count is the
// times the record API gives, and the steps are what the value shown adds to
// that many base rolls, in steps, rounded as JavaScript's Math.round. The API
// shows values cut rather than rounded (3.456% as 3.4%), so the steps are not
// checked against the table, as upstream does not.
func officialRolls(g Game, star int, stat PanelStat, times int) ([]any, error) {
	id, attr, ok := subAttr(g, star, stat.Key)
	value, valid := buildStatNumber(stat.Value)
	if !ok || !valid {
		return nil, gameError("cloud_invalid", "官方装备词条不在固定词条表中，未导出此角色。")
	}
	steps := math.Floor((value-float64(times)*attr.Base)/attr.Step + 0.5)
	return []any{id + "," + strconv.Itoa(times) + "," + strconv.Itoa(int(steps))}, nil
}

// closestRolls restores a substat whose value is miao's own, as the panels
// ark hands over and those its OCR reads keep them, or a kept official panel
// without roll counts: the rolls of the pinned table that give the value, or
// come closest to it.
func closestRolls(g Game, star int, stat PanelStat) ([]any, error) {
	value, ok := buildStatNumber(stat.Value)
	if !ok {
		return nil, gameError("cloud_invalid", "装备词条数值不可还原。")
	}
	tolerance := 0.051
	if !strings.HasSuffix(strings.TrimSpace(stat.Value), "%") {
		tolerance = 0.51
	}
	best := math.Inf(1)
	var found []any
	for id, attr := range g.Data.CloudGear.Meta.StarData[strconv.Itoa(star)].Sub {
		if attr.Key != mapCloudStat(stat.Key) {
			continue
		}
		for count := 1; count <= 6; count++ {
			for step := 0; step <= count*2; step++ {
				if diff := math.Abs(attr.Base*float64(count) + attr.Step*float64(step) - value); diff < best {
					best = diff
					found = []any{id + "," + strconv.Itoa(count) + "," + strconv.Itoa(step)}
				}
			}
		}
	}
	if len(found) == 0 || best > tolerance {
		return nil, gameError("cloud_invalid", "装备词条数值与固定词条表无法对应，未导出此角色。")
	}
	return found, nil
}
func mapCloudStat(key string) string {
	if value := map[string]string{"physical": "phy", "lightning": "elec"}[key]; value != "" {
		return value
	}
	return key
}

// showcaseRolls restores a substat a Mihomo panel keeps: Mihomo's roll count
// and the value cut as the game shows it, to one decimal for percentages and
// speed and to a whole number otherwise. miao's HomoApi keeps the count and
// step of Mihomo's raw answer; the step is the one whose value is cut to the
// kept one (allowing for the tables' float noise). Only a two-star relic's
// flat attack and defence, whose steps are smaller than one, can cut two
// steps to the same number; the smaller is taken.
func showcaseRolls(g Game, star int, stat PanelStat) ([]any, error) {
	value, ok := buildStatNumber(stat.Value)
	count := stat.Times + 1
	if !ok || count > 6 {
		return nil, gameError("cloud_invalid", "展柜装备词条数值不可还原。")
	}
	unit := 1.0
	if strings.HasSuffix(strings.TrimSpace(stat.Value), "%") || stat.Key == "speed" {
		unit = 0.1
	}
	for id, attr := range g.Data.CloudGear.Meta.StarData[strconv.Itoa(star)].Sub {
		if attr.Key != mapCloudStat(stat.Key) {
			continue
		}
		for step := 0; step <= count*2; step++ {
			if rolled := attr.Base*float64(count) + attr.Step*float64(step); rolled > value-1e-3 && rolled < value+unit+1e-3 {
				return []any{id + "," + strconv.Itoa(count) + "," + strconv.Itoa(step)}, nil
			}
		}
	}
	return nil, gameError("cloud_invalid", "展柜显示值与固定词条表无法对应，未导出此角色。")
}

// cloudGear writes a relic as miao's player data keeps it, its substats as
// rolls reads them.
func cloudGear(g Game, gear PanelEquipment, rolls func(star int, stat PanelStat) ([]any, error)) (map[string]any, error) {
	if !gear.Complete || len(gear.Main) != 1 {
		return nil, gameError("cloud_invalid", "装备资料不完整，无法导出。")
	}
	star, err := strconv.Atoi(gear.Rarity)
	if err != nil || star < 1 || star > 5 {
		return nil, gameError("cloud_invalid", "装备品质无效。")
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
		return nil, gameError("cloud_invalid", "主词条无法映射。")
	}
	out["mainId"] = mainID

	attrs := []any{}
	for _, stat := range gear.Sub {
		found, err := rolls(star, stat)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, found...)
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

// panelBasis is what miao's player data keeps of a character besides its
// relics: the calculation record, the panel's build, and the promotions of the
// character and its light cone.
type panelBasis struct {
	record         reference.Character
	profile        BuildProfile
	promote, light int
}

func resolvePanelBasis(ctx context.Context, g Game, panel CharacterPanel) (panelBasis, error) {
	record, err := findBuildCharacter(g.Calc, panel)
	if err != nil {
		return panelBasis{}, err
	}
	profile, err := buildProfile(panel, record)
	if err != nil {
		return panelBasis{}, err
	}
	promote, light, err := resolveCloudPromotions(ctx, g.Calc, record, profile)
	return panelBasis{record, profile, promote, light}, err
}

// panelArtis writes a panel's relics as miao keeps them in its player data,
// by slot, the substats as miao's attrIds (id,count,step) restored as miao
// reads the service the panel came from. An official panel keeps the value
// shown and the official roll count, as MysPanelHSR reads them, and its speed
// is corrected; a Mihomo panel keeps the value cut as the game shows it and
// Mihomo's roll count; a panel from ark and a piece read by its OCR keep
// miao's values. A changed panel's piece keeps the substats of the panel it
// was taken from, as miao's ProfileChange copies the relics. basis is the
// panel's own, or nil to resolve it only when the speed needs it.
func panelArtis(ctx context.Context, g Game, panel CharacterPanel, basis *panelBasis) (map[string]map[string]any, error) {
	// A panel kept before the roll counts were kept has them in the official
	// entry.
	official := map[int]map[string]int{}
	for _, field := range []string{"relics", "ornaments"} {
		for _, raw := range asList(panel.Official[field]) {
			gear := asObject(raw)
			times := map[string]int{}
			for _, item := range asList(gear["properties"]) {
				property := asObject(item)
				times[asText(property["property_type"])] = number(property["times"])
			}
			official[number(gear["pos"])] = times
		}
	}
	taken := map[*CharacterPanel]map[string]map[string]any{}
	artis := map[string]map[string]any{}
	for _, gear := range panel.Equipment {
		slot := strconv.Itoa(gear.Slot)
		var copied any
		if from := gear.From; from != nil {
			kept, ok := taken[from]
			if !ok {
				var err error
				if kept, err = panelArtis(ctx, g, *from, nil); err != nil {
					return nil, err
				}
				taken[from] = kept
			}
			copied = kept[slot]["attrIds"]
		}
		piece, err := cloudGear(g, gear, func(star int, stat PanelStat) ([]any, error) {
			if copied != nil {
				return nil, nil
			}
			switch panel.Source {
			case "mihoyo":
				times, ok := official[gear.Slot][stat.ID]
				if !ok {
					times = stat.Times
				}
				if times > 0 {
					return officialRolls(g, star, stat, times)
				}
			case "mihomo":
				return showcaseRolls(g, star, stat)
			}
			return closestRolls(g, star, stat)
		})
		if err != nil {
			return nil, err
		}
		if copied != nil {
			piece["attrIds"] = copied
		}
		artis[slot] = piece
	}
	if panel.Source == "mihoyo" {
		officialSpeed(ctx, g, panel, basis, artis)
	}
	return artis, nil
}

// officialSpeed is MysPanelHSRData's 速度修正. The record API shows speed
// substats cut to whole numbers, so miao calculates the character's speed from
// the relics as restored and, when the official total is more than 0.2 above
// it, adds a step to one speed substat after another, in slot order, counting
// 0.3 for a five-star relic and 0.2 for another, until no more than 0.05 is
// missing or twenty rounds have passed.
func officialSpeed(ctx context.Context, g Game, panel CharacterPanel, basis *panelBasis, artis map[string]map[string]any) {
	target, shown := 0.0, false
	for _, stat := range panel.Stats {
		if stat.ID == "4" {
			target, shown = buildStatNumber(stat.Value)
		}
	}
	slots := slices.SortedFunc(maps.Keys(artis), func(x, y string) int { return number(x) - number(y) })
	speeds := false
	for _, slot := range slots {
		for _, id := range asList(artis[slot]["attrIds"]) {
			speeds = speeds || strings.HasPrefix(asText(id), "7,")
		}
	}
	if !shown || !speeds {
		return
	}
	if basis == nil {
		resolved, err := resolvePanelBasis(ctx, g, panel)
		if err != nil {
			return
		}
		basis = &resolved
	}
	speed, ok := artisSpeed(ctx, g, panel, *basis, artis)
	missing := target - speed
	if !ok || speed == 0 || missing <= 0.2 {
		return
	}
	for round := 0; missing > 0.05 && round < 20; round++ {
		raised := false
		for _, slot := range slots {
			ids := asList(artis[slot]["attrIds"])
			for index, id := range ids {
				parts := strings.Split(asText(id), ",")
				if missing <= 0.05 || len(parts) != 3 || parts[0] != "7" {
					continue
				}
				ids[index] = parts[0] + "," + parts[1] + "," + strconv.Itoa(number(parts[2])+1)
				missing -= map[bool]float64{true: 0.3, false: 0.2}[number(artis[slot]["star"]) >= 5]
				raised = true
			}
		}
		if !raised {
			break
		}
	}
}

// artisSpeed is the speed miao's Attr gives the character wearing the relics
// as restored.
func artisSpeed(ctx context.Context, g Game, panel CharacterPanel, basis panelBasis, artis map[string]map[string]any) (float64, bool) {
	pieces := []PanelEquipment{}
	for slot, piece := range artis {
		_, decoded := decodeCloudGear(g, number(slot), piece)
		if decoded == nil {
			return 0, false
		}
		pieces = append(pieces, *decoded)
	}
	gear, err := buildGearSet(CharacterPanel{EquipmentKnown: true, Equipment: pieces})
	if err != nil {
		return 0, false
	}
	weapon := basis.profile.Weapon
	weapon.Promote = &basis.light
	raw, err := g.Calc.Change(ctx, basis.record, map[string]any{"level": panel.Level, "promote": basis.promote, "rank": panel.Rank,
		"talents": basis.profile.Talents, "trees": basis.profile.Trees, "equipment": gear, "weapon": weapon})
	var result struct {
		Attributes map[string]float64 `json:"attributes"`
	}
	if err != nil || json.Unmarshal(raw, &result) != nil {
		return 0, false
	}
	return result.Attributes["speed"], true
}

// cloudAvatar writes a kept panel as miao keeps the character in its player
// data, whichever service the panel came from, its relics as panelArtis
// restores them. source is the service the character is saved under
// (_source): share for data exchanged with ark, or miao's name for the
// service a kept panel came from; the times are when the panel was kept, or
// now for one that was not.
func cloudAvatar(ctx context.Context, g Game, panel CharacterPanel, source string) (string, json.RawMessage, error) {
	if g.Data == nil || g.Data.CloudGear == nil {
		return "", nil, gameError("cloud_invalid", "固定装备资料无法读取。")
	}
	basis, err := resolvePanelBasis(ctx, g, panel)
	if err != nil {
		return "", nil, err
	}
	record, profile := basis.record, basis.profile
	saved := panel.UpdatedAtMS
	if saved <= 0 {
		saved = time.Now().UnixMilli()
	}
	artis, err := panelArtis(ctx, g, panel, &basis)
	if err != nil {
		return "", nil, err
	}
	avatar := map[string]any{"_time": saved, "_update": saved, "_talent": saved, "id": record.ID, "name": record.Name, "elem": record.Element, "level": panel.Level, "promote": basis.promote, "cons": panel.Rank, "talent": originalCloudTalents(record, profile), "trees": profile.Trees, "artis": artis}
	avatar["weapon"] = nil
	if panel.Weapon != nil {
		avatar["weapon"] = map[string]any{"id": profile.Weapon.ID, "level": profile.Weapon.Level, "promote": basis.light, "affix": profile.Weapon.Refinement}
	}
	// Normalize through JSON so exported slices use the same representation as imported files.
	var data map[string]any
	_ = decodeObject(avatar, &data)
	exportID, raw, err := cleanCloudAvatar(data)
	if err != nil || source == "share" {
		return exportID, raw, err
	}
	// The clean-up marks every character share, as it marks imported data.
	var cleaned map[string]any
	if err = json.Unmarshal(raw, &cleaned); err != nil {
		return "", nil, err
	}
	cleaned["_source"] = source
	raw, err = json.Marshal(cleaned)
	return exportID, raw, err
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
	exportID, data, err := cloudAvatar(ctx, a.Game, *panel, "share")
	if err != nil {
		return nil, err
	}
	if err = a.CloudArchive.Update(client.Provider, choice, archive.Revision, role, map[string]json.RawMessage{exportID: data}, false); err != nil {
		return nil, err
	}
	return map[string]any{"captured": exportID, "note": "已保存官方面板的交换副本。强化档位按显示值还原，存在舍入误差，不代表真实升级顺序；尚未上传云服务。"}, nil
}
