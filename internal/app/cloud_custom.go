package app

import (
	"context"
	"errors"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/reference"
)

// ark-plugin's custom ranking (apps/customRank.js): ark<角色>[伤害|圣遗物]
// (排行|排名) lists the best panels ark keeps for a character, ordered by
// damage or relic score and narrowed by the filters after it, drawn as miao's
// character/rank-profile-list with ark's notes. The image is remembered for
// five minutes, so ark获取面板N can open an entry's panel. Upstream writes
// relics as Genshin's 圣遗物 in Star Rail too; the replies write 遗器, as Star
// Rail calls them, and the command and column names take both.

// customRankColumns are the columns ark filters by (ALLOWED_COLS). Anyone
// may filter by Eidolon; the rest need a super administrator and the ark
// token.
var customRankColumns = []string{"level", "promote", "cons", "talent_a", "talent_e", "talent_q", "weapon_level", "weapon_promote", "weapon_affix", "dmg_avg", "mark_score", "data_time", "artis_sets", "pos1", "pos2", "pos3", "pos4", "pos5", "pos6"}

// customRankAliases are ark's Chinese names of the columns (ALIAS_MAP), with
// 遗器 beside its 圣遗物.
var customRankAliases = map[string]string{"等级": "level", "突破": "promote", "命座": "cons", "命": "cons", "普攻": "talent_a", "战技": "talent_e", "爆发": "talent_q",
	"武器等级": "weapon_level", "武器突破": "weapon_promote", "精炼": "weapon_affix", "伤害": "dmg_avg", "评分": "mark_score", "伤害均值": "dmg_avg", "圣遗物评分": "mark_score", "圣遗物": "artis_sets", "遗器评分": "mark_score", "遗器": "artis_sets", "套装": "artis_sets"}

// customRankOperators are ark's comparison operators in the order of the rule
// numbers it receives (OP_TO_RULE, RULE_LABEL).
var customRankOperators = []string{">=", "=", "<=", ">", "<", "!="}

// customRankMainStats are the main stats of each relic slot in the order of
// the IDs ark filters them by (srMainIdx), and customRankStatNames the names
// ark reads for them (srStatNameMap); other words are taken as the stat's key.
var (
	customRankMainStats = map[string][]string{"1": {"hpPlus"}, "2": {"atkPlus"}, "3": {"hp", "atk", "def", "cpct", "cdmg", "heal", "effPct"}, "4": {"hp", "atk", "def", "speed"},
		"5": {"hp", "atk", "def", "phy", "fire", "ice", "elec", "wind", "quantum", "imaginary"}, "6": {"be", "recharge", "hp", "atk", "def"}}
	customRankStatNames = map[string]string{"生命": "hp", "大生命": "hp", "生命值": "hp", "攻击": "atk", "大攻击": "atk", "攻击力": "atk", "防御": "def", "大防御": "def", "防御力": "def", "速度": "speed",
		"暴击": "cpct", "暴击率": "cpct", "爆伤": "cdmg", "暴伤": "cdmg", "暴击伤害": "cdmg", "治疗": "heal", "治疗加成": "heal", "命中": "effPct", "效果命中": "effPct",
		"物伤": "phy", "物理": "phy", "物理属性伤害提高": "phy", "火伤": "fire", "火": "fire", "火属性伤害提高": "fire", "冰伤": "ice", "冰": "ice", "冰属性伤害提高": "ice",
		"雷伤": "elec", "雷": "elec", "雷属性伤害提高": "elec", "风伤": "wind", "风": "wind", "风属性伤害提高": "wind", "量子伤": "quantum", "量子": "quantum", "量子属性伤害提高": "quantum",
		"虚数伤": "imaginary", "虚数": "imaginary", "虚数属性伤害提高": "imaginary", "击破": "be", "击破特攻": "be", "充能": "recharge", "能量回复效率": "recharge", "充能效率": "recharge",
		"小生命": "hpPlus", "小攻击": "atkPlus"}
)

var (
	customRankWord      = regexp.MustCompile(`^ark(.+?)(伤害|圣遗物|遗器)?(排行|排名)(?:\s+(.+))?$`)
	customRankCount     = regexp.MustCompile(`(?i)^(?:nums|数量)[:：=](\d+)$`)
	customRankSort      = regexp.MustCompile(`(?i)^(?:(?:sort|排序)[:：]|(?:升序|降序|asc|desc)$)`)
	customRankCondition = regexp.MustCompile(`^([\w\x{4e00}-\x{9fff}]+)(>=|<=|!=|>|<|=)(.+)$`)
	customRankSlot      = regexp.MustCompile(`(?i)^pos[1-6]$`)
	customRankSlotName  = regexp.MustCompile(`^部位[1-6]$`)
)

// customRankFilter is one filter as ark receives it.
type customRankFilter struct {
	Type     string `json:"type"`
	Rule     int    `json:"rule"`
	Value    any    `json:"value"`
	RawValue string `json:"rawValue"`
}

// customRankQuery is the data of ark's rank/custom request: the column it
// orders by, the filters and how many rows to return.
type customRankQuery struct {
	Rank struct {
		Col   string `json:"col"`
		Order string `json:"order"`
	} `json:"rank"`
	Filter []customRankFilter `json:"filter"`
	Nums   int                `json:"nums"`
}

// customRankColumn is ark's resolveCol: pos1–pos6 or 部位1–部位6 name the
// relic slots, a Chinese name its column, anything else itself in lower case.
func customRankColumn(name string) string {
	switch {
	case customRankSlot.MatchString(name):
		return strings.ToLower(name)
	case customRankSlotName.MatchString(name):
		return "pos" + strings.TrimPrefix(name, "部位")
	case customRankAliases[name] != "":
		return customRankAliases[name]
	}
	return strings.ToLower(name)
}

// parseCustomRank reads the words of a custom ranking as ark's parseRankArgs
// does: nums:N (or 数量:N) asks for 1–50 rows, 20 by default; a column, an
// operator and a value add a filter; the other words, joined, name the
// character. Only the default descending order is taken. A column outside
// columns fails with denied, or with ark's own reply when denied is empty.
func parseCustomRank(raw string, columns []string, sort, denied string) (string, customRankQuery, error) {
	query := customRankQuery{Filter: []customRankFilter{}, Nums: 20}
	query.Rank.Col, query.Rank.Order = sort, "desc"
	name := ""
	for _, token := range strings.Fields(raw) {
		if match := customRankCount.FindStringSubmatch(token); match != nil {
			count, err := strconv.Atoi(match[1])
			if err != nil {
				// Only digits fail by overflowing, which ark caps at 50.
				count = 50
			}
			query.Nums = min(max(count, 1), 50)
			continue
		}
		if customRankSort.MatchString(token) {
			return "", query, errors.New("当前命令仅支持默认降序排序")
		}
		match := customRankCondition.FindStringSubmatch(token)
		if match == nil {
			name += token
			continue
		}
		column := customRankColumn(match[1])
		var value any = match[3]
		if !strings.HasPrefix(column, "pos") {
			// Slots keep their stat name; sets may be named or numbered.
			number, err := strconv.ParseFloat(match[3], 64)
			if err == nil && !math.IsNaN(number) && !math.IsInf(number, 0) {
				value = number
			} else if column != "artis_sets" {
				return "", query, errors.New(match[1] + " 的值必须是数字")
			}
		}
		if !slices.Contains(columns, column) {
			return "", query, errors.New(cmpOr(denied, "不支持筛选列 "+match[1]))
		}
		query.Filter = append(query.Filter, customRankFilter{Type: column, Rule: slices.Index(customRankOperators, match[2]), Value: value, RawValue: match[3]})
	}
	return name, query, nil
}

// customRankMainStat is the ID ark filters a slot's main stat by, read from
// the stat's name as written, and whether the slot has that main stat.
func customRankMainStat(slot, name string) (int, bool) {
	index := slices.Index(customRankMainStats[slot], cmpOr(customRankStatNames[name], name))
	return index + 1, index >= 0
}

// customRankLimit is the note ark writes in the ranking's 排名参与条件 line:
// the order and each filter as sent.
func customRankLimit(query customRankQuery) string {
	filters := []string{}
	for _, filter := range query.Filter {
		label := filter.Type
		if slot, ok := strings.CutPrefix(label, "pos"); ok {
			label = "部位" + slot
		}
		filters = append(filters, label+customRankOperators[filter.Rule]+filter.RawValue)
	}
	if len(filters) == 0 {
		filters = []string{"无"}
	}
	return "排序: " + map[string]string{"dmg_avg": "伤害均值", "mark_score": "遗器评分"}[query.Rank.Col] + " 降序 / 筛选: " + strings.Join(filters, " ")
}

// CustomRankEntry is one row of ark's custom ranking: the UID as ark shows it
// (partly hidden), the Eidolon and the talents ark sends, the light cone by
// ID, the relic sets with ark's label for them, the relic score and the
// ranked detail's average damage.
type CustomRankEntry struct {
	UID                      string
	Cons                     int
	Talents                  map[string]PanelTalent
	Weapon                   string
	WeaponLevel, WeaponAffix int
	Sets                     []string
	SetName                  string
	Mark                     float64
	Damage                   *float64
}

// CloudRankImage is ark's custom ranking as it draws it: the character, the
// mode (dmg or mark), the ranked detail's title, the 排名参与条件 note, the
// rows asked for and the rows returned.
type CloudRankImage struct {
	Character   Entry
	Mode        string
	DamageTitle string
	Limit       string
	Number      int
	Entries     []CustomRankEntry
}

// CloudRankImageBuilder draws ark's custom ranking with the plugin's rank
// template, or returns false to answer in text.
type CloudRankImageBuilder func(ImageContext, CloudRankImage) (Image, bool)

// customRankEntries reads the rows of a rank/custom answer, with ark's
// defaults for missing fields. ark sends the talents before Eidolon bonuses,
// which miao's getAvatarTalent adds, leaving out a talent sent as 0;
// upstream reads no Elation skill, which is drawn without a level.
func customRankEntries(record reference.Character, rows []any) []CustomRankEntry {
	entries := []CustomRankEntry{}
	for _, raw := range rows[:min(len(rows), 50)] {
		row := asObject(raw)
		if row == nil {
			continue
		}
		entry := CustomRankEntry{UID: cloudText(row["uid"]), Cons: Int(row["cons"]), Talents: map[string]PanelTalent{}, Weapon: asText(row["weapon_id"]),
			WeaponLevel: Int(row["weapon_level"]), WeaponAffix: Int(row["weapon_affix"]), Mark: cloudFloat(row["mark_score"])}
		for _, key := range []string{"a", "e", "q", "t", "me", "mt"} {
			if original := Int(row["talent_"+key]); original > 0 {
				entry.Talents[key] = PanelTalent{Level: original + talentBonus(record, key, entry.Cons), Original: original}
			}
		}
		sets := asObject(row["artisSet"])
		for _, name := range asList(sets["names"]) {
			entry.Sets = append(entry.Sets, cloudText(name))
		}
		entry.SetName = cloudText(sets["name"])
		if row["dmg_avg"] != nil {
			damage := cloudFloat(row["dmg_avg"])
			entry.Damage = &damage
		}
		entries = append(entries, entry)
	}
	return entries
}

// cloudFloat reads a number ark sends, 0 when it is not one.
func cloudFloat(value any) float64 {
	number, _ := strconv.ParseFloat(cloudNumber(value), 64)
	return number
}

// customRankCommand answers ark<角色>[伤害|圣遗物|遗器](排行|排名)[ 筛选] as ark's
// rank does. The words are read whole again, as upstream reads the message,
// since the host splits them at spaces.
func (a *App) customRankCommand(ctx context.Context, event *rayleabot.EventContext) error {
	match := customRankWord.FindStringSubmatch(strings.TrimSpace(event.Event.Command() + " " + strings.Join(event.Event.Args(), " ")))
	if match == nil {
		return event.Result(map[string]any{"handled": false})
	}
	sort := "dmg_avg"
	if match[2] == "圣遗物" || match[2] == "遗器" {
		sort = "mark_score"
	}
	columns, denied := []string{"cons"}, "除 cons 之外的筛选仅主人可用"
	if slices.Contains(event.SuperAdmins, event.Event.Actor.ID) {
		columns, denied = customRankColumns, ""
		if !a.arkConfigured(ctx, event) {
			// Upstream points to its #ark配置token; the token is kept by the
			// accounts plugin here.
			columns, denied = []string{"cons"}, "请先在米游社账号插件管理页的“ark 授权令牌”中配置自定义排名 token"
		}
	}
	name, query, err := parseCustomRank(strings.TrimSpace(match[1]+" "+match[4]), columns, sort, denied)
	if err != nil {
		return event.SendText(err.Error())
	}
	if name == "" {
		return event.SendText("请输入角色名")
	}
	character, ok := a.Catalog.Resolve(name, "character", a.aliasMap(event))
	if !ok {
		return event.SendText("找不到角色：" + name)
	}
	for index, filter := range query.Filter {
		if slot, ok := strings.CutPrefix(filter.Type, "pos"); ok {
			id, found := customRankMainStat(slot, filter.RawValue)
			if !found {
				return event.SendText("部位 " + slot + " 不存在主词条：" + filter.RawValue)
			}
			query.Filter[index].Value = id
		}
	}
	id, _ := strconv.Atoi(character.ID)
	decoded, err := a.arkRequest(ctx, event, "rank/custom", map[string]any{"charId": id, "data": query, "game": arkGame})
	result := asObject(decoded)
	if err != nil || asText(result["retcode"]) != "0" {
		return event.SendText(arkQueryFailure(result))
	}
	data := asObject(result["data"])
	var record reference.Character
	if a.Game.Calc != nil {
		record, _ = findReferenceCharacter(a.Game.Calc, CharacterPanel{ID: character.ID, Element: character.Element})
	}
	entries := customRankEntries(record, asList(data["rows"]))
	if len(entries) == 0 {
		return event.SendText(character.Name + " 暂无符合条件的排名数据")
	}
	mode := "dmg"
	if query.Rank.Col == "mark_score" {
		mode = "mark"
	}
	image := CloudRankImage{Character: character, Mode: mode, DamageTitle: cloudText(data["dmgTitle"]), Limit: customRankLimit(query), Number: query.Nums, Entries: entries}
	view := customRankView(a.Game, image)
	if a.cloudRankImage != nil {
		if drawn, ok := a.cloudRankImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	sent := a.sendKept(ctx, event, view)
	if queryID := firstText(data, "query_id", "queryId"); queryID != "" && len(queryID) <= 256 {
		// As upstream, the command and every message of the reply lead back to
		// the query for five minutes.
		for _, id := range append(sent, asText(event.Event.Payload["message_id"])) {
			if id != "" {
				_, _ = event.Actions().KVSetWithOptions(ctx, customRankKey(event, id), map[string]any{"query_id": queryID, "character_id": character.ID, "name": character.Name}, rayleabot.KVSetOptions{TTL: 5 * time.Minute})
			}
		}
	}
	return event.Result(map[string]any{"handled": true})
}

// customRankKey is where a message's custom ranking is remembered; message
// IDs are scoped to the protocol and bot account.
func customRankKey(event *rayleabot.EventContext, messageID string) string {
	return "ark-query:" + event.Event.SourceProtocol + ":" + event.Event.BotID + ":" + messageID
}

// customRankView is the custom ranking in text, for replies without images.
func customRankView(game Game, image CloudRankImage) View {
	title := image.Character.Name + map[string]string{"mark": "遗器评分"}[image.Mode] + "排行"
	v := View{Title: game.Name + " · " + title, Subtitle: "全服数据", Rows: []Row{}, Note: image.Limit}
	for index, entry := range image.Entries {
		value := "评分 " + strconv.FormatFloat(entry.Mark, 'f', 1, 64)
		if entry.Damage != nil {
			value = image.DamageTitle + " " + strconv.FormatFloat(*entry.Damage, 'f', 1, 64) + " · " + value
		}
		v.Rows = append(v.Rows, Row{Label: strconv.Itoa(index+1) + ". " + entry.UID, Value: value})
	}
	return v
}

// sendKept sends a view as sendView does, without ending the event, and
// returns the IDs of the messages sent.
func (a *App) sendKept(ctx context.Context, event *rayleabot.EventContext, view View) []string {
	segment := rayleabot.Text(view.Text())
	if path := a.renderView(ctx, event, view); path != "" {
		segment = rayleabot.Image(path)
	}
	id, err := post(ctx, event, segment)
	if err != nil {
		return nil
	}
	return []string{id}
}

// customRankHelp is ark's ark自定义排行帮助: the advanced help for a super
// administrator when the ark token is set, else the ordinary one. The
// examples name Star Rail characters, whose rankings this plugin answers;
// upstream's help, shared by both games, names Genshin ones besides 符玄.
func (a *App) customRankHelp(ctx context.Context, event *rayleabot.EventContext) error {
	prefix := a.Game.Prefix
	if slices.Contains(event.SuperAdmins, event.Event.Actor.ID) && a.arkConfigured(ctx, event) {
		return event.SendText(strings.Join([]string{
			"【ark自定义排行 · 高级】",
			"用法：" + prefix + "ark<角色>排行 [筛选/数量...]（默认伤害）",
			"用法：" + prefix + "ark<角色>伤害排行 [筛选/数量...]",
			"用法：" + prefix + "ark<角色>遗器排行 [筛选/数量...]",
			"",
			"筛选：列+运算符+值，运算符支持 >= <= > < = !=",
			"可用列：命座(命) 等级 突破 普攻 战技 爆发 武器等级 武器突破 精炼 伤害 评分 遗器",
			"排序：由命令决定，伤害排行按伤害降序，遗器排行按遗器评分降序",
			"数量：nums:N 或 数量:N（1-50，默认20）",
			"星铁可筛部位主词条：部位1~6=词条，如 部位3=暴击",
			"",
			"示例：",
			prefix + "ark希儿排行 命=6",
			prefix + "ark希儿伤害排行 等级>=80 精炼>=1 数量:10",
			prefix + "ark符玄遗器排行 部位4=速度",
		}, "\n"))
	}
	return event.SendText(strings.Join([]string{
		"【ark自定义排行】",
		"用法：" + prefix + "ark<角色>排行 [命座筛选]（默认伤害）",
		"用法：" + prefix + "ark<角色>伤害排行 [命座筛选]",
		"用法：" + prefix + "ark<角色>遗器排行 [命座筛选]",
		"",
		"普通用户仅支持按命座(cons)筛选，",
		"更多筛选项需主人在米游社账号插件管理页配置 ark token 后使用。",
		"命座列名：命座 / 命 / cons，运算符 >= <= > < = !=",
		"",
		"示例：",
		prefix + "ark希儿排行 cons=0",
		prefix + "ark符玄遗器排行 命座<=2",
	}, "\n"))
}

var customRankIndex = regexp.MustCompile(`^(?:[1-9]|1[0-9]|20)$`)

// customRankPanel is ark-plugin's ark获取面板N (apps/customRankPanel.js):
// replying to a custom ranking, or to the command that asked for it, within
// five minutes, it reads the Nth entry's panel with rank/custom/specific and
// shows it as 面板 does, under the UID ark shows. As the panel ark hands
// miao's ProfileDetail, it does not enter the group ranking. Without a
// quoted message the words are left to other plugins.
func (a *App) customRankPanel(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	quoted := quotedMessage(event)
	if len(args) != 1 || !customRankIndex.MatchString(args[0]) || quoted == "" {
		return event.Result(map[string]any{"handled": false})
	}
	index, _ := strconv.Atoi(args[0])
	remembered, err := event.Actions().KVGet(ctx, customRankKey(event, quoted))
	cache := asObject(remembered["value"])
	if err != nil || remembered["exists"] != true || asText(cache["query_id"]) == "" {
		return event.SendText("未找到该排行图片的查询缓存，请重新发送 " + a.Game.Prefix + "ark自定义排行 后再获取面板")
	}
	decoded, err := a.arkRequest(ctx, event, "rank/custom/specific", map[string]any{"query_id": asText(cache["query_id"]), "index": index - 1})
	if err != nil {
		return event.SendText("面板服务暂不可用")
	}
	uid, avatar, failure := customRankAvatar(asObject(decoded), asText(cache["character_id"]))
	if failure != "" {
		return event.SendText("获取面板失败：" + failure)
	}
	_, raw, err := cleanCloudAvatar(avatar)
	if err != nil {
		return event.SendText("获取面板失败：未找到对应角色面板")
	}
	panel, err := a.cloudAvatarPanel(ctx, raw)
	if err != nil {
		return event.SendText("获取面板失败：未找到对应角色面板")
	}
	// The panel shows the service it came from, as miao names it.
	if source := cloudText(avatar["_source"]); source != "" && len(source) <= 32 {
		panel.Source = source
	}
	return a.sendView(ctx, event, a.fullPanelView(ctx, event, panel, uid, false, ""))
}

// customRankAvatar reads a rank/custom/specific answer as upstream's
// getPanelData and setAvatars do: the player data under info or playerData,
// its UID as ark shows it, and the ranked character, else the first. Failed
// answers use local messages without forwarding upstream diagnostics.
func customRankAvatar(result map[string]any, id string) (uid string, avatar map[string]any, failure string) {
	if asText(result["retcode"]) != "0" {
		return "", nil, arkQueryFailure(result)
	}
	data := asObject(result["data"])
	for _, key := range []string{"info", "playerData"} {
		if nested := asObject(data[key]); nested != nil {
			data = nested
			break
		}
	}
	uid, avatars := cloudText(data["uid"]), asObject(data["avatars"])
	if uid == "" || avatars == nil {
		return "", nil, "返回数据异常"
	}
	avatar = asObject(avatars[id])
	if ids := slices.Sorted(maps.Keys(avatars)); avatar == nil && len(ids) > 0 {
		avatar = asObject(avatars[ids[0]])
	}
	if avatar == nil {
		return "", nil, "未找到对应角色面板"
	}
	return uid, avatar, ""
}
