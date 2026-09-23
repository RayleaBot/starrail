package images

import (
	"strconv"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// APICard draws 面板列表 and 面板更新 the way StarRail-plugin's new_card page
// does: each kept character's avatar, name and eidolons, those the query
// returned marked after 面板更新, with the UID, the service, the update time
// and the sender's name.
func APICard(context app.ImageContext, image app.APICardImage) (app.Image, bool) {
	resources := &app.ImageResources{Context: context}
	resources.Artwork("SR", "starrail-plugin", "resources/common/font/SR.ttf")
	characters := []any{}
	for _, character := range image.Characters {
		id := strconv.Itoa(character.AvatarID)
		characters = append(characters, map[string]any{
			"avatar": resources.Artwork("avatar-"+id, "starrail-plugin", "resources/panel/resources/avatar/"+id+".png"),
			"name":   character.Name, "rank": character.Rank, "updated": image.Update && character.IsNew,
		})
	}
	kind := ""
	if image.Update {
		kind = "update"
	}
	return app.Image{Template: "api-card", Data: map[string]any{
		"uid": image.UID, "api": image.API, "time": image.Time, "user_name": image.UserName, "type": kind, "characters": characters,
	}, Resources: resources.List}, true
}
