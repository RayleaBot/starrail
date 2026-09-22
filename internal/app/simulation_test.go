package app

import (
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSimulationReferenceProbabilityVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/simulation-probability-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Game     string `json:"game"`
		Kind     string `json:"kind"`
		Five     int    `json:"five"`
		Weekly   int    `json:"weekly"`
		Expected int    `json:"expected"`
	}
	if json.Unmarshal(raw, &cases) != nil {
		t.Fatal("bad vectors")
	}
	for _, c := range cases {
		if actual := simulationProbability(c.Kind, c.Five); actual != c.Expected {
			t.Fatalf("%s/%s five=%d week=%d: %d != %d", c.Game, c.Kind, c.Five, c.Weekly, actual, c.Expected)
		}
	}
}
func TestSimulationQuotaReplayResetAndBusinessDay(t *testing.T) {
	s := &SimulationStore{Game: "starrail", Deck: testGame(t).Data.Simulation, Directory: t.TempDir(), Random: func(n int) int { return n - 1 }}
	now := time.Date(2026, 9, 19, 19, 59, 0, 0, time.UTC)
	if _, err := s.Action("a", "simulation.configure", map[string]any{"daily_limit": 10}, now); err != nil {
		t.Fatal(err)
	}
	batch, err := s.Action("a", "simulation.draw", map[string]any{"count": 10, "request_ref": "first"}, now)
	if err != nil || batch["daily_remaining"] != 0 {
		t.Fatal(err)
	}
	restarted := &SimulationStore{Game: s.Game, Deck: s.Deck, Directory: s.Directory, Random: s.Random}
	again, err := restarted.Action("a", "simulation.draw", map[string]any{"count": 10, "request_ref": "first"}, now)
	if err != nil || again["total"] != int64(10) {
		t.Fatal("restart replay duplicated draws", err)
	}
	if _, err = s.Action("a", "simulation.draw", map[string]any{"count": 1, "request_ref": "first"}, now); err == nil || PublicError(err).Code != "plugin.game_simulation_conflict" {
		t.Fatal("mismatched replay accepted", err)
	}
	if _, err = s.Action("a", "simulation.reset", map[string]any{"confirm": true}, now); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Action("a", "simulation.draw", map[string]any{"count": 1, "request_ref": "new"}, now); err == nil || PublicError(err).Code != "plugin.game_simulation_quota" {
		t.Fatal("reset bypassed quota", err)
	}
	if result, err := s.Action("a", "simulation.draw", map[string]any{"count": 10, "request_ref": "next-day"}, now.Add(time.Minute)); err != nil || result["daily_used"] != 10 {
		t.Fatal("04:00 reset failed", err)
	}
	if result, err := s.Action("b", "simulation.status", nil, now); err != nil || result["total"] != int64(0) {
		t.Fatal("scope shared simulated inventory")
	}
}
func TestSimulationConcurrentAdmissionDoesNotPartiallySpend(t *testing.T) {
	s := &SimulationStore{Game: "starrail", Deck: testGame(t).Data.Simulation, Directory: t.TempDir(), Random: func(n int) int { return n - 1 }}
	now := time.Now()
	_, _ = s.Action("a", "simulation.configure", map[string]any{"daily_limit": 10}, now)
	var wg sync.WaitGroup
	var completed atomic.Int32
	for i := range 20 {
		wg.Go(func() {
			if _, err := s.Action("a", "simulation.draw", map[string]any{"count": 1, "request_ref": strconv.Itoa(i)}, now); err == nil {
				completed.Add(1)
			}
		})
	}
	wg.Wait()
	if completed.Load() != 10 {
		t.Fatal("quota raced", completed.Load())
	}
	status, err := s.Action("a", "simulation.status", nil, now)
	if err != nil || len(status["history"].([]SimulationDraw)) != 10 {
		t.Fatal("partial or missing batch", err)
	}
}
func TestSimulationPoolStateStaysIndependent(t *testing.T) {
	deck := *testGame(t).Data.Simulation
	state := newSimulationState(deck)
	s := &SimulationStore{Game: "starrail", Deck: &deck, Random: func(n int) int { return n - 1 }}
	selection := state.Selections["weapon"]
	if err := s.selectPool(&state, selection); err != nil {
		t.Fatal(err)
	}
	state.Pools["weapon"].Five = 80
	if first := s.drawOne(&state, selection, time.Now()); first.Rarity != 5 {
		t.Fatal("reference hard guarantee not applied")
	}
	if state.Pools["character"].Five != 0 || state.Pools["standard"].Five != 0 {
		t.Fatal("pity shared across pool types")
	}
}

func TestSimulationLimitTextFollowsUpstream(t *testing.T) {
	now := time.Date(2026, 9, 21, 20, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	yesterday := now.Add(-20 * time.Hour).UnixMilli()
	history := []SimulationDraw{
		{Name: "瓦尔特", Rarity: 5, Kind: "weapon", Interval: 90, TimeMS: yesterday},
		{Name: "于夜色中", Rarity: 5, Kind: "weapon", Interval: 60, TimeMS: now.UnixMilli()},
		{Name: "天倾", Rarity: 3, Kind: "weapon", TimeMS: now.UnixMilli()},
		{Name: "物穰", Rarity: 3, Kind: "standard", TimeMS: now.UnixMilli()},
	}
	// StarRail-plugin counts the pool's draws and five-stars today.
	status := map[string]any{"history": history}
	if got := simulationLimitText("weapon", status, now); got != "今日抽已抽2抽，其中1个五星，明日再来吧" {
		t.Errorf("weapon = %q", got)
	}
	if got := simulationLimitText("standard", status, now); got != "今日抽已抽1抽，没有五星，明日再来吧" {
		t.Errorf("standard = %q", got)
	}
}
