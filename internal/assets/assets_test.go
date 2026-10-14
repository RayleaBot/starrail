package assets

import (
	"testing"

	"github.com/RayleaBot/starrail/internal/app"
)

// The plugin starts exactly this way; a manifest or data change that the shared
// library rejects must fail here rather than when the host launches it.
func TestPluginStartsFromEmbeddedAssets(t *testing.T) {
	application, err := app.New(Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	application.Close()
}

// Prevent upstream data from naming 8010 星·同谐 again, or assigning 星欢愉
// to both Trailblazers: every built-in name and alias must name one character.
func TestCharacterNamesAndAliasesNameOneCharacter(t *testing.T) {
	application, err := app.New(Load(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()
	for _, entry := range application.Catalog.Entries {
		if entry.Kind != "character" {
			continue
		}
		for _, name := range append([]string{entry.Name}, entry.Aliases...) {
			resolved, ok := application.Catalog.Resolve(name, "character", nil)
			if !ok || resolved.ID != entry.ID {
				t.Errorf("%s (%s) name or alias %q resolved to %q (%v)", entry.Name, entry.ID, name, resolved.ID, ok)
			}
		}
	}
}
