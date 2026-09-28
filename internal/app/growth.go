package app

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type GrowthSkill struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Current int    `json:"current_level"`
	Target  int    `json:"target_level"`
	Max     int    `json:"max_level"`
}
type GrowthPlan struct {
	CharacterID   string        `json:"character_id"`
	Name          string        `json:"name"`
	Current       int           `json:"current_level"`
	Target        int           `json:"target_level"`
	Max           int           `json:"max_level"`
	WeaponID      int           `json:"weapon_id"`
	WeaponName    string        `json:"weapon_name"`
	WeaponCurrent int           `json:"current_weapon_level"`
	WeaponTarget  int           `json:"target_weapon_level"`
	WeaponMax     int           `json:"max_weapon_level"`
	Skills        []GrowthSkill `json:"skills"`
}

// growthSkillKinds are the calculator's skill icons and point-id suffixes in
// the order the game lists a character's skills; the catalog names them.
var growthSkillKinds = []struct {
	icon, kind string
}{{"_normal", "普攻"}, {"_bp", "战技"}, {"_ultra", "终结技"}, {"_passive", "天赋"}, {"_maze", "秘技"}}

// growthSkillName names a calculator skill point, which the official
// calculator lists without a name: by its icon or its point-id suffix through
// the catalog's talents, traces by their kind, the rest by their ID.
func growthSkillName(catalog Catalog, characterID string, raw map[string]any, id int) string {
	entry, hasEntry := catalog.Get(characterID)
	byKind := func(kind string) string {
		if hasEntry {
			for _, talent := range entry.Talents {
				if talent.Type == kind {
					return talent.Name
				}
			}
		}
		return kind
	}
	icon := strings.ToLower(asText(raw["item_url"]))
	for _, skill := range growthSkillKinds {
		if icon != "" && strings.Contains(icon, skill.icon) {
			return byKind(skill.kind)
		}
	}
	if strings.Contains(icon, "skilltree") {
		return "额外能力"
	}
	character, _ := strconv.Atoi(characterID)
	suffix := -1
	if character > 0 && id/1000 == character {
		suffix = id % 1000
	} else if character > 0 && id/100 == character {
		suffix = id % 100
	}
	if suffix >= 1 && suffix <= len(growthSkillKinds) {
		return byKind(growthSkillKinds[suffix-1].kind)
	}
	if suffix >= 101 && suffix <= 199 {
		return "额外能力 " + strconv.Itoa(suffix-100)
	}
	if suffix >= 201 && suffix <= 299 {
		return "属性加成 " + strconv.Itoa(suffix-200)
	}
	switch number(raw["point_type"]) {
	case 1:
		return "属性加成 " + strconv.Itoa(id)
	case 3:
		return "额外能力 " + strconv.Itoa(id)
	}
	return "技能 " + strconv.Itoa(id)
}
func prepareGrowth(id string, data map[string]any, catalog Catalog) (GrowthPlan, error) {
	avatar := asObject(data["avatar"])
	if avatar == nil {
		return GrowthPlan{}, gameError("growth_unavailable", "官方未提供此角色的养成详情。")
	}
	maxLevel := 80
	plan := GrowthPlan{CharacterID: id, Name: firstText(avatar, "name", "item_name"), Current: number(firstText(avatar, "level_current", "cur_level", "avatar_level")), Max: number(avatar["max_level"]), Skills: []GrowthSkill{}}
	if plan.Current == 0 {
		plan.Current = number(data["level"])
	}
	if plan.Max < plan.Current || plan.Max > maxLevel || plan.Current < 1 {
		return plan, gameError("growth_unavailable", "养成详情缺少受支持的当前等级或上限。")
	}
	plan.Target = plan.Current
	weapon := asObject(data["weapon"])
	if weapon == nil {
		weapon = asObject(data["equipment"])
	}
	if weapon == nil {
		weapon = asObject(avatar["weapon"])
	}
	if weapon != nil && number(firstText(weapon, "id", "item_id")) != 0 {
		plan.WeaponID = number(firstText(weapon, "id", "item_id"))
		plan.WeaponName = firstText(weapon, "name", "item_name")
		plan.WeaponCurrent = number(firstText(weapon, "level_current", "cur_level", "weapon_level"))
		plan.WeaponMax = number(weapon["max_level"])
		plan.WeaponTarget = plan.WeaponCurrent
		if plan.WeaponCurrent < 1 || plan.WeaponMax < plan.WeaponCurrent || plan.WeaponMax > maxLevel {
			return plan, gameError("growth_unavailable", "武器养成等级暂不兼容。")
		}
	}
	skills := asList(data["skill_list"])
	if skills == nil {
		skills = asList(data["skills"])
	}
	if skills == nil {
		skills = asList(avatar["skill_list"])
	}
	for _, raw := range skills {
		s := asObject(raw)
		skill := GrowthSkill{ID: number(firstText(s, "group_id", "point_id", "id")), Name: firstText(s, "name", "item_name"), Current: number(firstText(s, "level_current", "cur_level")), Max: number(s["max_level"])}
		if skill.ID <= 0 || skill.Current < 0 || skill.Max < 1 || skill.Max < skill.Current {
			return plan, gameError("growth_unavailable", "技能养成详情暂不兼容。")
		}
		if skill.Name == "" {
			skill.Name = growthSkillName(catalog, id, s, skill.ID)
		}
		skill.Target = skill.Current
		plan.Skills = append(plan.Skills, skill)
	}
	if len(plan.Skills) > 32 {
		return plan, gameError("growth_unavailable", "技能条目超出当前支持范围。")
	}
	return plan, nil
}
func (a *App) growthAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	choice := Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}
	id := asText(input["character_id"])
	if action == "growth.prepare" {
		result, err := client.Execute(ctx, choice, a.Game.ID+".growth_detail", map[string]any{"character_id": id})
		if err != nil {
			return nil, err
		}
		plan, err := prepareGrowth(id, result.Data, a.Catalog)
		return map[string]any{"plan": plan}, err
	}
	if action != "growth.compute" {
		return nil, gameError("operation_denied", "养成操作不存在。")
	}
	var plan GrowthPlan
	if decodeObject(input["plan"], &plan) != nil || plan.CharacterID != id {
		return nil, gameError("input_invalid", "养成目标与所选角色不一致。")
	}
	skills := []any{}
	for _, s := range plan.Skills {
		skills = append(skills, map[string]any{"id": s.ID, "current_level": s.Current, "target_level": s.Target})
	}
	parameters := map[string]any{"character_id": id, "current_level": plan.Current, "target_level": plan.Target, "skills": skills}
	if plan.WeaponID != 0 {
		parameters["weapon_id"] = plan.WeaponID
		parameters["current_weapon_level"] = plan.WeaponCurrent
		parameters["target_weapon_level"] = plan.WeaponTarget
	}
	result, err := client.Execute(ctx, choice, a.Game.ID+".growth_compute", parameters)
	if err != nil {
		return nil, err
	}
	view := growthView(a.Game, plan, result)
	changed := plan.Target > plan.Current || plan.WeaponTarget > plan.WeaponCurrent
	for _, s := range plan.Skills {
		changed = changed || s.Target > s.Current
	}
	if changed && len(view.Rows) == 0 {
		return nil, gameError("growth_unavailable", "官方材料结果暂未能识别，请保留目标后重试。")
	}
	return map[string]any{"view": view}, nil
}
func growthView(game Game, plan GrowthPlan, result QueryResult) View {
	type material struct {
		Name string
		Num  int64
	}
	totals := map[string]material{}
	var collect func(any, int)
	collect = func(raw any, depth int) {
		if depth > 8 {
			return
		}
		switch item := raw.(type) {
		case []any:
			for _, v := range item {
				collect(v, depth+1)
			}
		case map[string]any:
			name := firstText(item, "name", "item_name")
			num, err := strconv.ParseInt(asText(item["num"]), 10, 64)
			if name != "" && err == nil && num > 0 && num < 1e12 {
				id := firstText(item, "id", "item_id")
				if id == "" {
					id = name
				}
				m := totals[id]
				m.Name = plainGameText(name)
				m.Num += num
				totals[id] = m
				return
			}
			for _, v := range item {
				collect(v, depth+1)
			}
		}
	}
	for _, key := range []string{"avatar_consume", "equipment_consume", "skill_consume"} {
		collect(result.Data[key], 0)
	}

	keys := make([]string, 0, len(totals))
	for id := range totals {
		keys = append(keys, id)
	}
	slices.Sort(keys)
	view := View{Title: game.Name + "养成材料 · " + plan.Name, Subtitle: fmt.Sprintf("%s · 角色等级 %d → %d", result.Role.UID, plan.Current, plan.Target), Rows: []Row{}, Note: "来源：官方养成计算器。材料数量按本次目标计算，不执行升级或扣除；不假定背包库存。"}
	for _, id := range keys {
		m := totals[id]
		view.Rows = append(view.Rows, Row{Label: m.Name, Value: strconv.FormatInt(m.Num, 10)})
	}
	if len(keys) == 0 {
		view.Note += " 当前目标没有额外材料需求。"
	}
	return view
}
