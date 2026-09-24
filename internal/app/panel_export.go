package app

import (
	"context"
	"encoding/json"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// panelExportCommand is ark-plugin's 导出面板: the saved panels of the UID in
// use, or of a mentioned player, sent as a miao player data file, which
// upstream sends with every character miao keeps, whichever service it came
// from. A panel that cannot be written out is left out, and without any the
// reply is upstream's for a missing file. Who may export is ark's
// exportPanelRequire; a refused user gets no reply, as upstream.
func (a *App) panelExportCommand(ctx context.Context, event *rayleabot.EventContext) error {
	owner, err := a.panelOwner(ctx, event, "")
	if err != nil || owner.UID == "" {
		return event.SendText("请先绑定uid")
	}
	if !a.panelExportAllowed(ctx, event, owner.UID) {
		return event.Result(map[string]any{"handled": true})
	}
	saved, err := a.Profiles.Read(owner.UID)
	if err != nil {
		return event.SendText("面板数据文件不存在，请先更新面板数据")
	}
	content, count, err := a.playerDataFile(ctx, saved)
	if err != nil {
		return err
	}
	if count == 0 {
		return event.SendText("面板数据文件不存在，请先更新面板数据")
	}
	if len(content) > gachaFileLimit {
		return event.SendText("导出失败：面板文件超过 5 MB")
	}
	return sendFile(event, owner.UID+".json", content)
}

// panelExportAllowed reads ark's exportPanelRequire as its settings describe
// it: 0 anyone, 1 users with an account or whose UID ark verified for their
// QQ, 2 users with an account, 3 no one. Upstream's condition runs the other
// way, exporting for 3, never for 0, and for users without an account or
// whose UID ark has not verified.
func (a *App) panelExportAllowed(ctx context.Context, event *rayleabot.EventContext, uid string) bool {
	level := settings(event).Ark.ExportPanelRequire
	if level == 0 {
		return true
	}
	if level > 2 {
		return false
	}
	if listed, err := a.accountClient(event).List(ctx, 0); err == nil && hasAccount(listed) {
		return true
	}
	if level != 1 || event.Event.SourceProtocol != "onebot11" || !cloudQQPattern.MatchString(event.Event.Actor.ID) {
		return false
	}
	decoded, err := a.arkRequest(ctx, event, "verify/user", map[string]any{"uid": uid, "qq": event.Event.Actor.ID, "type": arkGame})
	return err == nil && asText(asObject(decoded)["retcode"]) == "100"
}

// miaoSources are the names miao saves a service's characters under (their
// _source): MysPanelHSR's for the official panel and HomoApi's for
// Mihomo's. A panel from ark keeps share.
var miaoSources = map[string]string{"mihoyo": "mysPanelHSR", "mihomo": "homo"}

// miaoPlayer is miao's player data file in the order Player.save writes it:
// the UID, the player's name and level, the fields miao leaves empty when no
// service fills them, when the panels were last refreshed, and the
// characters.
type miaoPlayer struct {
	UID     string                     `json:"uid"`
	Name    string                     `json:"name"`
	Level   any                        `json:"level"`
	Word    string                     `json:"word"`
	Face    string                     `json:"face"`
	Card    string                     `json:"card"`
	Sign    string                     `json:"sign"`
	Info    bool                       `json:"info"`
	Profile int64                      `json:"_profile,omitempty"`
	Avatars map[string]json.RawMessage `json:"avatars"`
}

// playerDataFile writes a UID's kept panels as the player data file miao
// saves, which ark-plugin's 导出面板 sends and its 导出面板数据 uploads as it
// is: every character that can be written out, under the service it came
// from. It returns the file and how many characters it holds.
func (a *App) playerDataFile(ctx context.Context, saved SavedProfiles) ([]byte, int, error) {
	player := miaoPlayer{UID: saved.UID, Name: saved.Nickname, Level: "", Profile: saved.RefreshedAtMS, Avatars: map[string]json.RawMessage{}}
	if saved.Level > 0 {
		player.Level = saved.Level
	}
	for _, kept := range saved.Panels {
		panel := kept.panel()
		if id, raw, err := cloudAvatar(ctx, a.Game, panel, cmpOr(miaoSources[panel.Source], panel.Source)); err == nil {
			player.Avatars[id] = raw
		}
	}
	content, err := json.Marshal(player)
	return content, len(player.Avatars), err
}
