package app

import (
	"strings"
	"testing"

	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

func TestGachaTextListsEachTopPullWithUpAndAverages(t *testing.T) {
	game := Game{ID: "starrail", Name: "星铁", Data: &GameData{Resources: GameResources{Pools: []PoolInfo{{From: "2026-01-01 00:00:00", To: "2026-01-21 23:59:59", Characters5: []string{"希儿"}}}}}}
	records := []gacha.Record{}
	add := func(id, name, rank, when string) {
		records = append(records, gacha.Record{ID: id, Name: name, Rank: rank, GachaType: "11", Time: when})
	}
	for index := range 9 {
		add(string(rune('a'+index)), "物穰", "3", "2026-01-02 00:00:00")
	}
	add("j", "三月七", "4", "2026-01-02 00:00:00")
	add("k", "瓦尔特", "5", "2026-01-02 00:00:01")
	add("l", "物穰", "3", "2026-01-03 00:00:00")
	add("m", "希儿", "5", "2026-01-03 00:00:01")
	view := GachaView(game, gacha.Archive{UID: "100000001", Records: records})
	text := view.Text()
	for _, want := range []string{"瓦尔特 11+", "希儿 2 UP", "UP 平均：13.0 抽", "1 个 · 平均 13.0 抽"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in\n%s", want, text)
		}
	}
}
