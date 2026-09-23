package images

import (
	"strconv"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// cardArtwork maps the images and font the card page names to
// StarRail-plugin's paths.
var cardArtwork = [][2]string{
	{"SR", "resources/common/font/SR.ttf"},
	{"card-logo", "resources/card/images/logo.png"},
	{"card-star", "resources/common/cont/star.png"},
}

// The phone theme address the official card gives when none is set, and the
// theme upstream shows instead.
const (
	cardMissingTheme = "https://act-webstatic.mihoyo.com/game_record/hkrpg/.png"
	cardDefaultTheme = "https://act-webstatic.mihoyo.com/game_record/hkrpg/SpriteOutput/PhoneTheme/Theme/PhoneThemeMain/221000.png"
)

// Card draws 卡片 the way StarRail-plugin's card page does: the phone theme
// behind the avatar, nickname, level, server and UID, the active days,
// characters, achievements, chests and Forgotten Hall progress, and the
// characters the official card lists with their level, element and
// eidolons.
func Card(context app.ImageContext, result app.QueryResult) (app.Image, bool) {
	stats, _ := result.Data["stats"].(map[string]any)
	if stats == nil {
		return app.Image{}, false
	}
	theme := app.Text(result.Data["phone_background_image_url"])
	if theme == cardMissingTheme {
		theme = cardDefaultTheme
	}
	avatars, _ := result.Data["avatar_list"].([]any)
	resources := newChallengeResources(context)
	for _, item := range cardArtwork {
		resources.local(item[0], item[1])
	}
	urls := []any{theme, result.Data["cur_head_icon_url"]}
	for _, raw := range avatars {
		avatar, _ := raw.(map[string]any)
		urls = append(urls, avatar["icon"])
	}
	resources.Prefetch("mihoyo", urls...)
	list := []any{}
	for _, raw := range avatars {
		avatar, _ := raw.(map[string]any)
		element := app.Text(avatar["element"])
		list = append(list, map[string]any{"rarity": app.Text(avatar["rarity"]), "icon": resources.official(avatar["icon"]), "level": app.Text(avatar["level"]),
			"element": element, "element_icon": resources.local("element-"+element, "resources/common/element/"+element+".png"),
			"rank": app.Int(avatar["rank"]), "name": app.Text(avatar["name"])})
	}
	return app.Image{Template: "card", Data: map[string]any{
		"background": resources.official(theme), "head": resources.official(result.Data["cur_head_icon_url"]),
		"nickname": result.Role.Nickname, "level": strconv.Itoa(result.Role.Level), "region": regionName(result.Role.Region), "uid": result.Role.UID,
		"active_days": app.Text(stats["active_days"]), "avatar_num": app.Text(stats["avatar_num"]), "achievement_num": app.Text(stats["achievement_num"]),
		"chest_num": app.Text(stats["chest_num"]), "abyss_process": app.Text(stats["abyss_process"]), "avatars": list,
	}, Resources: resources.List}, true
}
