package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RayleaBot/starrail/internal/artwork"
)

func writeArtwork(t *testing.T, root, name, content string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPicturesReadDownloadedLibraries(t *testing.T) {
	root := t.TempDir()
	a := App{Artwork: &artwork.Store{Root: root}, Game: Game{Prefix: "*", Pictures: Pictures{
		Atlas: []AtlasLibrary{{Source: "star-rail-atlas", Index: "path.json"}},
	}}}
	if hint := a.pictureHint(a.Game.Pictures.catalogSources()...); !strings.Contains(hint, "*素材更新 star-rail-atlas") {
		t.Fatal(hint)
	}
	writeArtwork(t, root, "star-rail-atlas/path.json", `{}`)
	if hint := a.pictureHint(a.Game.Pictures.catalogSources()...); hint != "" {
		t.Fatal(hint)
	}
}

func TestStaticPicturesReadCaptionedParts(t *testing.T) {
	root := t.TempDir()
	a := App{Artwork: &artwork.Store{Root: root}}
	writeArtwork(t, root, "starrail-plugin/resources/srsr/sy/1.jpg", "b")
	curve := StaticPicture{Source: "starrail-plugin", Parts: []StaticPart{{Text: "说明"}, {Text: "暴击", Path: "resources/srsr/sy/1.jpg"}}}
	if parts := a.staticParts(curve); len(parts) != 2 || len(parts[1]) != 2 || parts[1][0].Data["text"] != "暴击" {
		t.Fatal(parts)
	}
}
