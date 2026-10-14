package app

import (
	"encoding/json"
	"os"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RayleaBot/starrail/internal/localdata"
)

func TestSimulationChatFollowsStartedBanners(t *testing.T) {
	a := pluginApp(t)
	deck := a.Simulation.Deck
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	a.clock = &fakeClock{at: now}
	scope := `["onebot11","bot","private","u","u"]`
	state := newSimulationState(*deck)
	for _, kind := range []string{"character", "weapon"} {
		state.Selections[kind] = SimulationSelection{Kind: kind, BannerID: "removed-banner", Featured: "removed-target"}
	}
	if err := localdata.Write(a.Simulation.filename(scope), state); err != nil {
		t.Fatal(err)
	}
	host := newSDKHost(t, a, nil)
	for word, kind := range map[string]string{"十连": "character", "十连光锥": "weapon"} {
		host.message("*"+word, word)
		saved, err := a.Simulation.read(scope, now)
		if err != nil || len(saved.Requests) == 0 {
			t.Fatal(saved, err)
		}
		selection := saved.Requests[len(saved.Requests)-1].Selection
		if selection != (SimulationSelection{Kind: kind, BannerID: "4.6|上半|2026-09-28 06:00:00"}) {
			t.Fatal(selection)
		}
	}
	for _, tc := range []struct{ at, want string }{
		{"2026-09-28T03:59:59+08:00", "4.5|下半|2026-09-12 12:00:00"},
		{"2026-09-28T05:00:00+08:00", "4.5|下半|2026-09-12 12:00:00"},
		{"2026-09-27T22:00:00Z", "4.6|上半|2026-09-28 06:00:00"},
		{"2026-10-21T04:00:00Z", "4.6|下半|2026-10-21 12:00:00"},
		{"2026-12-01T00:00:00Z", "4.6|下半|2026-10-21 12:00:00"},
	} {
		at, _ := time.Parse(time.RFC3339, tc.at)
		for _, kind := range []string{"character", "weapon"} {
			if got := simulationChatSelection(*deck, kind, at); got.BannerID != tc.want || got.Featured != "" {
				t.Errorf("%s %s: %+v", tc.at, kind, got)
			}
		}
	}
	// Management can still explicitly select a future banner and target.
	selected := SimulationSelection{Kind: "character", BannerID: deck.Banners[0].ID, Featured: deck.Banners[0].Characters5[0]}
	result, err := a.Simulation.Action("management", "simulation.select", asMap(selected), now)
	if err != nil || result["selections"].(map[string]SimulationSelection)["character"] != selected {
		t.Fatal(result, err)
	}
}

func TestSimulationIgnoresSelectionFieldsForStandardAndCollaboration(t *testing.T) {
	deck := *testGame(t).Data.Simulation
	state := newSimulationState(deck)
	s := &SimulationStore{Deck: &deck}
	for _, kind := range []string{"standard", "collab_character", "collab_weapon"} {
		for _, target := range []string{"", "not-an-up", deck.CollabCharacters[0]} {
			if err := s.selectPool(&state, SimulationSelection{Kind: kind, BannerID: "not-a-banner", Featured: target}); err != nil {
				t.Fatal(kind, target, err)
			}
			if state.Selections[kind] != (SimulationSelection{Kind: kind}) {
				t.Fatal(state.Selections[kind])
			}
		}
	}
	for _, kind := range []string{"character", "weapon"} {
		if err := s.selectPool(&state, SimulationSelection{Kind: kind, BannerID: deck.Banners[0].ID, Featured: "not-an-up"}); err == nil {
			t.Fatal("accepted invalid target", kind)
		}
	}
}

func TestSimulationCollabPoolsAndSharedUps(t *testing.T) {
	deck := *testGame(t).Data.Simulation
	state := newSimulationState(deck)
	s := &SimulationStore{Deck: &deck, Random: func(int) int { return 0 }}
	selection := state.Selections["collab_character"]
	if err := s.selectPool(&state, selection); err != nil {
		t.Fatal(err)
	}
	for range 10 {
		draw := s.drawOne(&state, selection, time.Now())
		if !draw.Featured || draw.Item != "character" || !slices.Contains(deck.CollabCharacters, draw.Name) {
			t.Fatal(draw)
		}
	}
	// A lost 50/50 sets only this pool's next-five-star guarantee.
	state.Pools["collab_character"].Five = 90
	s.Random = func(n int) int { return n - 1 }
	draw := s.drawOne(&state, selection, time.Now())
	if draw.Featured || !slices.Contains(deck.FiveCharacters, draw.Name) || !state.Pools["collab_character"].UpFive || *state.Pools["character"] != (SimulationPity{}) {
		t.Fatal(draw, state.Pools)
	}
	// Collaboration four-stars have no rate-up or rate-up guarantee.
	for _, kind := range []string{"collab_character", "collab_weapon"} {
		selection = state.Selections[kind]
		state.Pools[kind].Five, state.Pools[kind].Four = 0, 9
		draw = s.drawOne(&state, selection, time.Now())
		if draw.Rarity != 4 || draw.Featured || state.Pools[kind].UpFour {
			t.Fatal(draw)
		}
	}
	selection = state.Selections["character"]
	selection.Featured = ""
	if err := s.selectPool(&state, selection); err != nil {
		t.Fatal(err)
	}
	banner, _ := simulationBanner(deck, selection)
	// Each UP can be selected when no explicit target is set.
	for index, name := range banner.Characters5 {
		s.Random = func(n int) int {
			if n == 10000 {
				return 0
			}
			return index % n
		}
		draw = s.drawOne(&state, selection, time.Now())
		if !draw.Featured || draw.Name != name {
			t.Fatal(draw, name)
		}
	}
}

func TestSimulationKind(t *testing.T) {
	for word, want := range map[string]string{"十连": "character", "十连光锥": "weapon", "十连常驻": "standard", "十连联动": "collab_character", "十连联动角色": "collab_character", "抽卡联动光锥": "collab_weapon"} {
		if got := simulationKind(word); got != want {
			t.Errorf("%s = %s", word, got)
		}
	}
}

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
