package images_test

import (
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func TestCardFollowsStarRailPlugin(t *testing.T) {
	data := decodeJSON(t, `{"stats":{"active_days":321,"avatar_num":40,"achievement_num":600,"chest_num":2000,"abyss_process":"第十二关"},
		"phone_background_image_url":"https://act-webstatic.mihoyo.com/game_record/hkrpg/.png","avatar_list":[{"id":1308,"rank":0,"element":"lightning"},{"id":1306,"rank":1}]}`)
	image, ok := images.Card(app.ImageContext{}, app.QueryResult{Role: app.Role{UID: "100000001", Nickname: "开拓者", Level: 70, Region: "prod_qd_cn"}, Data: data})
	if !ok {
		t.Fatal("card")
	}
	// The player's server shows by name.
	if image.Data["region"] != "无名客" || image.Data["level"] != "70" || image.Data["active_days"] != "321" || len(image.Data["avatars"].([]any)) != 2 {
		t.Errorf("card = %v", image.Data)
	}
}
