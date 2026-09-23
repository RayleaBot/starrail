package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

// panel_api counts from 1 as upstream's panelApi.yaml default; a value the
// list does not have reads as the first service.
func TestPanelAPISettingPicksAService(t *testing.T) {
	a := &App{Game: Game{PanelAPIs: []string{"https://one.example/v1/info/", "https://two.example/v1/info/"}}}
	for setting, want := range map[any]int{nil: 1, 2: 2, 0: 1, 3: 1, "x": 1} {
		event := &rayleabot.EventContext{Config: map[string]any{}}
		if setting != nil {
			event.Config["panel_api"] = setting
		}
		if index, api := a.panelAPI(event); index != want || apiHost(api) != []string{"one.example", "two.example"}[want-1] {
			t.Errorf("panel_api %v picked %d %s, want %d", setting, index, api, want)
		}
	}
}

// A synthetic answer in the service's shape. The assist character is not in
// the showcase, so it goes first; entries without an ID are dropped, and
// only the fields the pages and damage module read are kept, checked.
const syntheticAPIAnswer = `{"playerDetailInfo":{"isDisplayAvatarList":true,
	"assistAvatar":{"avatarId":8002,"name":"{nickname}","level":80,"rank":6,"damage_type":"Physical"},
	"displayAvatars":[
		{"avatarId":1102,"name":"希儿","level":80,"rank":2,"damage_type":"Quantum",
			"properties":{"hpBase":931.392,"attackBase":640.332,"criticalChance":0.673,"quantumAdded":0.388,"unknown":1,"speedBase":"115"},
			"equipment":{"id":23001,"name":"于夜色中","level":80,"rank":1,"rarity":"CombatPowerLightconeRarity5","hp":1058.4,"atk":582.12,"def":463.05},
			"relics":[{"id":61011,"name":"野穗伴行的快枪手","level":15,"max_level":15,"main_affix_name":"生命值","main_affix_tag":"HPDelta","main_affix_value":705.6,"score":21.4,
				"sub_affix_id":[{"name":"暴击率","tag":"CriticalChanceBase","value":0.0648,"cnt":2,"weight":1},{"name":"速度","tag":"<SpeedDelta>","value":4.6,"cnt":2}]}],
			"relic_sets":[{"set_id":"101","set_name":"2件套"}],
			"behaviorList":[{"id":1102001,"level":6,"type":"普攻","anchor":"Point01","icon":"/icon/skill/1102_basic_atk.png","max_level":6},
				{"id":1102101,"level":1,"anchor":"Point09","icon":"../../secret.png"}]},
		{"name":"无编号"}]}}`

func TestParseAPIAnswerReadsTheShowcase(t *testing.T) {
	characters, err := parseAPIAnswer(decoded(t, syntheticAPIAnswer), "100000001")
	if err != nil {
		t.Fatal(err)
	}
	if len(characters) != 2 || characters[0].AvatarID != 8002 || characters[0].Name != "开拓者" || characters[1].AvatarID != 1102 {
		t.Fatalf("characters = %+v", characters)
	}
	seele := characters[1]
	if seele.Properties["hpBase"] != 931.392 || seele.Properties["speedBase"] != 115 || len(seele.Properties) != 5 {
		t.Errorf("properties = %v", seele.Properties)
	}
	if seele.Equipment.ID != 23001 || seele.Equipment.Rarity != "CombatPowerLightconeRarity5" || seele.Equipment.Defence != 463.05 {
		t.Errorf("equipment = %+v", seele.Equipment)
	}
	relic := seele.Relics[0]
	if relic.MaxLevel != 15 || relic.MainTag != "HPDelta" || relic.Score != 21.4 || relic.Affixes[0].Weight != 1 || relic.Affixes[1].Tag != "" || relic.Affixes[1].Count != 2 {
		t.Errorf("relic = %+v", relic)
	}
	if seele.RelicSets[0] != (APIRelicSet{ID: 101, Name: "2件套"}) {
		t.Errorf("sets = %+v", seele.RelicSets)
	}
	if seele.Behaviors[0].Icon != "/icon/skill/1102_basic_atk.png" || seele.Behaviors[0].Type != "普攻" || seele.Behaviors[1].Icon != "" || seele.Behaviors[1].Anchor != "Point09" {
		t.Errorf("behaviors = %+v", seele.Behaviors)
	}
	// An assist the showcase also has is not repeated.
	answer := decoded(t, `{"playerDetailInfo":{"isDisplayAvatarList":true,"assistAvatar":{"avatarId":1102},"displayAvatars":[{"avatarId":1001},{"avatarId":1102}]}}`)
	if characters, err = parseAPIAnswer(answer, "100000001"); err != nil || len(characters) != 2 || characters[0].AvatarID != 1001 {
		t.Errorf("characters = %+v %v", characters, err)
	}
}

func TestParseAPIAnswerRepliesAsUpstream(t *testing.T) {
	for raw, want := range map[string]string{
		`{"detail":"UID 不存在"}`:                               "UID 不存在",
		`{"detail":{"reason":1}}`:                            "uid:100000001未查询到任何数据",
		`{"status":"ok"}`:                                    "uid:100000001未查询到任何数据",
		`{"playerDetailInfo":{"isDisplayAvatarList":false}}`: "uid:100000001更新面板失败\n可能是角色展柜未开启或者该用户不存在",
		`{"playerDetailInfo":null}`:                          "uid:100000001更新面板失败\n可能是角色展柜未开启或者该用户不存在",
	} {
		if _, err := parseAPIAnswer(decoded(t, raw), "100000001"); friendlyError(err) != want {
			t.Errorf("%s replied %q, want %q", raw, friendlyError(err), want)
		}
	}
}

// Keep merges as updateData: the characters a query returned go first,
// marked new and stamped, replacing their older entries; the rest stay, no
// longer new.
func TestPanelAPIStoreMergesAsUpdateData(t *testing.T) {
	store := &PanelAPIStore{Directory: t.TempDir()}
	first := time.UnixMilli(1_700_000_000_000)
	if _, err := store.Keep("100000001", []APICharacter{{AvatarID: 1001, Rank: 1}, {AvatarID: 1102}}, first); err != nil {
		t.Fatal(err)
	}
	merged, err := store.Keep("100000001", []APICharacter{{AvatarID: 1102, Rank: 2}, {AvatarID: 1205}}, first.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	kept, err := store.Read("100000001")
	if err != nil {
		t.Fatal(err)
	}
	ids := []int{}
	for _, character := range kept {
		ids = append(ids, character.AvatarID)
	}
	if len(merged) != 3 || len(kept) != 3 || ids[0] != 1102 || ids[1] != 1205 || ids[2] != 1001 {
		t.Fatalf("kept %v", ids)
	}
	if !kept[0].IsNew || kept[0].Rank != 2 || kept[0].LastUpdateTime != first.Add(time.Hour).UnixMilli() || kept[2].IsNew || kept[2].LastUpdateTime != first.UnixMilli() {
		t.Errorf("kept = %+v", kept)
	}
	if _, err := store.Read("../x"); err == nil {
		t.Error("a path was read as a UID")
	}
}

// The query goes to the chosen service with upstream's headers; an answer
// starts the one-minute gap, a failed request does not.
func TestPanelAPIClientSignsAndKeepsTheGap(t *testing.T) {
	var asked []*http.Request
	reply := `{"playerDetailInfo":{"isDisplayAvatarList":true,"displayAvatars":[{"avatarId":1102}]}}`
	failing := false
	client := &PanelAPIClient{HTTP: doerFunc(func(req *http.Request) (*http.Response, error) {
		asked = append(asked, req)
		if failing {
			return nil, errors.New("offline")
		}
		return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(reply))}, nil
	})}
	sign := func(_ context.Context, uid string) (string, error) { return "stub," + uid, nil }
	failing = true
	if _, err := client.Query(context.Background(), sign, "https://one.example/v1/info/", "100000001"); friendlyError(err) != "UID:100000001更新面板失败\n面板服务连接超时，请稍后重试" {
		t.Fatalf("a failed request replied %v", err)
	}
	failing = false
	characters, err := client.Query(context.Background(), sign, "https://one.example/v1/info/", "100000001")
	if err != nil || len(characters) != 1 {
		t.Fatalf("characters = %v %v", characters, err)
	}
	last := asked[len(asked)-1]
	if last.URL.String() != "https://one.example/v1/info/100000001" || last.Header.Get("x-request-sr") != "stub,100000001" || last.Header.Get("library") != "hewang1an" {
		t.Errorf("request = %s %v", last.URL, last.Header)
	}
	if _, err = client.Query(context.Background(), sign, "https://one.example/v1/info/", "100000001"); friendlyError(err) != "查询过于频繁，请60秒后重试~" {
		t.Errorf("a second query replied %v", err)
	}
	client.queried["100000001"] = time.Now().Add(-30*time.Second - 500*time.Millisecond)
	if _, err = client.Query(context.Background(), sign, "https://one.example/v1/info/", "100000001"); friendlyError(err) != "查询过于频繁，请30秒后重试~" {
		t.Errorf("a later query replied %v", err)
	}
	client.queried["100000001"] = time.Now().Add(-time.Minute)
	if _, err = client.Query(context.Background(), sign, "https://one.example/v1/info/", "100000001"); err != nil {
		t.Errorf("a query after the gap failed: %v", err)
	}
	if len(asked) != 3 {
		t.Errorf("%d requests were sent", len(asked))
	}
}

// handleBehaviorList lists the first five traces, raised by each eidolon up
// to the character's, with their icons, and drops the technique.
func TestAPISkillsFollowHandleBehaviorList(t *testing.T) {
	character := APICharacter{AvatarID: 1001, Rank: 3, Behaviors: []APIBehavior{
		{ID: 1001001, Level: 6, Type: "普攻"}, {ID: 1001002, Level: 10, Type: "战技"}, {ID: 1001003, Level: 10, Type: "终结技"},
		{ID: 1001004, Level: 10, Type: "天赋"}, {ID: 1001007, Level: 1, Type: "秘技"}, {ID: 1001101, Level: 1, Type: "天赋"}}}
	raises := map[string][]RankSkillUp{"100103": {{ID: "100103", Num: 2}, {ID: "100101", Num: 1}}, "100105": {{ID: "100102", Num: 2}}}
	skills := apiSkills(character, raises)
	if len(skills) != 4 {
		t.Fatalf("skills = %+v", skills)
	}
	for i, want := range []struct {
		level int
		file  string
	}{{7, "1001_basic_atk.png"}, {10, "1001_skill.png"}, {12, "1001_ultimate.png"}, {10, "1001_talent.png"}} {
		if skills[i].Level != want.level || skills[i].File != want.file {
			t.Errorf("skill %d = %d %s, want %d %s", i, skills[i].Level, skills[i].File, want.level, want.file)
		}
	}
}
