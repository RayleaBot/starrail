package panelapi

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/app"
)

// The pinned signing script runs in the engine and gives upstream's header:
// the time in seconds, then a hex digest.
func TestSignRunsThePinnedScript(t *testing.T) {
	header, err := sign(context.Background(), "100000001")
	if err != nil {
		t.Fatal(err)
	}
	stamp, digest, _ := strings.Cut(header, ",")
	seconds, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil || time.Since(time.Unix(seconds, 0)).Abs() > time.Minute || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(digest) {
		t.Fatal("the header is not the time and a digest")
	}
}

// weltInput is a synthetic Welt at eidolon 6 in the shape the panel page
// passes the damage module.
func weltInput() map[string]any {
	skills := []map[string]any{{"id": 1004001, "level": 6}, {"id": 1004002, "level": 12}, {"id": 1004003, "level": 12}, {"id": 1004004, "level": 12}}
	return map[string]any{"avatarId": 1004, "level": 80, "rank": 6, "damage_type": "Imaginary",
		"properties": map[string]any{"hpBase": 1125, "attackBase": 620, "defenseBase": 509, "speedBase": 102, "imaginaryAdded": 0.388},
		"equipment":  map[string]any{}, "relics": []any{}, "relic_sets": []any{}, "behaviorList": skills, "skillTree": skills}
}

// The module lists its three attacks for Welt. A fresh interpreter each time
// keeps its eidolon 6 extra skill hit from adding up, which the module's
// shared table does upstream on every panel drawn.
func TestDamageRunsTheBundledModule(t *testing.T) {
	var first []app.APIDamage
	for range 2 {
		damages, err := damage(context.Background(), weltInput())
		if err != nil {
			t.Fatal(err)
		}
		if len(damages) != 3 || damages[1].Value.Critical == nil || *damages[1].Value.Critical <= 0 || *damages[1].Value.Expect > *damages[1].Value.Critical {
			t.Fatalf("damages = %+v", damages)
		}
		if first != nil && *damages[1].Value.Critical != *first[1].Value.Critical {
			t.Fatal("the skill's hits added up between panels")
		}
		first = damages
	}
	input := weltInput()
	input["avatarId"] = 1001
	if _, err := damage(context.Background(), input); err == nil {
		t.Error("a character the module does not calculate had damage")
	}
}

func TestScriptsReadTheEidolonTable(t *testing.T) {
	raises := Scripts().RankSkillUp["100103"]
	if len(raises) != 2 || raises[0] != (app.RankSkillUp{ID: "100103", Num: 2}) {
		t.Errorf("eidolon 3 of March 7th raises %+v", raises)
	}
}
