package images

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RayleaBot/starrail/internal/app"
	"github.com/RayleaBot/starrail/internal/artwork"
)

func TestGachaSimulationKeepsDrawOrderAndFallsBackToMiao(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"starrail-plugin/" + simulationImages + "char_image/1005.png",
		"starrail-plugin/" + simulationImages + "weaponlife/BgPathsMemory.png",
		"miao-plugin/resources/meta-sr/character/遐蝶/imgs/splash.webp",
		"miao-plugin/resources/meta-sr/weapon/记忆/新光锥/icon.webp",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("png"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	catalog := app.Catalog{Entries: []app.Entry{
		{ID: "1005", Name: "卡芙卡", Kind: "character", Element: "雷"},
		{ID: "1401", Name: "遐蝶", Kind: "character", Element: "量子"},
		{ID: "21999", Name: "新光锥", Kind: "weapon"},
	}}
	context := app.ImageContext{Artwork: &artwork.Store{Root: root}, Catalog: catalog}
	draws := []app.SimulationDraw{}
	for range 10 {
		draws = append(draws, app.SimulationDraw{Name: "新光锥", Rarity: 3, Item: "weapon"})
	}
	draws[0] = app.SimulationDraw{Name: "卡芙卡", Rarity: 5, Item: "character", Interval: 76}
	draws[9] = app.SimulationDraw{Name: "遐蝶", Rarity: 5, Item: "character", Interval: 12, Guaranteed: true}
	image, ok := GachaSimulation(context, app.SimulationImage{Draws: draws})
	if !ok {
		t.Fatal("no image")
	}
	rows := image.Data["rows"].([]any)
	sizes := []int{}
	for _, row := range rows {
		sizes = append(sizes, len(row.(map[string]any)["cards"].([]any)))
	}
	if len(sizes) != 3 || sizes[0] != 3 || sizes[1] != 4 || sizes[2] != 3 {
		t.Fatalf("row sizes = %v", sizes)
	}
	first := rows[0].(map[string]any)["cards"].([]any)[0].(map[string]any)
	last := rows[2].(map[string]any)["cards"].([]any)[2].(map[string]any)
	cone := rows[1].(map[string]any)["cards"].([]any)[0].(map[string]any)
	if first["count"] != "76抽" || first["rank"] != "five" || last["count"] != "大保底 12抽" || cone["rank"] != "three" || cone["character"] != false {
		t.Errorf("cards = %v / %v / %v", first, last, cone)
	}
	paths := map[string]string{}
	for _, resource := range image.Resources {
		paths[resource.ID] = filepath.ToSlash(resource.Path)
	}
	// StarRail-plugin's own art first; miao's for what its old pool lacks,
	// with the light cone's path found from miao's directories.
	if paths[first["image"].(string)] != "assets/starrail-plugin/"+simulationImages+"char_image/1005.png" ||
		paths[last["image"].(string)] != "assets/miao-plugin/resources/meta-sr/character/遐蝶/imgs/splash.webp" ||
		paths[cone["image"].(string)] != "assets/miao-plugin/resources/meta-sr/weapon/记忆/新光锥/icon.webp" ||
		paths[cone["path"].(string)] != "assets/starrail-plugin/"+simulationImages+"weaponlife/BgPathsMemory.png" {
		t.Errorf("resources = %v", paths)
	}
}
