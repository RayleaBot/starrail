package app

import (
	"encoding/json"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

// The record API shows substats cut, not rounded; miao's MysPanelHSR takes
// the official roll count and rounds the steps the value shown adds.
func TestOfficialRollsReadCutValues(t *testing.T) {
	g := testGame(t)
	cases := []struct {
		stat  PanelStat
		times int
		want  string
	}{
		{PanelStat{Key: "atk", Value: "3.4%"}, 1, "5,1,0"},      // 3.456%
		{PanelStat{Key: "cdmg", Value: "17.4%"}, 3, "9,3,3"},    // 17.496%
		{PanelStat{Key: "atkPlus", Value: "52"}, 3, "2,3,1"},    // 52.92
		{PanelStat{Key: "speed", Value: "8"}, 4, "7,4,0"},       // speed is corrected later
		{PanelStat{Key: "cpct", Value: "1000%"}, 1, "8,1,3078"}, // upstream checks no table
	}
	for _, tc := range cases {
		if rolls, err := officialRolls(g, 5, tc.stat, tc.times); err != nil || len(rolls) != 1 || rolls[0] != tc.want {
			t.Errorf("%+v ×%d = %v %v, want %s", tc.stat, tc.times, rolls, err, tc.want)
		}
	}
	// Every roll of every table is read; a five-star relic's, cut as the
	// record API shows it, gives back its count and steps, speed aside.
	for star, table := range g.Data.CloudGear.Meta.StarData {
		for id, attr := range table.Sub {
			percent := !strings.HasSuffix(attr.Key, "Plus") && attr.Key != "speed"
			for count := 1; count <= 6; count++ {
				for step := 0; step <= count*2; step++ {
					value := attr.Base*float64(count) + attr.Step*float64(step)
					shown := strconv.FormatFloat(math.Floor(value+1e-9), 'f', 0, 64)
					if percent {
						shown = strconv.FormatFloat(math.Floor(value*10+1e-9)/10, 'f', 1, 64) + "%"
					}
					rolls, err := officialRolls(g, number(star), PanelStat{Key: attr.Key, Value: shown}, count)
					if err != nil {
						t.Fatalf("%s★ %s %s ×%d refused: %v", star, attr.Key, shown, count, err)
					}
					if want := id + "," + strconv.Itoa(count) + "," + strconv.Itoa(step); star == "5" && attr.Key != "speed" && rolls[0] != want {
						t.Errorf("%s★ %s %s ×%d = %v, want %s", star, attr.Key, shown, count, rolls, want)
					}
				}
			}
		}
	}
}

// A Mihomo panel keeps each substat's roll count and its value cut as the
// game shows it; the step is the one whose value cuts to it.
func TestShowcaseRollsRestoreCutValues(t *testing.T) {
	g := testGame(t)
	cases := []struct {
		star int
		stat PanelStat
		want string
	}{
		{5, PanelStat{Key: "speed", Value: "4.6", Times: 1}, "7,2,2"},
		{5, PanelStat{Key: "cdmg", Value: "18.1%", Times: 2}, "9,3,4"},
		{5, PanelStat{Key: "atkPlus", Value: "21", Times: 0}, "2,1,2"},
		// 3.456% shows as 3.4%, further from it than the closest-value search
		// of official values allows.
		{4, PanelStat{Key: "hp", Value: "3.4%", Times: 0}, "4,1,2"},
	}
	for _, tc := range cases {
		if rolls, err := showcaseRolls(g, tc.star, tc.stat); err != nil || len(rolls) != 1 || rolls[0] != tc.want {
			t.Errorf("%+v = %v %v, want %s", tc.stat, rolls, err, tc.want)
		}
	}
	if _, err := showcaseRolls(g, 5, PanelStat{Key: "speed", Value: "4.6"}); err == nil {
		t.Fatal("restored speed no single roll gives")
	}
	// miao's HomoApi keeps Mihomo's count and step as they are; every roll of
	// every table, cut as Mihomo shows it, gives them back. A two-star relic's
	// flat attack and defence step by less than one, so two of their steps can
	// cut to the same whole number.
	for star, table := range g.Data.CloudGear.Meta.StarData {
		for id, attr := range table.Sub {
			if star == "2" && (attr.Key == "atkPlus" || attr.Key == "defPlus") {
				continue
			}
			for count := 1; count <= 6; count++ {
				for step := 0; step <= count*2; step++ {
					value := attr.Base*float64(count) + attr.Step*float64(step)
					shown := strconv.FormatFloat(math.Floor(value+1e-9), 'f', 0, 64)
					switch {
					case attr.Key == "speed":
						shown = strconv.FormatFloat(math.Floor(value*10+1e-9)/10, 'f', -1, 64)
					case !strings.HasSuffix(attr.Key, "Plus"):
						shown = strconv.FormatFloat(math.Floor(value*10+1e-9)/10, 'f', 1, 64) + "%"
					}
					want := id + "," + strconv.Itoa(count) + "," + strconv.Itoa(step)
					if rolls, err := showcaseRolls(g, number(star), PanelStat{Key: attr.Key, Value: shown, Times: count - 1}); err != nil || rolls[0] != want {
						t.Errorf("%s★ %s %s ×%d = %v %v, want %s", star, attr.Key, shown, count, rolls, err, want)
					}
				}
			}
		}
	}
}

// sharedSeele is a panel ark hands over: 希儿 with two relics, whose
// substats are the attrIds of sharedRelics.
func sharedSeele(t *testing.T, a *App) CharacterPanel {
	t.Helper()
	player := cloudObject(t, `{"uid":"100000001","avatars":{"1102":{"id":1102,"elem":"quantum","level":80,"promote":6,"cons":0,"talent":{"a":6,"e":10,"t":10,"q":10},"trees":["1102101"],
		"weapon":{"id":23001,"level":80,"promote":6,"affix":1},"artis":{"1":{"level":15,"star":5,"id":61011,"mainId":1,"attrIds":["5,2,2","7,1,1","9,3,4","8,1,0"]},
		"2":{"level":15,"star":5,"id":61012,"mainId":1,"attrIds":["1,2,1","7,2,2","10,1,2","12,1,0"]}}}}}`)
	panels := a.arkPlayerPanels(t.Context(), "100000001", player)
	if len(panels) != 1 {
		t.Fatalf("panels = %+v", panels)
	}
	return panels[0]
}

var sharedRelics = map[string][]string{"1": {"5,2,2", "7,1,1", "8,1,0", "9,3,4"}, "2": {"10,1,2", "12,1,0", "1,2,1", "7,2,2"}}

// shownPanel is sharedSeele as a service shows it: each substat's value and
// the count the service gives by slot and key.
func shownPanel(panel CharacterPanel, source string, shown map[string][2]any) CharacterPanel {
	panel.Source = source
	panel.Equipment = slices.Clone(panel.Equipment)
	for index, piece := range panel.Equipment {
		piece.Sub = slices.Clone(piece.Sub)
		for i, stat := range piece.Sub {
			value := shown[stat.Key]
			if item, ok := shown[strconv.Itoa(piece.Slot)+stat.Key]; ok {
				value = item
			}
			piece.Sub[i].Value, piece.Sub[i].Times = value[0].(string), value[1].(int)
		}
		panel.Equipment[index] = piece
	}
	return panel
}

// writtenRelics are the attrIds cloudAvatar writes for a panel, by slot,
// sorted.
func writtenRelics(t *testing.T, a *App, panel CharacterPanel) map[string][]string {
	t.Helper()
	_, raw, err := cloudAvatar(t.Context(), a.Game, panel, "share")
	if err != nil {
		t.Fatal(err)
	}
	var avatar struct {
		Artis map[string]struct{ AttrIds []string } `json:"artis"`
	}
	if err := json.Unmarshal(raw, &avatar); err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for slot, piece := range avatar.Artis {
		out[slot] = slices.Sorted(slices.Values(piece.AttrIds))
	}
	return out
}

// officialShown is sharedSeele's relics as the record API shows them: the
// official counts, and the values cut to one decimal for percentages and to
// whole numbers otherwise, speed included.
var officialShown = map[string][2]any{"atk": {"7.7%", 2}, "1speed": {"2", 1}, "2speed": {"4", 2}, "cdmg": {"18.1%", 3}, "cpct": {"2.5%", 1},
	"hpPlus": {"71", 2}, "effPct": {"4.3%", 1}, "stance": {"5.1%", 1}}

// An official panel is written as miao's MysPanelHSR reads it: the counts the
// record API gives with the steps the cut values round to, without the
// substat it only previews, and with speed steps added one relic after
// another until the speed reaches the official total, itself cut to a whole
// number. A changed panel keeps the relics of the panel it took them from.
func TestOfficialPanelWritesAsMysPanelHSR(t *testing.T) {
	a := pluginApp(t)
	official := shownPanel(sharedSeele(t, a), "mihoyo", officialShown)
	// The first relic's speed is 2.6 (7,1,2) here, which the record API shows
	// as 2; the total speed, 0.3 above the shared panel's, shows cut too.
	official.Stats = slices.Clone(official.Stats)
	for i, stat := range official.Stats {
		if value, _ := buildStatNumber(stat.Value); stat.ID == "4" {
			official.Stats[i].Value = strconv.Itoa(int(value + 0.3))
		}
	}
	// A relic below +3 shows the substat it will gain as a preview.
	official.Equipment[0].Sub = append(official.Equipment[0].Sub, PanelStat{ID: "31", Key: "defPlus", Value: "19", Times: 1})
	official.Official = map[string]any{"relics": []any{map[string]any{"pos": json.Number("1"),
		"properties": []any{map[string]any{"property_type": json.Number("31"), "times": json.Number("1"), "value": "19", "is_preview": true}}}}}
	// Both speed substats restore at step 0, a whole 1.0 below the total;
	// miao adds a step to each in turn, twice, and stops 0.2 past it.
	want := map[string][]string{"1": {"5,2,2", "7,1,2", "8,1,0", "9,3,4"}, "2": {"1,2,1", "10,1,2", "12,1,0", "7,2,2"}}
	if got := writtenRelics(t, a, official); !reflect.DeepEqual(got, want) {
		t.Fatalf("official panel = %v, want %v", got, want)
	}
	if _, err := a.Profiles.Keep("100000001", []CharacterPanel{official}, "米游社", nil); err != nil {
		t.Fatal(err)
	}
	six := 6
	changed, err := a.changedPanel(t.Context(), "100000001", PanelChange{CharacterID: "1102", Cons: &six})
	if err != nil {
		t.Fatal(err)
	}
	if got := writtenRelics(t, a, changed); !reflect.DeepEqual(got, want) {
		t.Fatalf("changed panel = %v, want %v", got, want)
	}
}

// 导出面板 writes every kept panel back as miao keeps it: a panel ark handed
// over and one read from Mihomo give the relics they were read from.
func TestCloudAvatarWritesEverySource(t *testing.T) {
	a := pluginApp(t)
	shared := sharedSeele(t, a)
	want := map[string][]string{}
	for slot, ids := range sharedRelics {
		want[slot] = slices.Sorted(slices.Values(ids))
	}
	if got := writtenRelics(t, a, shared); !reflect.DeepEqual(got, want) {
		t.Fatalf("ark panel = %v, want %v", got, want)
	}
	// The same relics as Mihomo shows them: counts less one, values cut.
	panel := shownPanel(shared, "mihomo", map[string][2]any{"atk": {"7.7%", 1}, "speed": {"2.3", 0}, "2speed": {"4.6", 1}, "cdmg": {"18.1%", 2}, "cpct": {"2.5%", 0},
		"hpPlus": {"71", 1}, "effPct": {"4.3%", 0}, "stance": {"5.1%", 0}})
	if got := writtenRelics(t, a, panel); !reflect.DeepEqual(got, want) {
		t.Fatalf("Mihomo panel = %v, want %v", got, want)
	}
	// The file is the one miao saves: the player's name, level and last
	// refresh, and each character under the name miao gives its service,
	// with the time it was kept.
	official := shownPanel(shared, "mihoyo", officialShown)
	kept := map[string]CharacterPanel{"100000001": shared, "100000002": panel, "100000003": official}
	sources := map[string]string{"100000001": "share", "100000002": "homo", "100000003": "mysPanelHSR"}
	for uid, source := range sources {
		saved, err := a.Profiles.Keep(uid, []CharacterPanel{kept[uid]}, "test", &ShowcaseProfile{Nickname: "开拓者", Level: 70})
		if err != nil {
			t.Fatal(err)
		}
		content, count, err := a.playerDataFile(t.Context(), saved)
		var file struct {
			UID     string `json:"uid"`
			Name    string `json:"name"`
			Level   int    `json:"level"`
			Info    bool   `json:"info"`
			Profile int64  `json:"_profile"`
			Avatars map[string]struct {
				Source string `json:"_source"`
				Time   int64  `json:"_time"`
			} `json:"avatars"`
		}
		if err == nil {
			err = json.Unmarshal(content, &file)
		}
		avatar := file.Avatars["1102"]
		if err != nil || count != 1 || file.UID != uid || file.Name != "开拓者" || file.Level != 70 || file.Info || file.Profile != saved.RefreshedAtMS || avatar.Source != source || avatar.Time != saved.Panels["1102"].UpdatedAtMS {
			t.Fatalf("%s file = %s %d %v", uid, content, count, err)
		}
		// 导入面板数据 reads the uploaded file back.
		data, err := cloudPlayerObject(string(content))
		if avatars, e := cleanCloudPlayer(uid, data); err != nil || e != nil || len(avatars) != 1 {
			t.Fatalf("%s read back = %v %v %v", uid, avatars, err, e)
		}
	}
}

func TestCloudCapturePromotionResolutionUsesPinnedBaseAttributes(t *testing.T) {
	for _, key := range []string{"sr_1102"} {
		engine := calcEngine(t)
		metadata := engine.Metadata()
		var cases []struct {
			Key   string
			Input BuildProfile
		}
		if err := json.Unmarshal(pluginFile(t, "internal/assets/testdata/calc-vectors.json"), &cases); err != nil {
			t.Fatal(err)
		}
		var profile BuildProfile
		var record reference.Character
		for _, c := range cases {
			if c.Key == key && c.Input.Rank == 0 {
				profile = c.Input
				break
			}
		}
		for _, r := range metadata.Characters {
			if r.Key == key {
				record = r
				break
			}
		}
		profile.Promote = nil
		profile.Weapon.Promote = nil
		cp, wp, err := resolveCloudPromotions(t.Context(), engine, record, profile)
		if err != nil || cp != 6 || wp != 6 {
			t.Fatal(key, cp, wp, err)
		}
	}
}
