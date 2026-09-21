package images_test

import (
	"testing"

	gamekit "github.com/RayleaBot/game-plugin-kit"
	"github.com/RayleaBot/plugin-starrail/internal/images"
)

func TestHelpFollowsStarRailPlugin(t *testing.T) {
	help := gamekit.HelpImage{Title: "星铁帮助", Groups: []gamekit.HelpGroup{{Title: "战绩", Commands: []gamekit.HelpCommand{{ID: "challenge", Usage: "*忘却之庭"}, {ID: "gears", Usage: "*黄金"}}}}}
	image, ok := images.Help(gamekit.ImageContext{}, help)
	if !ok {
		t.Fatal("help")
	}
	row := image.Data["groups"].([]any)[0].(map[string]any)["rows"].([]any)[0].([]any)
	// Icon 110 is the last of row 11 in upstream's sprite; commands upstream
	// does not list have none.
	if row[0].(map[string]any)["position"] != "-450px -500px" || row[1].(map[string]any)["position"] != nil || row[2] != nil {
		t.Errorf("row = %v", row)
	}
}
