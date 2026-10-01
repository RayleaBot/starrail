package app

import (
	"context"
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

var cloudRankUID = regexp.MustCompile(`[0-9]{9,10}`)

// digitRuns and nonDigits split getRank's words into the character's name
// and the UID.
var digitRuns, nonDigits = regexp.MustCompile(`\d+`), regexp.MustCompile(`\D+`)

// characterRankCommand is ark-plugin's 角色排名 (getRank): the words after it
// name a character and, with their digits, a UID, and ark answers the
// character's damage rank for the UID after updating it. A name that is no
// character is left to other plugins.
func (a *App) characterRankCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	text := strings.TrimSpace(strings.Join(args, " "))
	name, uid := strings.TrimSpace(digitRuns.ReplaceAllString(text, "")), nonDigits.ReplaceAllString(text, "")
	if name == "" || uid == "" {
		return event.SendText("命令格式错误，示例：" + a.Game.Prefix + "角色排名雷电将军123456789")
	}
	entry, ok := a.Catalog.Resolve(name, "character", a.aliasMap(event))
	if !ok {
		return event.Result(map[string]any{"handled": false})
	}
	id, _ := strconv.Atoi(entry.ID)
	decoded, err := a.arkRequest(ctx, event, "rank/data", map[string]any{"uid": uid, "id": id, "update": 1})
	return event.SendText(characterRankReply(a.Game.Prefix, uid, entry.Name, decoded, err))
}

// characterRankReply is getRank's reply to ark's answer.
func characterRankReply(prefix, uid, name string, decoded any, err error) string {
	if !arkSucceeded(decoded, err) {
		return arkDealError(prefix, decoded, err)
	}
	result := asObject(decoded)
	return "uid:" + uid + "的" + name + "全服伤害排名为 " + asText(result["rank"]) + "，伤害评分: " + arkScore(result["score"])
}

// totalRankCommand is ark-plugin's 总排名 (getAllRank): ark's damage rank of
// every character kept for the UID, the one named, mentioned or in use.
func (a *App) totalRankCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	owner, err := a.panelOwner(ctx, event, cloudRankUID.FindString(strings.Join(args, "")))
	if err != nil || owner.UID == "" {
		return event.SendText("请先绑定UID")
	}
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	ids := slices.Sorted(maps.Keys(saved.Panels))
	names := []string{}
	for _, id := range ids {
		name := id
		if entry, ok := a.Catalog.Get(id); ok {
			name = entry.Name
		}
		names = append(names, name)
	}
	decoded, err := a.arkRequest(ctx, event, "rank/self", map[string]any{"ids": ids, "uid": owner.UID, "type": arkGame})
	return event.SendText(totalRankReply(a.Game.Prefix, owner.UID, names, decoded, err))
}

// totalRankReply is getAllRank's reply to ark's answer: a line for each
// character ark ranks, in the order they were asked for.
func totalRankReply(prefix, uid string, names []string, decoded any, err error) string {
	if !arkSucceeded(decoded, err) {
		return arkDealError(prefix, decoded, err)
	}
	text := "uid:" + uid + "的星铁全服排名数据:\n"
	for index, raw := range asList(asObject(decoded)["rank"]) {
		item := asObject(raw)
		if asText(item["retcode"]) == "100" && index < len(names) {
			text += names[index] + "全服伤害排名为" + asText(item["rank"]) + "，伤害评分: " + arkScore(item["score"]) + "\n"
		}
	}
	return text
}

// arkScore writes a damage score as ark-plugin does, with toFixed(2).
func arkScore(value any) string {
	score, err := strconv.ParseFloat(asText(value), 64)
	if err != nil {
		return asText(value)
	}
	return JSFixed(score, 2)
}

// rankStatsCommand is ark-plugin's <角色>排名统计 (getSpecificRank), which
// asks rank/specific for the character's ranking distribution. ark now
// answers with the two distributions its panel page draws as 排名统计, the
// relic scores first; upstream's command reads a single distribution from
// the answer's data and so only replies 未知错误. They are drawn here as that
// block draws them, without a panel's place; without an image the reply lists
// each distribution's scores, as other commands fall back to text. A name
// that is no character ends the message without a reply, as upstream's
// return true does; a failed request, a failed answer or distributions that
// cannot be read reply as upstream's dealError.
func (a *App) rankStatsCommand(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	entry, ok := a.Catalog.Resolve(strings.TrimSpace(strings.Join(args, "")), "character", a.aliasMap(event))
	if !ok {
		return event.Result(map[string]any{"handled": true})
	}
	id, _ := strconv.Atoi(entry.ID)
	decoded, err := a.arkRequest(ctx, event, "rank/specific", map[string]any{"id": id, "percent": 0})
	distributions, ok := decoded.([]any)
	if err != nil || !ok {
		return event.SendText(arkDealError(a.Game.Prefix, decoded, err))
	}
	image, view, ok := readRankStats(entry, distributions)
	if !ok {
		return event.SendText(arkDealError(a.Game.Prefix, nil, nil))
	}
	if a.rankStatsImage != nil {
		if drawn, ok := a.rankStatsImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return a.sendView(ctx, event, view)
}

// RankStatsImage is what <角色>排名统计 draws: the character, ark's damage
// and relic distributions (nil when ark sent none), the calculation the
// damage ranks and how many UIDs each distribution counts.
type RankStatsImage struct {
	Character     Entry
	Damage, Artis *RankCurve
	DamageTitle   string
	Totals        [2]string
}

// RankStatsImageBuilder draws <角色>排名统计 with the plugin's template, or
// returns false when it cannot.
type RankStatsImageBuilder func(ImageContext, RankStatsImage) (Image, bool)

// readRankStats reads rank/specific's answer as the panel page reads it: the
// damage distribution second and the relic one first, as rankCurve turns
// them into curves, with no panel placed on them, and as text the score at
// each ranking percentile. ok is false when neither can be read.
func readRankStats(entry Entry, distributions []any) (image RankStatsImage, view View, ok bool) {
	image = RankStatsImage{Character: entry}
	none := RankCurve{Percent: -100, Score: -100}
	data := func(index int) map[string]any { return asObject(asObject(distributions[index])["data"]) }
	if len(distributions) > 1 {
		if image.Damage = rankCurve(distributions[1], none, true); image.Damage != nil {
			image.DamageTitle, image.Totals[0] = plainGameText(asText(data(1)["name"])), cloudNumber(data(1)["total"])
		}
	}
	if len(distributions) > 0 {
		if image.Artis = rankCurve(distributions[0], none, false); image.Artis != nil {
			image.Totals[1] = cloudNumber(data(0)["total"])
		}
	}
	if image.Damage == nil && image.Artis == nil {
		return RankStatsImage{}, View{}, false
	}
	view = View{Title: entry.Name + "排名统计", Rows: []Row{}}
	if image.DamageTitle != "" {
		view.Subtitle = "排名趋势 · " + image.DamageTitle
	}
	for _, part := range []struct {
		title string
		index int
		total string
	}{{"伤害排名", 1, image.Totals[0]}, {"遗器排名", 0, image.Totals[1]}} {
		section := Section{Title: part.title, Rows: []Row{}}
		var scores []float64
		if part.index < len(distributions) {
			scores = cloudScores(data(part.index))
		}
		if scores == nil {
			section.Text = "暂无数据"
		} else {
			section.Rows = append(section.Rows, Row{Label: "统计样本", Value: part.total + " 个UID"})
			for index, score := range scores {
				section.Rows = append(section.Rows, Row{Label: "TOP " + strconv.Itoa(cloudPercentiles[index]) + "%", Value: strconv.FormatFloat(score, 'f', -1, 64)})
			}
		}
		view.Sections = append(view.Sections, section)
	}
	return image, view, true
}

// cloudExchangeCommand is ark-plugin's 导出面板数据 and 导入面板数据<UID>:
// export uploads the UID's player data file as upstream uploads miao's, for
// ten minutes; import downloads a UID's upload into the UID's kept panels.
// Who may do either is ark's exportPanelData and importPanelData; the UID is
// the one named, mentioned or in use. An answer other than retcode 100, or a
// failed request, replies as upstream's dealError.
func (a *App) cloudExchangeCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	level := settings(event).Ark.ImportPanelData
	if command == "cloud-export" {
		level = settings(event).Ark.ExportPanelData
	}
	listed, _ := a.accountClient(event).List(ctx, 0)
	if refusal := panelDataRefusal(level, hasAccount(listed), slices.Contains(event.SuperAdmins, event.Event.Actor.ID)); refusal != "" {
		return event.SendText(refusal)
	}
	owner, err := a.panelOwner(ctx, event, cloudRankUID.FindString(strings.Join(args, "")))
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	if command == "cloud-export" {
		saved, err := a.Profiles.Read(owner.UID)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		raw, count, err := a.playerDataFile(ctx, saved)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		if count == 0 {
			return event.SendText("面板数据文件不存在，请先更新面板数据")
		}
		decoded, err := a.arkRequest(ctx, event, "panel/upload", map[string]any{"uid": owner.UID, "type": arkGame, "data": string(raw)})
		if !arkSucceeded(decoded, err) {
			return event.SendText(arkDealError(a.Game.Prefix, decoded, err))
		}
		return event.SendText("导出成功，请在另一个安装此插件的Bot上输入 " + a.Game.Prefix + "导入面板数据" + owner.UID + " ，有效期十分钟~")
	}
	decoded, err := a.arkRequest(ctx, event, "panel/download", map[string]any{"uid": owner.UID, "type": arkGame})
	if !arkSucceeded(decoded, err) {
		return event.SendText(arkDealError(a.Game.Prefix, decoded, err))
	}
	data, err := cloudPlayerObject(asObject(decoded)["data"])
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	avatars, err := cleanCloudPlayer(owner.UID, data)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	// As upstream writes the download into the UID's player data, the panels
	// are kept where 面板 reads them.
	panels := []CharacterPanel{}
	for _, raw := range avatars {
		panel, err := a.cloudAvatarPanel(ctx, raw)
		if err != nil {
			return event.SendText(friendlyError(err))
		}
		panels = append(panels, panel)
	}
	if _, err = a.Profiles.Keep(owner.UID, panels, "share", nil); err != nil {
		return event.SendText(friendlyError(err))
	}
	return event.SendText("导入成功")
}

// cloudAvatarPanel turns an exchange avatar (miao player data, as cleaned by
// cleanCloudAvatar) into a panel, its properties calculated from its parts as
// miao does when it loads saved player data.
func (a *App) cloudAvatarPanel(ctx context.Context, raw json.RawMessage) (CharacterPanel, error) {
	var avatar struct {
		ID      int            `json:"id"`
		Elem    string         `json:"elem"`
		Level   int            `json:"level"`
		Promote *int           `json:"promote"`
		Cons    int            `json:"cons"`
		Talent  map[string]int `json:"talent"`
		Trees   []string       `json:"trees"`
		Weapon  *struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Level   int    `json:"level"`
			Promote *int   `json:"promote"`
			Affix   int    `json:"affix"`
		} `json:"weapon"`
		Artis map[string]map[string]any `json:"artis"`
	}
	if err := json.Unmarshal(raw, &avatar); err != nil || a.Game.Calc == nil {
		return CharacterPanel{}, gameError("cloud_invalid", "面板文件的角色数据无法读取。")
	}
	id := strconv.Itoa(avatar.ID)
	record, err := findReferenceCharacter(a.Game.Calc, CharacterPanel{ID: id, Element: avatar.Elem})
	if err != nil {
		return CharacterPanel{}, err
	}
	panel := CharacterPanel{ID: id, Level: avatar.Level, Promote: avatar.Promote, Rank: max(0, min(6, avatar.Cons)), Element: record.Element, Source: "share", RankKnown: true, WeaponKnown: true, EquipmentKnown: true,
		Stats: []PanelStat{}, Equipment: []PanelEquipment{}, Skills: []PanelSkill{}, Ranks: []PanelSkill{}}
	if entry, ok := a.Catalog.Get(id); ok {
		panel.Name = entry.Name
	}
	for slot := 1; slot <= 6; slot++ {
		if gear := avatar.Artis[strconv.Itoa(slot)]; gear != nil {
			if _, piece := decodeCloudGear(a.Game, slot, gear); piece != nil {
				panel.Equipment = append(panel.Equipment, *piece)
			}
		}
	}
	weapon := PanelEquipment{Main: []PanelStat{}, Sub: []PanelStat{}, Complete: true}
	if avatar.Weapon != nil {
		// Player data gives the light cone's ID.
		for _, item := range a.Game.Calc.Metadata().Weapons {
			if avatar.Weapon.ID != 0 && item.ID == strconv.Itoa(avatar.Weapon.ID) || avatar.Weapon.Name != "" && item.Name == avatar.Weapon.Name {
				weapon.ID, weapon.Name = item.ID, item.Name
			}
		}
		if entry, ok := a.Catalog.Get(weapon.ID); ok {
			weapon.Rarity = strconv.Itoa(entry.Rarity)
		}
		weapon.Level, weapon.Promote, weapon.Refinement = avatar.Weapon.Level, avatar.Weapon.Promote, max(1, min(5, avatar.Weapon.Affix))
	}
	talents := map[string]int{}
	for key, level := range avatar.Talent {
		talents[key] = level + talentBonus(record, key, panel.Rank)
	}
	return a.computedPanel(ctx, record, panel, weapon, talents, avatar.Trees)
}

// panelDataRefusal is ark-plugin's checkPermission: the reply refusing
// 导出面板数据 or 导入面板数据 at a permission level, empty when allowed.
func panelDataRefusal(level int, account, superAdmin bool) string {
	switch {
	case level == 0, level == 1 && (account || superAdmin), level == 2 && superAdmin:
		return ""
	case level == 1:
		return "为确保数据安全，目前仅允许绑定CK用户导入/导出自己UID的面板数据，请联系Bot主人导入/导出..."
	case level == 2:
		return "为确保数据安全，目前仅允许主人导入/导出自己UID的面板数据，请联系Bot主人导入/导出..."
	}
	return "当前功能已被禁用..."
}

// hasAccount is Yunzai's user.hasCk: an account of the game is bound.
func hasAccount(listed Accounts) bool {
	return slices.ContainsFunc(listed.Items, func(account Account) bool { return len(account.Roles) > 0 })
}

func cmpOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// cloudUsage is ark-plugin's arktoken用量 (apps/usage.js) for super
// administrators: the quota ark reports for the ark token, or for anonymous
// requests while none is set.
func (a *App) cloudUsage(ctx context.Context, event *rayleabot.EventContext) error {
	decoded, err := a.arkRequest(ctx, event, "auth/usage", map[string]any{})
	if err != nil {
		return event.SendText("查询失败，服务暂不可用")
	}
	return event.SendText(arkUsageText(decoded))
}

// arkUsageText is usage.js's formatUsageResult: the permission, what is left
// of every request's minute, hour and day quota, the custom ranking's
// multiplier and what is left of its ordinary and advanced quota.
func arkUsageText(decoded any) string {
	result := asObject(decoded)
	data := asObject(result["data"])
	if asText(result["retcode"]) != "0" || data == nil {
		return arkQueryFailure(result)
	}
	auth, quota := asObject(data["auth"]), asObject(data["quota"])
	rank, custom := asObject(quota["rank"]), asObject(quota["custom"])
	remaining := func(item any) string { return cmpOr(asText(asObject(item)["remaining"]), "-") }
	permission := map[string]string{"1": "高级"}[asText(auth["permission"])]
	return strings.Join([]string{
		"权限类型：" + cmpOr(permission, "普通"),
		"全部请求剩余额度：" + remaining(rank["minute"]) + "/" + remaining(rank["hour"]) + "/" + remaining(rank["day"]),
		"自定义排名请求额度倍率：" + cmpOr(asText(auth["limit_normal"]), "1") + "x",
		"自定义排名普通请求剩余额度：" + remaining(custom["normal"]),
		"自定义排名高级请求剩余额度：" + remaining(custom["advanced"]),
	}, "\n")
}

// arkPanelRefresh is ark-plugin's refreshPanel, which runs before miao's
// 更新面板 for the sender's current UID, whatever UID the words name: with
// newUserPanel a UID without kept panels first takes the panels ark keeps for
// it (ark hands them over to the QQ that verified the UID), then ark is told
// to refresh its own copy. Upstream does not wait for ark's answer; the
// accounts plugin answers only while the event is open, so the notice is
// waited for, five seconds at most.
func (a *App) arkPanelRefresh(ctx context.Context, event *rayleabot.EventContext) {
	listed, err := a.accountClient(event).List(ctx, 0)
	if err != nil {
		return
	}
	uid := a.currentUID(listed)
	if uid == "" {
		return
	}
	if settings(event).Ark.NewUserPanel {
		a.arkNewUserPanel(ctx, event, uid)
	}
	notified, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, _ = a.arkRequest(notified, event, "panel/refresh", map[string]any{"uid": uid, "type": arkGame})
}

// arkNewUserPanel keeps the panels ark keeps for a UID that has none kept,
// and says how many characters ark sent. ark checks the UID against the QQ
// that verified it, so only OneBot11 senders are asked for.
func (a *App) arkNewUserPanel(ctx context.Context, event *rayleabot.EventContext, uid string) {
	if event.Event.SourceProtocol != "onebot11" || !cloudQQPattern.MatchString(event.Event.Actor.ID) {
		return
	}
	if saved, err := a.Profiles.Read(uid); err != nil || len(saved.Panels) > 0 {
		return
	}
	decoded, err := a.arkRequest(ctx, event, "panel/data", map[string]any{"uid": uid, "type": arkGame, "qq": event.Event.Actor.ID})
	result := asObject(decoded)
	if err != nil || asText(result["retcode"]) != "100" {
		return
	}
	player := asObject(asObject(result["data"])["playerData"])
	if panels := a.arkPlayerPanels(ctx, uid, player); len(panels) > 0 {
		if _, err := a.Profiles.Keep(uid, panels, "share", nil); err == nil {
			notice(ctx, event, "[ark-plugin]已自动从API获取"+strconv.Itoa(len(asObject(player["avatars"])))+"个数据")
		}
	}
}

// arkPlayerPanels are the panels of the player data ark keeps for a UID,
// calculated from their parts as 导入面板数据 does; characters that cannot be
// read are left out.
func (a *App) arkPlayerPanels(ctx context.Context, uid string, player map[string]any) []CharacterPanel {
	panels := []CharacterPanel{}
	avatars, err := cleanCloudPlayer(uid, player)
	if err != nil {
		return panels
	}
	for _, raw := range avatars {
		if panel, err := a.cloudAvatarPanel(ctx, raw); err == nil {
			panels = append(panels, panel)
		}
	}
	return panels
}
