package app

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

type PanelStat struct {
	ID    string `json:"id"`
	Key   string `json:"key,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Base  string `json:"base,omitempty"`
	Added string `json:"added,omitempty"`
	// Times is what the service gives of an equipment substat's rolls: the
	// record API's count, which counts the first roll, or the upgrades after
	// it, Mihomo's count less one.
	Times int `json:"times,omitempty"`
}
type PanelEquipment struct {
	Promote    *int            `json:"promote,omitempty"`
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Slot       int             `json:"slot"`
	Level      int             `json:"level"`
	Rarity     string          `json:"rarity"`
	Refinement int             `json:"refinement,omitempty"`
	SetName    string          `json:"set_name,omitempty"`
	Main       []PanelStat     `json:"main"`
	Sub        []PanelStat     `json:"sub"`
	Complete   bool            `json:"complete"`
	Score      *EquipmentScore `json:"score,omitempty"`
	// From is the kept panel a changed panel took the piece from; it is not
	// stored.
	From *CharacterPanel `json:"-"`
}
type PanelSkill struct {
	Kind        string `json:"kind,omitempty"`
	PointType   int    `json:"point_type,omitempty"`
	ID          string `json:"id,omitempty"`
	ExtraLevel  int    `json:"extra_level,omitempty"`
	Name        string `json:"name"`
	Level       int    `json:"level,omitempty"`
	Active      bool   `json:"active"`
	Description string `json:"description,omitempty"`
}
type CharacterPanel struct {
	EquipmentKnown  bool             `json:"equipment_known"`
	WeaponKnown     bool             `json:"weapon_known"`
	Promote         *int             `json:"promote,omitempty"`
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Level           int              `json:"level"`
	Rank            int              `json:"rank"`
	RankKnown       bool             `json:"rank_known"`
	Element         string           `json:"element"`
	Source          string           `json:"source"`
	Stats           []PanelStat      `json:"stats"`
	Weapon          *PanelEquipment  `json:"weapon,omitempty"`
	Equipment       []PanelEquipment `json:"equipment"`
	Skills          []PanelSkill     `json:"skills"`
	Ranks           []PanelSkill     `json:"ranks"`
	ScoreRule       string           `json:"score_rule,omitempty"`
	ScoreNote       string           `json:"score_note,omitempty"`
	TotalScore      *float64         `json:"total_score,omitempty"`
	ScoredEquipment int              `json:"scored_equipment,omitempty"`
	// ScoreDetail is the upstream scoring detail for the panel image; it is
	// not stored with the panel.
	ScoreDetail *ScoreDetail `json:"-"`
	// Official is the official character entry the panel was read from, for
	// images that show fields the panel does not keep. It is not stored.
	Official map[string]any `json:"-"`
	// UpdatedAtMS is when the kept panel was saved; it is not stored with the
	// panel.
	UpdatedAtMS int64 `json:"-"`
	// Relics are the relics as miao keeps them, restored when the panel was
	// kept; nil for a panel not kept.
	Relics *MiaoRelics `json:"relics,omitempty"`
}

var markup = regexp.MustCompile(`<[^>]*>`)

func plainGameText(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "<br>", "\n"), "<br/>", "\n")
	return html.UnescapeString(markup.ReplaceAllString(value, ""))
}

var srStatKeys = map[string]string{"1": "hpPlus", "2": "atkPlus", "3": "defPlus", "4": "speed", "5": "cpct", "6": "cdmg", "7": "heal", "9": "recharge", "10": "effPct", "11": "effDef", "12": "physical", "14": "fire", "16": "ice", "18": "lightning", "20": "wind", "22": "quantum", "24": "imaginary", "27": "hp", "29": "atk", "31": "def", "32": "hpPlus", "33": "atkPlus", "34": "defPlus", "51": "speed", "52": "cpct", "53": "cdmg", "54": "recharge", "55": "heal", "56": "effPct", "57": "effDef", "58": "stance", "59": "stance"}

func panelStats(raw []any, metadata map[string]any, valueKey string) []PanelStat {
	result := []PanelStat{}
	for _, value := range raw {
		item := asObject(value)
		id := asText(item["property_type"])
		display := asText(item[valueKey])
		// A relic below +3 previews the substat it will gain; miao's
		// MysPanelHSR leaves it out.
		if id == "" || display == "" || item["is_preview"] == true {
			continue
		}
		name := firstText(asObject(metadata[id]), "name", "property_name_relic")
		if name == "" {
			name = "属性 " + id
		}
		key := srStatKeys[id]
		// The record API has used both ID families for flat and percent
		// relic stats. The explicit unit keeps both representations correct.
		baseKey := strings.TrimSuffix(key, "Plus")
		if baseKey == "hp" || baseKey == "atk" || baseKey == "def" {
			key = baseKey
			if !strings.HasSuffix(strings.TrimSpace(display), "%") {
				key += "Plus"
			}
		}

		result = append(result, PanelStat{ID: id, Key: key, Name: plainGameText(name), Value: display, Base: asText(item["base"]), Added: asText(item["add"]), Times: number(item["times"])})
	}
	return result
}
func oneStat(value any) []any {
	if item := asObject(value); item != nil {
		return []any{item}
	}
	return nil
}

// NormalizePanels reads the characters of the official avatar info into
// panels. Property names come from its property_info.
func NormalizePanels(result QueryResult, catalog Catalog) []CharacterPanel {
	metadata := asObject(result.Data["property_info"])
	panels := []CharacterPanel{}
	for _, raw := range asList(result.Data["avatar_list"]) {
		item := asObject(raw)
		id := asText(item["id"])
		if id == "" {
			continue
		}
		entry, hasEntry := catalog.Get(id)
		name := asText(item["name"])
		if name == "" && hasEntry {
			name = entry.Name
		}
		if name == "" {
			name = "角色 " + id
		}
		panel := CharacterPanel{ID: id, Name: plainGameText(name), Level: number(item["level"]), Rank: number(item["rank"]), Element: asText(item["element"]), Source: "mihoyo", Stats: panelStats(asList(item["properties"]), metadata, "final"), Equipment: []PanelEquipment{}, Skills: []PanelSkill{}, Ranks: []PanelSkill{}, Official: item}
		if panel.Element == "" && hasEntry {
			panel.Element = entry.Element
		}
		if value, exists := item["promote_level"]; exists {
			n := number(value)
			panel.Promote = &n
		}
		_, panel.RankKnown = item["rank"]
		_, panel.WeaponKnown = item["equip"]
		_, relics := item["relics"]
		_, ornaments := item["ornaments"]
		panel.EquipmentKnown = relics && ornaments
		if weapon := asObject(item["equip"]); weapon != nil && firstText(weapon, "name", "id") != "" {
			w := PanelEquipment{ID: asText(weapon["id"]), Name: plainGameText(asText(weapon["name"])), Level: number(weapon["level"]), Rarity: asText(weapon["rarity"]), Refinement: number(weapon["rank"]), Main: []PanelStat{}, Sub: []PanelStat{}}
			if value, exists := weapon["promote_level"]; exists {
				n := number(value)
				w.Promote = &n
			}
			panel.Weapon = &w
		}
		for _, field := range []string{"relics", "ornaments"} {
			for _, raw := range asList(item[field]) {
				gear := asObject(raw)
				equipment := PanelEquipment{ID: asText(gear["id"]), Name: plainGameText(asText(gear["name"])), Slot: number(gear["pos"]), Level: number(gear["level"]), Rarity: asText(gear["rarity"])}
				equipment.SetName = catalog.ArtifactSets[equipment.ID]
				if equipment.SetName == "" {
					equipment.SetName = catalog.ArtifactSets[equipment.Name]
				}
				equipment.Main = panelStats(oneStat(gear["main_property"]), metadata, "value")
				equipment.Sub = panelStats(asList(gear["properties"]), metadata, "value")
				_, subPresent := gear["properties"]
				equipment.Complete = subPresent && len(equipment.Main) > 0
				panel.Equipment = append(panel.Equipment, equipment)
			}
		}
		// Traces and memosprite skills name themselves in their stages.
		skills := append(asList(item["skills"]), asList(asObject(item["servant_detail"])["servant_skills"])...)
		for _, raw := range skills {
			skill := asObject(raw)
			name, description := "", ""
			for _, raw := range asList(skill["skill_stages"]) {
				stage := asObject(raw)
				if name == "" {
					name = asText(stage["name"])
				}
				if description == "" {
					description = asText(stage["desc"])
				}
			}
			if name == "" {
				name = "行迹 " + asText(skill["point_id"])
			}
			active := true
			if flag, ok := skill["is_activated"].(bool); ok {
				active = flag
			}
			panel.Skills = append(panel.Skills, PanelSkill{Kind: asText(skill["remake"]), PointType: number(skill["point_type"]), ID: asText(skill["point_id"]), ExtraLevel: number(skill["extra_level"]), Name: plainGameText(name), Level: number(skill["level"]), Active: active, Description: plainGameText(description)})
		}
		for _, raw := range asList(item["ranks"]) {
			rank := asObject(raw)
			active, _ := rank["is_unlocked"].(bool)
			panel.Ranks = append(panel.Ranks, PanelSkill{Name: plainGameText(asText(rank["name"])), Active: active, Description: plainGameText(asText(rank["desc"]))})
		}
		panels = append(panels, panel)
	}
	return panels
}

func PanelView(game Game, panels []CharacterPanel, uid string) View {
	v := View{Title: game.Name + "角色面板", Subtitle: uid, Rows: []Row{}, Sections: []Section{}, Note: "属性与装备来自米游社；未返回的数值不作推算。"}
	if len(panels) == 0 {
		v.Note = "此账号未返回所选角色的面板。"
		return v
	}
	if len(panels) > 1 {
		for _, p := range panels {
			v.Rows = append(v.Rows, Row{Label: p.Name, Value: "等级 " + strconv.Itoa(p.Level) + " · 星魂 " + strconv.Itoa(p.Rank)})
		}
		v.Note = "请指定角色名称或 ID 查看完整面板。"
		return v
	}
	p := panels[0]
	v.Title = p.Name + " · 角色面板"
	v.Rows = append(v.Rows, Row{Label: "等级", Value: strconv.Itoa(p.Level)}, Row{Label: "星魂", Value: strconv.Itoa(p.Rank)})
	rows := []Row{}
	for _, stat := range p.Stats {
		rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
	}
	v.Sections = append(v.Sections, Section{Title: "角色属性", Rows: rows})
	if p.Weapon != nil {
		w := p.Weapon
		rows = []Row{{Label: w.Name, Value: "等级 " + strconv.Itoa(w.Level) + " · 叠影 " + strconv.Itoa(w.Refinement)}}
		for _, stat := range append(append([]PanelStat{}, w.Main...), w.Sub...) {
			rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
		}
		v.Sections = append(v.Sections, Section{Title: "光锥", Rows: rows})
	}
	for _, equipment := range p.Equipment {
		rows = []Row{{Label: "等级", Value: strconv.Itoa(equipment.Level)}}
		for _, stat := range equipment.Main {
			rows = append(rows, Row{Label: "主属性 · " + stat.Name, Value: stat.Value})
		}
		for _, stat := range equipment.Sub {
			rows = append(rows, Row{Label: stat.Name, Value: stat.Value})
		}
		if equipment.Score != nil {
			rows = append(rows, Row{Label: "评分", Value: strconv.FormatFloat(equipment.Score.Value, 'f', 1, 64) + " · " + equipment.Score.Grade})
		}
		v.Sections = append(v.Sections, Section{Title: strconv.Itoa(equipment.Slot) + "号位 · " + equipment.Name, Rows: rows})
	}
	rows = []Row{}
	for _, skill := range p.Skills {
		status := "未解锁"
		if skill.Active {
			status = "等级 " + strconv.Itoa(skill.Level)
		}
		rows = append(rows, Row{Label: skill.Name, Value: status})
	}
	if len(rows) > 0 {
		v.Sections = append(v.Sections, Section{Title: "技能与行迹", Rows: rows})
	}
	rows = []Row{}
	for _, rank := range p.Ranks {
		status := "未解锁"
		if rank.Active {
			status = "已解锁"
		}
		rows = append(rows, Row{Label: rank.Name, Value: status})
	}
	if len(rows) > 0 {
		v.Sections = append(v.Sections, Section{Title: "星魂", Rows: rows})
	}
	if p.TotalScore != nil {
		v.Rows = append(v.Rows, Row{Label: "装备评分合计", Value: strconv.FormatFloat(*p.TotalScore, 'f', 1, 64) + " · 已评分 " + strconv.Itoa(p.ScoredEquipment) + " / " + strconv.Itoa(len(p.Equipment)) + " 件"})
	}
	if p.ScoreRule != "" {
		v.Note += "\n评分规则：" + p.ScoreRule + "。"
	}
	if p.ScoreNote != "" {
		v.Note += "\n" + p.ScoreNote
	}
	return v
}
