package images_test

import (
	"testing"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/images"
)

func TestHelpFollowsStarRailPlugin(t *testing.T) {
	help := app.HelpImage{Title: "星铁帮助", Groups: []app.HelpGroup{{Title: "战绩", Commands: []app.HelpCommand{{ID: "challenge", Usage: "*忘却之庭"}, {ID: "gears", Usage: "*黄金"}}}}}
	image, ok := images.Help(app.ImageContext{}, help)
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
