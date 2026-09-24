package app

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/reference"
)

func TestOfficialCloudRollRestoration(t *testing.T) {
	rolls, err := cloudRolls(testGame(t), 5, PanelStat{Key: "cpct", Value: "6.48%"}, 2)
	if err != nil || len(rolls) != 1 {
		t.Fatal(rolls, err)
	}
	if _, err = cloudRolls(testGame(t), 5, PanelStat{Key: "cpct", Value: "1000%"}, 1); err == nil {
		t.Fatal("unrepresentable stat exported")
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
}

// 导出面板 writes every kept panel back as miao keeps it: a panel ark handed
// over and one read from Mihomo give the relics they were read from.
func TestCloudAvatarWritesEverySource(t *testing.T) {
	a := pluginApp(t)
	attrIds := map[string][]string{"1": {"5,2,2", "7,1,1", "9,3,4", "8,1,0"}, "2": {"1,2,1", "7,2,2", "10,1,2", "12,1,0"}}
	player := cloudObject(t, `{"uid":"100000001","avatars":{"1102":{"id":1102,"elem":"quantum","level":80,"promote":6,"cons":0,"talent":{"a":6,"e":10,"t":10,"q":10},"trees":["1102101"],
		"weapon":{"id":23001,"level":80,"promote":6,"affix":1},"artis":{"1":{"level":15,"star":5,"id":61011,"mainId":1,"attrIds":["5,2,2","7,1,1","9,3,4","8,1,0"]},
		"2":{"level":15,"star":5,"id":61012,"mainId":1,"attrIds":["1,2,1","7,2,2","10,1,2","12,1,0"]}}}}}`)
	panels := a.arkPlayerPanels(t.Context(), "100000001", player)
	if len(panels) != 1 {
		t.Fatalf("panels = %+v", panels)
	}
	written := func(panel CharacterPanel) map[string][]string {
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
	want := map[string][]string{}
	for slot, ids := range attrIds {
		want[slot] = slices.Sorted(slices.Values(ids))
	}
	if got := written(panels[0]); !reflect.DeepEqual(got, want) {
		t.Fatalf("ark panel = %v, want %v", got, want)
	}
	// The same relics as Mihomo shows them: counts less one, values cut.
	shown := map[string][2]any{"atk": {"7.7%", 1}, "speed": {"2.3", 0}, "cdmg": {"18.1%", 2}, "cpct": {"2.5%", 0}, "hpPlus": {"71", 1}, "effPct": {"4.3%", 0}, "stance": {"5.1%", 0}}
	panel := panels[0]
	panel.Source = "mihomo"
	panel.Equipment = slices.Clone(panel.Equipment)
	for index, piece := range panel.Equipment {
		piece.Sub = slices.Clone(piece.Sub)
		for i, stat := range piece.Sub {
			value := shown[stat.Key]
			if piece.Slot == 2 && stat.Key == "speed" {
				value = [2]any{"4.6", 1}
			}
			piece.Sub[i].Value, piece.Sub[i].Times = value[0].(string), value[1].(int)
		}
		panel.Equipment[index] = piece
	}
	if got := written(panel); !reflect.DeepEqual(got, want) {
		t.Fatalf("Mihomo panel = %v, want %v", got, want)
	}
	// The file is the one miao saves: the player's name, level and last
	// refresh, and each character under the name miao gives its service,
	// with the time it was kept.
	official := panels[0]
	official.Source = "mihoyo"
	kept := map[string]CharacterPanel{"100000001": panels[0], "100000002": panel, "100000003": official}
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
