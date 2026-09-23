package app

import (
	"context"
	"math"
	"math/rand/v2"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// StarRail-plugin's own panel service (apps/panel.js) answers beside miao's
// panels, the way Miao-Yunzai routes the words with MiaoOrStarrail's default
// settings (miao on, StarRail-plugin off): MiaoOrStarrail hands the words it
// takes to miao, and panel.js answers the ones it leaves. The service is
// one of the APIs its defSet/panelApi.yaml lists, which the panel_api
// setting picks; the characters it returns are kept per UID apart from the
// other panels, as upstream keeps its data/panel.

// APICardImage is StarRail-plugin's new_card page: a UID's kept characters,
// the service and the update time, and the sender's name. After 面板更新 it
// marks the characters the query returned.
type APICardImage struct {
	UID, API, UserName, Time string
	Update                   bool
	Characters               []APICharacter
}

type APICardImageBuilder func(ImageContext, APICardImage) (Image, bool)

// APIPanelImage is StarRail-plugin's new_panel page for one character: the
// character as the service returned it, the skills the page lists, the
// damage table (nil when the damage module does not calculate the
// character) and the picture drawn, a file of the starrail-plugin source.
type APIPanelImage struct {
	UID, API  string
	Character APICharacter
	Skills    []APISkill
	Damages   []APIDamage
	Picture   string
}

// APISkill is a skill the panel page lists, at the level the eidolons raise
// it to, with its icon under panel/resources/skill.
type APISkill struct {
	APIBehavior
	File string
}

type APIPanelImageBuilder func(ImageContext, APIPanelImage) (Image, bool)

// panelAPI is the API the panel_api setting picks, counted from 1 as
// upstream's default; a setting out of range picks the first.
func (a *App) panelAPI(event *rayleabot.EventContext) (int, string) {
	index := settings(event).PanelAPI
	if index < 1 || index > len(a.Game.PanelAPIs) {
		index = 1
	}
	return index, a.Game.PanelAPIs[index-1]
}

// apiHost is how upstream names an API in replies and pages: the host of its
// address.
func apiHost(api string) string {
	parts := strings.Split(api, "/")
	if len(parts) < 3 {
		return api
	}
	return parts[2]
}

// apiUID is a UID panel.js's panel rule reads after 面板.
var apiUID = regexp.MustCompile(`^[0-9]{9,10}$`)

// apiIndexWord is upstream's reading of 切换面板API: the first number in the
// message.
var apiIndexWord = regexp.MustCompile(`[1-9][0-9]*`)

// apiCommand answers panel.js's commands: 面板 and 面板列表 (plmb), 面板更新
// (update), and 切换面板API and API列表, which it leaves to the bot's masters.
func (a *App) apiCommand(ctx context.Context, event *rayleabot.EventContext, command string, args []string) error {
	master := slices.Contains(event.SuperAdmins, event.Event.Actor.ID)
	switch command {
	case "api-panel-list":
		return a.apiPanelList(ctx, event, args)
	case "api-panel-refresh":
		return a.apiPanelRefresh(ctx, event, args)
	case "api-list":
		if !master {
			return event.SendText("仅限主人可以查看API列表")
		}
		lines := []string{"API列表："}
		for i, api := range a.Game.PanelAPIs {
			lines = append(lines, strconv.Itoa(i+1)+"："+apiHost(api))
		}
		index, api := a.panelAPI(event)
		return event.SendText(strings.Join(lines, "\n") + "\n当前API：\n" + strconv.Itoa(index) + "：" + apiHost(api))
	}
	word := event.Event.Command()
	if len(args) == 0 && (word == "设置面板" || word == "切换面板") {
		// MiaoOrStarrail takes the bare word ending in 面板 first and hands it
		// to miao, which finds no character in it.
		return event.Result(map[string]any{"handled": false})
	}
	if !master {
		return event.SendText("仅限主人可以切换API")
	}
	index, err := strconv.Atoi(apiIndexWord.FindString(strings.Join(append([]string{word}, args...), " ")))
	if err != nil || index > len(a.Game.PanelAPIs) {
		return event.SendText("请输入正确的API序号")
	}
	if _, err = event.Actions().ConfigWrite(ctx, map[string]any{"panel_api": index}); err != nil {
		return event.SendText("切换API失败，请前往控制台查看报错！")
	}
	return event.SendText("切换API成功，当前API：" + apiHost(a.Game.PanelAPIs[index-1]))
}

// apiPanel answers <角色>面板<UID> as panel.js's panel does: the character
// kept for the UID, queried when it is not kept, with its skills, traces,
// relics and the damage its module calculates.
func (a *App) apiPanel(ctx context.Context, event *rayleabot.EventContext, name, uid string) error {
	_, api := a.panelAPI(event)
	character, err := a.apiCharacter(ctx, event, name, uid, api)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	image := APIPanelImage{UID: uid, API: apiHost(api), Character: character, Skills: apiSkills(character, a.panelAPIScripts.RankSkillUp), Picture: a.apiPicture(character)}
	behaviors := []APIBehavior{}
	for _, skill := range image.Skills {
		behaviors = append(behaviors, skill.APIBehavior)
	}
	// The module reads the listed skills as behaviorList and every trace as
	// skillTree; upstream shows no damage when it fails.
	image.Damages, _ = a.panelAPIScripts.Damage(ctx, struct {
		APICharacter
		Behaviors []APIBehavior `json:"behaviorList"`
		Tree      []APIBehavior `json:"skillTree"`
	}{character, behaviors, character.Behaviors})
	view := apiPanelView(image)
	if a.apiPanelImage != nil {
		if drawn, ok := a.apiPanelImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
			// As upstream, 原图 then answers with the picture the panel drew.
			_ = a.rememberImage(event, "artwork:starrail-plugin/"+image.Picture)
		}
	}
	return a.sendView(ctx, event, view)
}

// apiCharacter is getCharData: the kept character, else the one a new query
// returns. It is matched by ID when the name resolves, as the plugin's other
// panel commands match; otherwise by the service's name, as upstream matches
// its alias to the name.
func (a *App) apiCharacter(ctx context.Context, event *rayleabot.EventContext, name, uid, api string) (APICharacter, error) {
	entry, known := a.Catalog.Resolve(name, "character", a.aliasMap(event))
	if known {
		name = entry.Name
	}
	wanted := func(character APICharacter) bool {
		if known {
			return strconv.Itoa(character.AvatarID) == entry.ID
		}
		return character.Name == name
	}
	kept, err := a.APIPanels.Read(uid)
	if err != nil {
		return APICharacter{}, err
	}
	if i := slices.IndexFunc(kept, wanted); i >= 0 {
		return kept[i], nil
	}
	characters, err := a.apiCharacters(ctx, event, uid, api, true)
	if err != nil {
		return APICharacter{}, err
	}
	if i := slices.IndexFunc(characters, wanted); i >= 0 {
		return characters[i], nil
	}
	return APICharacter{}, panelAPIError("未查询到uid：" + uid + " 角色：" + name + " 的数据，请检查角色是否放在了助战或者展柜\n请检查角色名是否正确,已设置的会有延迟,等待一段时间后重试~")
}

// apiSkillKinds name the icons of the first five traces, the skills.
var apiSkillKinds = []string{"basic_atk", "skill", "ultimate", "talent", "technique"}

// apiSkills is handleBehaviorList: the first five traces at the levels the
// character's eidolons raise them to, with their icons, and without the
// technique.
func apiSkills(character APICharacter, raises map[string][]RankSkillUp) []APISkill {
	skills := []APISkill{}
	for i, behavior := range character.Behaviors[:min(len(apiSkillKinds), len(character.Behaviors))] {
		id := strconv.Itoa(behavior.ID)
		key := id[:min(5, len(id))] + id[len(id)-1:]
		for eidolon := 1; eidolon <= character.Rank; eidolon++ {
			for _, raise := range raises[strconv.Itoa(character.AvatarID*100+eidolon)] {
				if raise.ID == key {
					behavior.Level += raise.Num
				}
			}
		}
		if behavior.Type != "秘技" {
			skills = append(skills, APISkill{APIBehavior: behavior, File: id[:min(4, len(id))] + "_" + apiSkillKinds[i] + ".png"})
		}
	}
	return skills
}

// apiPicture is getCharImage, the picture the panel draws: four times in
// five a random one of the character's pro-file directory, else its
// profile/normal-character picture or a random one of its directory there,
// else the default picture. The Trailblazers are named 星 and 穹 there.
func (a *App) apiPicture(character APICharacter) string {
	name := character.Name
	switch character.AvatarID {
	case 8002, 8004:
		name = "星"
	case 8001, 8003:
		name = "穹"
	}
	pictures := func(dir string) []string {
		return slices.DeleteFunc(a.Artwork.List("starrail-plugin", dir), func(file string) bool {
			return !slices.Contains([]string{".jpg", ".png", ".webp"}, strings.ToLower(path.Ext(file)))
		})
	}
	if files := pictures("resources/pro-file/pro-character/" + name); len(files) > 0 && rand.Float64() < 0.8 {
		return files[rand.IntN(len(files))]
	}
	file := "resources/profile/normal-character/" + name + ".webp"
	if _, ok := a.Artwork.File("starrail-plugin", file); ok {
		return file
	}
	if files := pictures("resources/profile/normal-character/" + name); len(files) > 0 {
		return files[rand.IntN(len(files))]
	}
	return "resources/panel/resources/char_image/" + strconv.Itoa(character.AvatarID) + ".png"
}

// apiPanelView is the panel in text, for replies without images.
func apiPanelView(image APIPanelImage) View {
	character := image.Character
	view := View{Title: character.Name + " · 角色面板", Subtitle: "UID：" + image.UID, Rows: []Row{{Label: "等级", Value: "Lv." + strconv.Itoa(character.Level)}, {Label: "星魂", Value: strconv.Itoa(character.Rank)}},
		Sections: []Section{}, Note: "当前面版服务：" + image.API}
	if equipment := character.Equipment; equipment.ID > 0 {
		view.Rows = append(view.Rows, Row{Label: equipment.Name, Value: "Lv." + strconv.Itoa(equipment.Level) + " · 叠影" + strconv.Itoa(equipment.Rank)})
	}
	rows := []Row{}
	for _, skill := range image.Skills {
		rows = append(rows, Row{Label: skill.Type, Value: strconv.Itoa(skill.Level)})
	}
	view.Sections = append(view.Sections, Section{Title: "技能", Rows: rows})
	rows = []Row{}
	for _, relic := range character.Relics {
		rows = append(rows, Row{Label: relic.Name, Value: "+" + strconv.Itoa(relic.Level) + " · " + relic.MainName})
	}
	view.Sections = append(view.Sections, Section{Title: "遗器", Rows: rows})
	if image.Damages != nil {
		rows = []Row{}
		for _, damage := range image.Damages {
			rows = append(rows, Row{Label: damage.Title, Value: "暴击 " + apiDamageText(damage.Value.Critical) + " · 期望 " + apiDamageText(damage.Value.Expect)})
		}
		view.Sections = append(view.Sections, Section{Title: "伤害计算", Rows: rows})
	}
	return view
}

// apiDamageText is a damage as the page writes it, floored; one the module
// could not calculate is NaN.
func apiDamageText(value *float64) string {
	if value == nil {
		return "NaN"
	}
	return strconv.FormatFloat(math.Floor(*value), 'f', -1, 64)
}

// apiOwner is the UID panel.js reads when the command names none: the one in
// use, or that of the user mentioned, which upstream reads from its own
// binding and MiaoOrStarrail meant to be miao's. A sender without one gets
// upstream's reply.
func (a *App) apiOwner(ctx context.Context, event *rayleabot.EventContext, uid string) (string, string) {
	owner, err := a.panelOwner(ctx, event, uid)
	if err == nil {
		return owner.UID, ""
	}
	if PublicError(err).Code == "plugin.game_role_missing" {
		return "", "尚未绑定uid,请发送" + a.Game.Prefix + "绑定uid进行绑定"
	}
	return "", friendlyError(err)
}

// apiPanelList answers 面板 and 面板列表 as plmb does: the characters kept for
// the UID in use, queried from the service when none are kept.
func (a *App) apiPanelList(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	if len(args) > 0 {
		// plmb's rule ends at 列表; miao's list takes the UID after it.
		return a.panelCommand(ctx, event, "panel-list", args)
	}
	uid, refusal := a.apiOwner(ctx, event, "")
	if refusal != "" {
		return event.SendText(refusal)
	}
	_, api := a.panelAPI(event)
	characters, err := a.apiCharacters(ctx, event, uid, api, false)
	if err != nil {
		// Upstream leaves this error to the bot's log; the reply is its
		// other commands'.
		return event.SendText(friendlyError(err))
	}
	// The page shows when the last query returned characters; without one
	// it says so, as upstream means to.
	updated := "该页数据为缓存数据，非最新数据"
	if i := slices.IndexFunc(characters, func(character APICharacter) bool { return character.IsNew && character.LastUpdateTime > 0 }); i >= 0 {
		updated = apiTime(time.UnixMilli(characters[i].LastUpdateTime))
	}
	return a.sendView(ctx, event, a.apiCardView(ctx, APICardImage{UID: uid, API: apiHost(api), UserName: event.Event.Actor.Nickname, Time: updated, Characters: characters}))
}

// apiPanelRefresh answers 面板更新 as update does: it queries the service and
// shows every kept character, marking those returned. Upstream then passes
// the message on, so Atlas and miao's 面板更新 answer it as well.
func (a *App) apiPanelRefresh(ctx context.Context, event *rayleabot.EventContext, args []string) error {
	uid := ""
	if len(args) > 0 {
		uid = args[0]
	}
	uid, refusal := a.apiOwner(ctx, event, uid)
	if refusal != "" {
		return event.SendText(refusal)
	}
	_, api := a.panelAPI(event)
	characters, err := a.apiCharacters(ctx, event, uid, api, true)
	if err != nil {
		return event.SendText(friendlyError(err))
	}
	a.sendKept(ctx, event, a.apiCardView(ctx, APICardImage{UID: uid, API: apiHost(api), UserName: event.Event.Actor.Nickname, Time: apiTime(time.Now()), Update: true, Characters: characters}))
	if handled, err := a.pictureMessage(ctx, event); handled {
		return err
	}
	return a.panelCommand(ctx, event, "panel-refresh", args)
}

// apiCharacters is getPanelData: the UID's kept characters, queried from the
// service when none are kept or refresh asks for it.
func (a *App) apiCharacters(ctx context.Context, event *rayleabot.EventContext, uid, api string, refresh bool) ([]APICharacter, error) {
	kept, err := a.APIPanels.Read(uid)
	if err != nil || len(kept) > 0 && !refresh {
		return kept, err
	}
	notice(ctx, event, "正在获取uid"+uid+"面板数据中~\n可能需要一段时间，请耐心等待")
	fresh, err := a.APIService.Query(ctx, a.panelAPIScripts.Sign, api, uid)
	if err != nil {
		return nil, err
	}
	return a.APIPanels.Keep(uid, fresh, time.Now())
}

// apiTime is upstream's moment format YYYY-MM-DD HH:mm:ss dddd, whose
// weekday is English as no locale is set.
func apiTime(at time.Time) string {
	return at.In(chinaZone).Format("2006-01-02 15:04:05 Monday")
}

func (a *App) apiCardView(ctx context.Context, image APICardImage) View {
	view := View{Title: "角色数据", Subtitle: "UID:" + image.UID, Rows: []Row{}, Note: "面板更新服务: " + image.API + "\n更新时间：" + image.Time}
	for _, character := range image.Characters {
		value := "星魂 " + strconv.Itoa(character.Rank)
		if image.Update && character.IsNew {
			value += " · 本次更新"
		}
		view.Rows = append(view.Rows, Row{Label: character.Name, Value: value})
	}
	if a.apiCardImage != nil {
		if drawn, ok := a.apiCardImage(a.imageContext(ctx), image); ok {
			view.Image = &drawn
		}
	}
	return view
}
