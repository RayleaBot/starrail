package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

type BuildWeapon struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Level      int    `json:"level"`
	Promote    *int   `json:"promote"`
	Refinement int    `json:"refinement"`
}
type BuildProfile struct {
	Conditions         *BuildConditions   `json:"conditions,omitempty"`
	CandidateEquipment *[]BuildGear       `json:"candidate_equipment,omitempty"`
	Level              int                `json:"level"`
	Promote            *int               `json:"promote"`
	Rank               int                `json:"rank"`
	Talents            map[string]int     `json:"talents"`
	Trees              []string           `json:"trees"`
	Attributes         map[string]float64 `json:"attributes"`
	Weapon             BuildWeapon        `json:"weapon"`
	Equipment          []BuildGear        `json:"equipment"`
	EnemyLevel         int                `json:"enemy_level"`
	CandidateWeapon    *BuildWeapon       `json:"candidate_weapon,omitempty"`
	// DmgIndex asks for miao's 伤害 mode: the detail numbered from 1, or 0
	// for the rule's default.
	DmgIndex *int `json:"dmg_index,omitempty"`
}
type BuildStat struct {
	ID      string  `json:"id,omitempty"`
	Percent bool    `json:"percent,omitempty"`
	Key     string  `json:"key"`
	Value   float64 `json:"value"`
	Times   int     `json:"times,omitempty"`
}
type BuildGear struct {
	Slot    int         `json:"slot,omitempty"`
	SetID   string      `json:"set_id,omitempty"`
	SetName string      `json:"set_name"`
	Main    BuildStat   `json:"main"`
	Sub     []BuildStat `json:"sub"`
}
type BuildResult struct {
	Conditions    *BuildConditions      `json:"conditions,omitempty"`
	EquipmentFrom *BuildEquipmentSource `json:"equipment_from,omitempty"`
	Variant       string                `json:"variant,omitempty"`
	Source        string                `json:"source"`
	Version       string                `json:"version"`
	CharacterID   string                `json:"character_id"`
	Character     string                `json:"character"`
	EnemyLevel    int                   `json:"enemy_level"`
	Baseline      BuildScenario         `json:"baseline"`
	Candidate     *BuildScenario        `json:"candidate"`
}
type BuildEquipmentSource struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
}
type BuildScenario struct {
	Weapon     BuildWeapon        `json:"weapon"`
	Attributes map[string]float64 `json:"attributes"`
	Results    []BuildSkillResult `json:"results"`
	// EnemyName and CreatedBy are the rule's enemy and author, as miao's
	// damage table names them.
	EnemyName string `json:"enemy_name,omitempty"`
	CreatedBy string `json:"created_by,omitempty"`
	// Matrix is miao's 词条伤害计算 in 伤害 mode.
	Matrix *DamageMatrix `json:"matrix,omitempty"`
}

// DamageMatrix is the damage of one detail with one substat of each row's
// attribute traded for one of each column's.
type DamageMatrix struct {
	Index int            `json:"index"`
	Title string         `json:"title"`
	Avg   float64        `json:"avg"`
	Dmg   *float64       `json:"dmg"`
	Attrs []DamageAttr   `json:"attrs"`
	Rows  [][]DamageCell `json:"rows"`
}

type DamageAttr struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// DamageCell is one trade; Type is na on the diagonal, else avg, gt or lt
// against the detail's own average.
type DamageCell struct {
	Type string   `json:"type"`
	Avg  float64  `json:"avg"`
	Dmg  *float64 `json:"dmg"`
}
type BuildSkillResult struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Expected *float64 `json:"expected"`
	Text     string   `json:"text,omitempty"`
	// Default marks the detail upstream ranks the group by.
	Default  bool     `json:"default,omitempty"`
	Critical *float64 `json:"critical"`
	Buffs    []string `json:"buffs"`
	Kind     string   `json:"kind"`
}
type BuildWeaponChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// findBuildCharacter returns the reference character only when it has a damage
// rule; some characters are catalogued for scoring alone.
func findBuildCharacter(engine *reference.Engine, panel CharacterPanel) (reference.Character, error) {
	record, err := findReferenceCharacter(engine, panel)
	if err == nil && record.Script == "" {
		return reference.Character{}, buildUnavailable("character")
	}
	return record, err
}

func findReferenceCharacter(engine *reference.Engine, panel CharacterPanel) (reference.Character, error) {
	metadata := engine.Metadata()
	wanted := canonicalBuildElement(panel.Element)
	ruleID := referenceCharacterID(panel)
	for _, record := range metadata.Characters {
		match := record.ID == ruleID
		if match && canonicalBuildElement(record.Element) == wanted {
			return record, nil
		}
	}
	if ruleID != panel.ID {
		err := gameError("build_unavailable", "固定参考尚无此强化形态的自动计算规则，可使用手动伤害试算。")
		err.Details = map[string]any{"reason": "enhanced_rule_missing"}
		return reference.Character{}, err
	}
	return reference.Character{}, buildUnavailable("character")
}

// Pinned MysPanelHSRApi selects enhanced rules from the actual skill prefix,
// while account queries continue to use the official, original character ID.
func referenceCharacterID(panel CharacterPanel) string {
	if !slices.Contains([]string{"1212", "1205", "1005", "1006", "1004", "1102", "1217", "1310", "1306", "1307"}, panel.ID) {
		return panel.ID
	}
	for _, skill := range panel.Skills {
		if strings.HasPrefix(skill.ID, "1"+panel.ID) {
			return "2" + panel.ID[1:]
		}
	}
	return panel.ID
}
func canonicalBuildElement(value string) string {
	value = normalizedElement(value)
	if mapped := map[string]string{"fire": "pyro", "ice": "cryo", "lightning": "electro", "wind": "anemo", "雷": "electro"}[value]; mapped != "" {
		return mapped
	}
	return value
}
func buildUnavailable(reason string) error {
	err := gameError("build_unavailable", "当前面板缺少自动计算所需的数据，或固定参考尚未支持此角色。")
	err.Details = map[string]any{"reason": reason}
	return err
}
func buildStatNumber(raw string) (float64, bool) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	raw = strings.TrimSuffix(raw, "%")
	value, err := strconv.ParseFloat(raw, 64)
	return value, err == nil && !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= 1e8
}
func buildProfile(panel CharacterPanel, record reference.Character) (BuildProfile, error) {
	if !panel.RankKnown || panel.Level < 1 || panel.Rank < 0 || panel.Rank > 6 || (panel.Weapon == nil && !panel.WeaponKnown) {
		return BuildProfile{}, buildUnavailable("identity")
	}
	maxLevel := 80
	if panel.Level > maxLevel {
		return BuildProfile{}, buildUnavailable("level")
	}
	w := panel.Weapon
	if w == nil {
		zero := 0
		w = &PanelEquipment{Promote: &zero, Refinement: 1}
	}
	p := BuildProfile{Level: panel.Level, Promote: panel.Promote, Rank: panel.Rank, Weapon: BuildWeapon{ID: w.ID, Name: w.Name, Level: w.Level, Promote: w.Promote, Refinement: w.Refinement}, Talents: map[string]int{}, Trees: []string{}, Attributes: map[string]float64{}, Equipment: []BuildGear{}, EnemyLevel: 103}
	// miao profile-detail passes enemy level 80 for Star Rail.
	p.EnemyLevel = 80

	if w.ID != "" && (w.Level < 1 || w.Level > 90 || w.Refinement < 1 || w.Refinement > 5) {
		return p, buildUnavailable("weapon")
	}
	keys := map[string]string{"1": "hp", "2": "atk", "3": "def", "4": "speed", "5": "cpct", "6": "cdmg", "7": "heal", "9": "recharge", "10": "effPct", "11": "effDef", "58": "stance"}
	if value, ok := elementDamageBonus(panel); ok {
		p.Attributes["dmg"] = value
	}
	for _, stat := range panel.Stats {
		key := keys[stat.ID]
		if key == "" {
			continue
		}
		if value, ok := buildStatNumber(stat.Value); ok {
			p.Attributes[key] = value
		}
		if slices.Contains([]string{"hp", "atk", "def", "speed"}, key) {
			if value, ok := buildStatNumber(stat.Base); ok {
				p.Attributes[key+"Base"] = value
			}
		}
	}
	for _, key := range []string{"hp", "atk", "def", "hpBase", "atkBase", "defBase", "cpct", "cdmg", "dmg"} {
		if _, ok := p.Attributes[key]; !ok {
			return p, buildUnavailable("attributes." + key)
		}
	}
	talents := asObject(record.Data["talent"])
	for key, talent := range PanelTalents(panel, record) {
		p.Talents[key] = talent.Level
	}
	for _, skill := range panel.Skills {
		if skill.Active && skill.PointType != 2 && skill.ID != "" {
			p.Trees = append(p.Trees, RecordSkillID(panel, record, skill))
		}
	}

	for key, raw := range talents {
		if !slices.Contains([]string{"a", "e", "q", "t", "me", "mt", "xe"}, key) {
			continue
		}
		if len(asObject(raw)) > 0 && p.Talents[key] == 0 {
			return p, buildUnavailable("talent." + key)
		}
	}
	var err error
	p.Equipment, err = buildGearSet(panel)
	return p, err
}

// damageIndexRange is the runner's answer to a damage number past the rule's
// details.
var damageIndexRange = regexp.MustCompile(`dmg\.index_range\.([0-9]+)`)

func referenceBuild(ctx context.Context, engine *reference.Engine, record reference.Character, input BuildProfile) (BuildResult, error) {
	var raw map[string]any
	if decodeObject(input, &raw) != nil {
		return BuildResult{}, buildUnavailable("input")
	}
	result, err := engine.Run(ctx, record, raw)
	if err != nil {
		if match := damageIndexRange.FindStringSubmatch(err.Error()); match != nil {
			return BuildResult{}, gameError("build_input_invalid", "序号输入错误："+record.Name+"最多只支持"+match[1]+"种伤害计算哦")
		}
		return BuildResult{}, buildUnavailable("reference_calculation")
	}
	var output BuildResult
	if json.Unmarshal(result, &output) != nil {
		return output, buildUnavailable("result")
	}
	return output, nil
}
func (a *App) buildAction(ctx context.Context, client AccountsClient, action string, input map[string]any) (map[string]any, error) {
	panel, err := a.queryCharacterPanel(ctx, client, Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}, asText(input["character_id"]))
	if err != nil {
		return nil, err
	}
	record, err := findBuildCharacter(a.Game.Calc, panel)
	if err != nil {
		return nil, err
	}
	profile, err := buildProfile(panel, record)
	if err != nil {
		return nil, err
	}
	if action == "build.prepare" {
		calculated, calcErr := referenceBuild(ctx, a.Game.Calc, record, profile)
		if calcErr != nil {
			return nil, calcErr
		}
		applyBuildIdentity(&calculated, panel, record)
		metadata := a.Game.Calc.Metadata()
		choices := []BuildWeaponChoice{}
		for _, weapon := range metadata.Weapons {
			if weapon.Game == record.Game && weapon.Type == record.WeaponType {
				choices = append(choices, BuildWeaponChoice{ID: weapon.ID, Name: weapon.Name})
			}
		}
		slices.SortFunc(choices, func(a, b BuildWeaponChoice) int { return strings.Compare(a.Name, b.Name) })
		return map[string]any{"character": panel.Name, "weapon": calculated.Baseline.Weapon, "weapons": choices, "talents": profile.Talents, "enemy_level": profile.EnemyLevel, "version": calculated.Version, "build": calculated}, nil
	}
	if action != "build.compare" {
		return nil, gameError("operation_denied", "操作不存在。")
	}
	if raw, exists := input["conditions"]; exists {
		profile.Conditions, err = a.validateConditions(panel.ID, raw)
		if err != nil {
			return nil, err
		}
	}
	if value, ok := input["enemy_level"]; ok {
		var level float64
		if decodeObject(value, &level) != nil || level < 1 || level > 200 || math.Trunc(level) != level {
			return nil, gameError("build_input_invalid", "敌人等级应为 1–200 的整数。")
		}
		profile.EnemyLevel = int(level)
	}
	if value, exists := input["candidate_weapon"]; exists {
		var w BuildWeapon
		maxPromote, maxLevel := 6, 90
		if value == nil || decodeObject(value, &w) != nil || w.Promote == nil || *w.Promote < 0 || *w.Promote > maxPromote || w.Refinement < 1 || w.Refinement > 5 || (w.ID != "" && (w.Level < 1 || w.Level > maxLevel)) || (w.ID == "" && (w.Level != 0 || *w.Promote != 0 || w.Refinement != 1)) {
			return nil, gameError("build_input_invalid", "请填写有效的光锥、等级、突破阶段和叠影。")
		}
		profile.CandidateWeapon = &w
		if w.ID != "" {
			metadata := a.Game.Calc.Metadata()
			valid := false
			for _, weapon := range metadata.Weapons {
				if weapon.ID == w.ID && weapon.Type == record.WeaponType {
					valid = true
					break
				}
			}
			steps := []int{1, 20, 30, 40, 50, 60, 70, 80}
			if !valid || w.Level < steps[*w.Promote] || w.Level > steps[*w.Promote+1] {
				return nil, gameError("build_input_invalid", "候选武器类型或突破阶段与所选角色、等级不符。")
			}
		}
	}
	var equipmentSource *BuildEquipmentSource
	if value, exists := input["equipment_from_character_id"]; exists {
		id := asText(value)
		number, err := strconv.Atoi(id)
		if err != nil || number <= 0 || number > 1000000000 {
			return nil, gameError("build_input_invalid", "请选择有效的装备来源角色。")
		}
		source := panel
		if id != panel.ID {
			source, err = a.queryCharacterPanel(ctx, client, Selection{AccountRef: asText(input["account_ref"]), RoleRef: asText(input["role_ref"])}, id)
			if err != nil {
				return nil, err
			}
		}
		gear, err := buildGearSet(source)
		if err != nil {
			return nil, err
		}
		profile.CandidateEquipment = &gear
		equipmentSource = &BuildEquipmentSource{CharacterID: source.ID, Name: source.Name}
	}
	result, err := referenceBuild(ctx, a.Game.Calc, record, profile)
	if err != nil {
		return nil, err
	}
	applyBuildIdentity(&result, panel, record)
	result.EquipmentFrom = equipmentSource
	result.Conditions = profile.Conditions
	return map[string]any{"build": result}, nil
}
func applyBuildIdentity(result *BuildResult, panel CharacterPanel, record reference.Character) {
	result.CharacterID = panel.ID
	result.Character = panel.Name
	if record.ID != panel.ID {
		result.Variant = "enhanced"
		result.Character += "（强化形态）"
	}
}

// panelDamage calculates the reference damage of a panel already read from the
// account, so a panel reply does not query it twice.
func (a *App) panelDamage(ctx context.Context, panel CharacterPanel, dmgIndex *int) (BuildResult, error) {
	record, err := findBuildCharacter(a.Game.Calc, panel)
	if err != nil {
		return BuildResult{}, err
	}
	profile, err := buildProfile(panel, record)
	if err != nil {
		return BuildResult{}, err
	}
	profile.DmgIndex = dmgIndex
	result, err := referenceBuild(ctx, a.Game.Calc, record, profile)
	if err != nil {
		return BuildResult{}, err
	}
	applyBuildIdentity(&result, panel, record)
	return result, nil
}

// fullPanelView answers a single-character panel the way upstream's
// <角色>面板 does: attributes and equipment with their scores, then the
// reference damage. A part that cannot be calculated is named in the note
// instead of failing the whole reply. When record is set, a panel viewed in a
// group also enters the group ranking; change is the 面板换装 word of a
// changed panel.
func (a *App) fullPanelView(ctx context.Context, event *rayleabot.EventContext, panel CharacterPanel, uid string, record bool, change string) View {
	return a.panelView(ctx, event, panel, uid, record, change, nil)
}

// panelView is fullPanelView, in miao's 伤害 mode when damage, calculated in
// that mode, is given: the damage table numbered, the chosen detail's
// substat trades and its buffs in place of the relics.
func (a *App) panelView(ctx context.Context, event *rayleabot.EventContext, panel CharacterPanel, uid string, record bool, change string, damage *BuildResult) View {
	missing := []string{}
	if scored, err := a.scorePanel(ctx, panel); err == nil {
		panel = scored
	} else {
		missing = append(missing, "评分："+friendlyError(err))
	}
	view := PanelView(a.Game, []CharacterPanel{panel}, uid)
	image := PanelImage{Panel: panel, UID: uid, Change: change, DamageMode: damage != nil}
	if change != "" {
		view.Note = "该面板为非实际数据。当前替换命令：" + change
	}
	result, err := BuildResult{}, error(nil)
	if damage != nil {
		result = *damage
	} else {
		result, err = a.panelDamage(ctx, panel, nil)
	}
	if err == nil {
		damage := BuildView(a.Game, result)
		view.Sections = append(view.Sections, Section{Title: "参考伤害 · " + result.Version, Rows: damage.Rows})
		image.Damage = &result
	} else {
		missing = append(missing, "伤害："+friendlyError(err))
	}
	if record {
		a.recordRank(event, uid, panel, image.Damage)
	}
	if len(missing) > 0 {
		view.Note += "\n" + strings.Join(missing, "\n")
	}
	if a.panel != nil {
		if a.Game.Calc != nil {
			image.Record, _ = findReferenceCharacter(a.Game.Calc, panel)
		}
		// As miao, 原图 then answers with the picture the panel drew.
		var ref string
		image.Splash, ref = a.panelSplash(panel, image.Record.Name)
		if drawn, ok := a.panel(a.imageContext(ctx), image); ok {
			view.Image = &drawn
			_ = a.rememberImage(event, ref)
		}
	}
	return view
}

func BuildView(game Game, result BuildResult) View {
	rows := []Row{}
	for _, item := range result.Baseline.Results {
		value := item.Text
		if item.Expected != nil {
			value = fmt.Sprintf("期望 %.1f", *item.Expected)
		}
		if item.Critical != nil {
			value += fmt.Sprintf(" · 暴击 %.1f", *item.Critical)
		}
		rows = append(rows, Row{Label: item.Title, Value: value})
	}
	return View{Title: result.Character + " · 参考伤害", Subtitle: game.Name + " · " + result.Version, Rows: rows, Note: "按固定参考列出的战斗情境自动应用角色、武器和套装增益。"}
}

// elementDamageBonus reads the damage bonus of the character's own element from
// the official panel. Only displayed percentages are accepted, so a fractional
// number from an unknown endpoint revision is never mistaken for one.
func elementDamageBonus(panel CharacterPanel) (float64, bool) {
	element := normalizedElement(panel.Element)
	if mapped := map[string]string{"fire": "pyro", "ice": "cryo", "lightning": "electro", "wind": "anemo"}[element]; mapped != "" {
		element = mapped
	}
	statID := map[string]string{"physical": "12", "pyro": "14", "cryo": "16", "electro": "18", "anemo": "20", "quantum": "22", "imaginary": "24"}[element]
	for _, stat := range panel.Stats {
		if statID == "" || stat.ID != statID {
			continue
		}
		raw := strings.ReplaceAll(strings.TrimSpace(stat.Value), ",", "")
		if !strings.HasSuffix(raw, "%") {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSuffix(raw, "%"), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		return value, true
	}
	return 0, false
}

// PanelTalent is a skill level as miao's panel shows it: Level includes
// constellation or eidolon bonuses, Original does not.
type PanelTalent struct {
	Level    int
	Original int
}

// PanelTalents maps the panel's skills to miao talent keys (a, e and q, plus
// t, me, mt and xe in Star Rail) with the calculation record.
func PanelTalents(panel CharacterPanel, record reference.Character) map[string]PanelTalent {
	result := map[string]PanelTalent{}
	talents := asObject(record.Data["talent"])
	talentIDs := asObject(record.Data["talentId"])
	skillKinds := map[string]string{"普攻": "a", "战技": "e", "终结技": "q", "天赋": "t", "秘技": "z", "欢愉技": "xe", "忆灵技": "me", "忆灵天赋": "mt"}
	for _, skill := range panel.Skills {
		key := asText(talentIDs[RecordSkillID(panel, record, skill)])
		if skillKinds[skill.Kind] != "" {
			key = skillKinds[skill.Kind]
		}
		if key == "" {
			for k, raw := range talents {
				if firstText(asObject(raw), "name") == skill.Name {
					key = strings.TrimRight(k, "123")
					break
				}
			}
		}
		// The official level already includes constellation/eidolon bonuses.
		if key != "" && skill.Level > 0 && skill.Level <= 20 {
			result[key] = PanelTalent{Level: skill.Level, Original: skill.Level - skill.ExtraLevel}
		}
	}
	return result
}

// RecordSkillID translates a Star Rail skill ID of a character variant (such
// as the Trailblazer paths) to the record's own IDs.
func RecordSkillID(panel CharacterPanel, record reference.Character, skill PanelSkill) string {
	skillID := skill.ID
	if record.ID == panel.ID {
		return skillID
	}
	if strings.HasPrefix(skillID, "1"+panel.ID) {
		skillID = record.ID + strings.TrimPrefix(skillID, "1"+panel.ID)
	}
	traceData, traceDetails := asObject(record.Data["tree"]), asObject(record.Data["treeData"])
	for _, candidate := range []string{skill.ID, skillID, "1" + skillID} {
		if traceData[candidate] != nil || traceDetails[candidate] != nil {
			return candidate
		}
	}
	return skillID
}

func buildGearSet(panel CharacterPanel) ([]BuildGear, error) {
	if !panel.EquipmentKnown {
		return nil, buildUnavailable("equipment.missing")
	}
	out := []BuildGear{}
	slots := map[int]bool{}
	for _, gear := range panel.Equipment {
		if !gear.Complete || len(gear.Main) != 1 || gear.Slot < 1 || gear.Slot > 6 || slots[gear.Slot] || gear.SetName == "" {
			return nil, buildUnavailable("equipment")
		}
		slots[gear.Slot] = true
		piece := BuildGear{Slot: gear.Slot, SetName: gear.SetName, Sub: []BuildStat{}}
		for i, stat := range append(append([]PanelStat{}, gear.Main...), gear.Sub...) {
			value, ok := buildStatNumber(stat.Value)
			if !ok || stat.ID == "" || stat.Key == "" {
				return nil, buildUnavailable("equipment.stat")
			}
			s := BuildStat{ID: stat.ID, Key: stat.Key, Value: value, Percent: strings.HasSuffix(strings.TrimSpace(stat.Value), "%")}
			if i == 0 {
				piece.Main = s
			} else {
				piece.Sub = append(piece.Sub, s)
			}
		}
		out = append(out, piece)
	}
	return out, nil
}
