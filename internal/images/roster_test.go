package images_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/app"
	"github.com/RayleaBot/plugin-starrail/internal/assets"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func decodeJSON(t *testing.T, text string) map[string]any {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestRosterFollowsMiao(t *testing.T) {
	application, err := app.New(assets.Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	list := decodeJSON(t, `{"avatar_list":[{"id":8001,"level":80,"rank":6},{"id":1308,"level":80,"rank":0},{"id":1306,"level":70,"rank":1}]}`)
	asked := []string{}
	context := app.ImageContext{Game: application.Game, Catalog: application.Catalog, Now: time.Now(), Query: func(operation string, input map[string]any) (app.QueryResult, error) {
		asked = append(asked, operation)
		// One detail query without IDs answers for every character.
		if input != nil {
			t.Errorf("detail input = %v", input)
		}
		return app.QueryResult{Data: decodeJSON(t, `{"avatar_list":[{"id":1308,"level":80,"rank":0,"equip":{"id":21013,"name":"别让世界静下来","level":80,"rank":5,"rarity":4}}]}`)}, nil
	}}
	image, ok := images.Characters(context, app.QueryResult{Role: app.Role{UID: "100000001", Nickname: "开拓者", Level: 70}, Data: list})
	if !ok || strings.Join(asked, ",") != "starrail.character" {
		t.Fatalf("image = %v asked = %v", image.Data, asked)
	}
	avatars := image.Data["avatars"].([]any)
	// Without talents both count four; the Trailblazer's eidolons win.
	if first := avatars[0].(map[string]any); first["name"] != "穹·毁灭" || avatars[1].(map[string]any)["name"] != "黄泉" || avatars[2].(map[string]any)["level"] != 70 {
		t.Errorf("order = %v", avatars)
	}
	stats := image.Data["stats"].([]any)
	// Acheron counts one gold card; the Trailblazer and a four-star light
	// cone count none; Sparkle at eidolon 1 counts two.
	if len(stats) != 2 || stats[0].(map[string]any)["value"] != 3 || stats[1].(map[string]any)["value"] != 3 {
		t.Errorf("stats = %v", stats)
	}
	if player := image.Data["player"].(map[string]any); player["name"] != "开拓者" || player["show_level"] != true {
		t.Errorf("player = %v", player)
	}

	image, _ = images.Training(context, app.QueryResult{Role: app.Role{UID: "100000001"}, Data: list})
	rows := image.Data["rows"].([]any)
	acheron := rows[1].(map[string]any)
	// miao shortens light cones longer than four characters.
	if weapon := acheron["weapon"].(map[string]any); weapon["name"] != "别让世界静下" || weapon["badge"] != 6 {
		t.Errorf("weapon = %v", acheron["weapon"])
	}
	if talents := rows[0].(map[string]any)["talents"].([]any); len(talents) != 4 || talents[3].(map[string]any)["level"] != "-" {
		t.Errorf("talents = %v", talents)
	}
}
