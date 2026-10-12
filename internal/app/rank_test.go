package app

import (
	"strings"
	"sync"
	"testing"
)

func TestRankKeepsOneEntryPerUIDAndCharacter(t *testing.T) {
	store := &GroupStore{Directory: t.TempDir()}
	scope := GroupScope{Protocol: "onebot11", BotID: "bot", GroupID: "group"}
	base := RankEntry{ActorID: "user1", UID: "100000001", CharacterID: "1102", Score: 100, Panel: &CharacterPanel{ID: "1102"}, UpdatedAtMS: 5}
	var wg sync.WaitGroup
	for range 6 {
		wg.Go(func() {
			if err := store.Submit(scope, base); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	// The same member's second UID ranks on its own, as upstream ranks UIDs.
	base.UID = "100000002"
	if err := store.Submit(scope, base); err != nil {
		t.Fatal(err)
	}
	data, _ := store.Read(scope)
	if len(data.Rank) != 2 || data.RankSinceMS != 5 {
		t.Fatalf("rank = %d entries since %d", len(data.Rank), data.RankSinceMS)
	}
	base.Score = -1
	if err := store.Submit(scope, base); err == nil {
		t.Fatal("invalid score accepted")
	}
}

func TestRankEntriesFollowMiao(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"1","entries":[{"id":"1102","name":"希儿","kind":"character","rarity":5},{"id":"1001","name":"三月七","kind":"character","rarity":4}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Catalog: catalog}
	entry := func(uid, id string, score, damage float64) RankEntry {
		e := RankEntry{UID: uid, CharacterID: id, Score: score, Panel: &CharacterPanel{ID: id}}
		if damage > 0 {
			e.Damage = &RankDamage{Value: damage}
		}
		return e
	}
	all := []RankEntry{entry("3", "1102", 180, 50000), entry("1", "1102", 220, 40000), entry("2", "1102", 150, 0),
		entry("1", "1001", 200, 30000), {UID: "9", CharacterID: "1102", Score: 999}}
	uids := func(entries []RankEntry) string {
		out := []string{}
		for _, e := range entries {
			out = append(out, e.UID+":"+e.CharacterID)
		}
		return strings.Join(out, ",")
	}
	// Damage ranks only panels with a default damage; entries kept without a
	// panel (from before panels were stored) do not rank.
	if got := uids(a.rankEntries(all, "1102", "dmg")); got != "3:1102,1:1102" {
		t.Errorf("dmg = %s", got)
	}
	if got := uids(a.rankEntries(all, "1102", "mark")); got != "1:1102,3:1102,2:1102" {
		t.Errorf("mark = %s", got)
	}
	// Every character's best, by UID then rarity.
	if got := uids(a.rankEntries(all, "", "mark")); got != "1:1102,1:1001" {
		t.Errorf("best = %s", got)
	}
	if len(a.rankEntries(all, "1102", "crit")) != 0 {
		t.Error("crit ranking has no upstream values")
	}
	for word, mode := range map[string]string{"希儿排名": "dmg", "希儿遗器排行": "mark", "最高分希儿": "mark", "希儿词条排名": "valid", "希儿双爆排名": "crit"} {
		if got := rankMode(word); got != mode {
			t.Errorf("rankMode(%s) = %s, want %s", word, got, mode)
		}
	}
	if name := rankStrip.ReplaceAllString("群内最高分希儿", ""); name != "希儿" {
		t.Errorf("name = %s", name)
	}
}

func TestRankScoresTakesTheDefaultDetail(t *testing.T) {
	a := &App{}
	total := 180.5
	expected := 12345.0
	damage := &BuildResult{Baseline: BuildScenario{Results: []BuildSkillResult{{Title: "E伤害", Expected: new(float64)}, {Title: "Q伤害", Expected: &expected, Default: true}, {Title: "治疗", Text: "12.5%"}}}}
	entry := RankEntry{}
	a.rankScores(&entry, CharacterPanel{ID: "1", TotalScore: &total, ScoredEquipment: 5, ScoreDetail: &ScoreDetail{Grade: "SS"}}, damage)
	if entry.Score != total || entry.Grade != "SS" || entry.Damage == nil || entry.Damage.Title != "Q伤害" || entry.Damage.Value != expected || entry.Panel == nil {
		t.Fatalf("entry = %+v", entry)
	}
	damage.Baseline.Results[1].Default, damage.Baseline.Results[2].Default = false, true
	a.rankScores(&entry, CharacterPanel{ID: "1"}, damage)
	if entry.Score != 0 || entry.Damage == nil || entry.Damage.Value != 12.5 || entry.Damage.Text != "12.5%" {
		t.Fatalf("text entry = %+v", entry)
	}
}

func TestRankSetNameFollowsMiao(t *testing.T) {
	catalog := Catalog{SetAbbrs: map[string]string{"炽烈的炎之魔女": "魔女", "角斗士的终幕礼": "角斗"}}
	panel := func(sets ...string) CharacterPanel {
		equipment := []PanelEquipment{}
		for _, set := range sets {
			equipment = append(equipment, PanelEquipment{SetName: set})
		}
		return CharacterPanel{Equipment: equipment}
	}
	for want, sets := range map[string][]string{
		// One set whose name and count fit in seven characters keeps its name.
		"深林的记忆4": {"深林的记忆", "深林的记忆", "深林的记忆", "深林的记忆", "角斗士的终幕礼"},
		"魔女4":    {"炽烈的炎之魔女", "炽烈的炎之魔女", "炽烈的炎之魔女", "炽烈的炎之魔女"},
		// Two sets use abbreviations; a set without one keeps its name.
		"魔女2+深林的记忆2": {"炽烈的炎之魔女", "炽烈的炎之魔女", "深林的记忆", "深林的记忆", "角斗士的终幕礼"},
		"":           {"炽烈的炎之魔女", "深林的记忆"},
	} {
		if got := RankSetName(catalog, panel(sets...)); got != want {
			t.Errorf("RankSetName(%v) = %q, want %q", sets, got, want)
		}
	}
	if got := RankDamageTitle("Q·万雷· 凝渊伤害测试标题"); got != "Q万雷凝渊伤害测试标题" {
		t.Errorf("title = %q", got)
	}
	if got := RankDamageTitle("开Q后的长长的重击蒸发伤害"); got != "开Q后的长长的重击蒸发" {
		t.Errorf("title = %q", got)
	}
}
