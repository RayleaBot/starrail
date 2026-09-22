package app

import (
	"slices"
	"strconv"
)

func cloudIDs(values []string, maxCount int, uid bool) bool {
	if len(values) < 1 || len(values) > maxCount {
		return false
	}
	seen := map[string]bool{}
	for _, v := range values {
		if seen[v] {
			return false
		}
		seen[v] = true
		if uid {
			if !uidPattern.MatchString(v) {
				return false
			}
		} else {
			n, e := strconv.Atoi(v)
			if e != nil || n < 1 || n > 1000000000 {
				return false
			}
		}
	}
	return true
}
func extendedCloudRequest(input CloudInput) (string, map[string]any, bool, error) {
	body := map[string]any{"version": "0.1.0"}
	invalid := func() (string, map[string]any, bool, error) {
		return "", nil, true, gameError("input_invalid", "云查询需要有效的 UID、角色列表或版本，且列表不能重复。")
	}
	switch input.Mode {
	case "panel_refresh":
		if !uidPattern.MatchString(input.UID) {
			return invalid()
		}
		body["uid"] = input.UID
		body["type"] = arkGame
		return "panel/refresh", body, true, nil
	case "self_rank":
		if !uidPattern.MatchString(input.UID) || !cloudIDs(input.CharacterIDs, 250, false) {
			return invalid()
		}
		body["uid"] = input.UID
		body["ids"] = input.CharacterIDs
		body["type"] = arkGame
		return "rank/self", body, true, nil
	case "group_rank":
		if !cloudIDs(input.UIDs, 50, true) || !cloudIDs([]string{input.CharacterID}, 1, false) || !slices.Contains([]string{"", "dmg", "mark"}, input.Query) {
			return invalid()
		}
		body["uids"] = input.UIDs
		body["id"], _ = strconv.Atoi(input.CharacterID)
		body["update"] = 2
		body["query"] = input.Query
		if input.Query == "" {
			body["query"] = "dmg"
		}
		body["data"] = nil
		return "rank/group", body, true, nil
	}
	return "", nil, false, nil
}
func rankArrayResult(game Game, input CloudInput, list []any) (CloudResult, error) {
	if len(list) < 1 || len(list) > 2 {
		return CloudResult{}, gameError("cloud_invalid", "云排名分组数量暂不兼容。")
	}
	v := View{Title: game.Name + "云排名", Subtitle: "UID " + input.UID, Rows: []Row{}, Note: "来源：ark.ivny.cn。伤害与装备的名次、百分位和分数分别展示。"}
	if len(input.localPanel) > 0 {
		v.Subtitle = "导入面板 · 角色 " + input.CharacterID
		v.Note = "来源：ark.ivny.cn；根据明确上传的本地面板分别计算伤害与装备名次，不代表本人官方实时成绩。"
	}
	for i, raw := range list {
		item := asObject(raw)
		if code := asText(item["retcode"]); code != "100" && code != "0" {
			return CloudResult{}, gameError("cloud_rejected", "云服务没有提供完整排名结果。")
		}
		title := "伤害排名"
		if input.Query == "mark" || i == 1 {
			title = "装备评分排名"
		}
		v.Sections = append(v.Sections, Section{Title: title, Rows: cloudNumericRows(item, "rank", "服务排名", "percent", "百分位（%）", "score", "评分", "sum", "收录总量")})
	}
	return CloudResult{View: &v}, nil
}
func extendedCloudResult(game Game, input CloudInput, result map[string]any) (CloudResult, bool, error) {
	if input.Mode == "panel_refresh" {
		view := View{Title: game.Name + "云面板更新请求已接受", Subtitle: "UID " + input.UID, Rows: []Row{}, Note: "来源：ark；可稍后重新查询云面板。"}
		return CloudResult{View: &view}, true, nil
	}
	if !slices.Contains([]string{"self_rank", "group_rank"}, input.Mode) {
		return CloudResult{}, false, nil
	}
	list := asList(result["rank"])
	ids := input.UIDs
	if input.Mode == "self_rank" {
		ids = input.CharacterIDs
	}
	if len(list) != len(ids) {
		return CloudResult{}, true, gameError("cloud_invalid", "云结果与请求列表无法一一对应。")
	}
	title := map[string]string{"self_rank": "多角色全服排名", "group_rank": "多人角色云排名"}[input.Mode]
	v := View{Title: game.Name + title, Rows: []Row{}, Note: "来源：ark.ivny.cn；只展示请求列表的服务收录数据，不合并不同赛季或算法，不使用米游社 CK。"}
	for i, raw := range list {
		item := asObject(raw)
		label := "UID " + ids[i]
		if input.Mode == "self_rank" {
			label = "角色 " + ids[i]
		}
		rows := cloudNumericRows(item, "rank", "服务排名", "percent", "百分位（%）", "score", "评分", "sum", "收录总量")
		section := Section{Title: label, Rows: rows}
		if len(rows) == 0 {
			section.Text = "服务未提供此项数据。"
		}
		v.Sections = append(v.Sections, section)
	}
	return CloudResult{View: &v}, true, nil
}
