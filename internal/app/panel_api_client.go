package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// PanelAPIScripts are the pinned StarRail-plugin scripts its panel service
// commands run.
type PanelAPIScripts struct {
	// Sign computes a UID's x-request-sr header with getSign's script.
	Sign func(ctx context.Context, uid string) (string, error)
}

// PanelAPIClient queries StarRail-plugin's panel service. As upstream, a UID
// is queried at most once a minute after a query that answered (its
// STAR_RAILWAY:userPanelDataTime key); a query in flight holds its UID, so a
// second one waits and then counts from it.
type PanelAPIClient struct {
	HTTP    HTTPDoer
	mu      sync.Mutex
	queried map[string]time.Time
	flights map[string]chan struct{}
}

const (
	panelAPIGap      = time.Minute
	panelAPIMaxBytes = 4 * 1024 * 1024
)

func panelAPIError(text string) error { return gameError("panel_api_failed", text) }

// Query asks the service at api for the UID's characters, as getPanelData
// does when it queries.
func (c *PanelAPIClient) Query(ctx context.Context, sign func(context.Context, string) (string, error), api, uid string) ([]APICharacter, error) {
	if !uidPattern.MatchString(uid) {
		return nil, gameError("input_invalid", "UID 格式不正确。")
	}
	release, err := c.hold(ctx, uid)
	if err != nil {
		return nil, err
	}
	defer release()
	unreachable := panelAPIError("UID:" + uid + "更新面板失败\n面板服务连接超时，请稍后重试")
	header, err := sign(ctx, uid)
	if err != nil {
		return nil, unreachable
	}
	raw, err := c.request(ctx, api+uid, header)
	if err != nil {
		return nil, unreachable
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var answer map[string]any
	if decoder.Decode(&answer) != nil {
		return nil, unreachable
	}
	c.mu.Lock()
	c.queried[uid] = time.Now()
	c.mu.Unlock()
	return parseAPIAnswer(answer, uid)
}

// hold waits for a query of the UID in flight, keeps the one-minute gap and
// marks the UID in flight until release.
func (c *PanelAPIClient) hold(ctx context.Context, uid string) (func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.flights == nil {
		c.queried, c.flights = map[string]time.Time{}, map[string]chan struct{}{}
	}
	for c.flights[uid] != nil {
		done := c.flights[uid]
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			c.mu.Lock()
			return nil, ctx.Err()
		case <-done:
		}
		c.mu.Lock()
	}
	now := time.Now()
	for other, at := range c.queried {
		if now.Sub(at) >= panelAPIGap {
			delete(c.queried, other)
		}
	}
	if at, ok := c.queried[uid]; ok {
		seconds := int(math.Ceil(float64(panelAPIGap-now.Sub(at)) / float64(time.Second)))
		return nil, panelAPIError("查询过于频繁，请" + strconv.Itoa(seconds) + "秒后重试~")
	}
	done := make(chan struct{})
	c.flights[uid] = done
	return func() {
		c.mu.Lock()
		delete(c.flights, uid)
		close(done)
		c.mu.Unlock()
	}, nil
}

// request sends the query with upstream's headers to the service's own host;
// like upstream it reads the answer whatever the status.
func (c *PanelAPIClient) request(ctx context.Context, address, header string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RayleaBot-GamePlugins/0.1.0")
	req.Header.Set("x-request-sr", header)
	req.Header.Set("library", "hewang1an")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, panelAPIMaxBytes+1))
	if err == nil && len(raw) > panelAPIMaxBytes {
		err = io.ErrUnexpectedEOF
	}
	return raw, err
}

// parseAPIAnswer reads the service's answer as getPanelData does: its error
// detail, the player it found and whether the player shows characters, then
// the assist character unless the showcase has it, and the showcase. The
// answer is untrusted: characters without an ID are left out and every field
// is checked and bounded.
func parseAPIAnswer(answer map[string]any, uid string) ([]APICharacter, error) {
	if detail, ok := answer["detail"]; ok {
		if text := apiText(detail, 200); text != "" {
			return nil, panelAPIError(text)
		}
		return nil, panelAPIError("uid:" + uid + "未查询到任何数据")
	}
	player, ok := answer["playerDetailInfo"]
	if !ok {
		return nil, panelAPIError("uid:" + uid + "未查询到任何数据")
	}
	info := asObject(player)
	if shown, _ := info["isDisplayAvatarList"].(bool); !shown {
		return nil, panelAPIError("uid:" + uid + "更新面板失败\n可能是角色展柜未开启或者该用户不存在")
	}
	characters := []APICharacter{}
	for _, raw := range asList(info["displayAvatars"]) {
		if character, ok := parseAPICharacter(asObject(raw)); ok && len(characters) < 32 {
			characters = append(characters, character)
		}
	}
	if assist, ok := parseAPICharacter(asObject(info["assistAvatar"])); ok && !containsAPICharacter(characters, assist.AvatarID) {
		characters = append([]APICharacter{assist}, characters...)
	}
	return characters, nil
}

func containsAPICharacter(characters []APICharacter, id int) bool {
	for _, character := range characters {
		if character.AvatarID == id {
			return true
		}
	}
	return false
}

// apiProperties are the totals the panel page and damage module read: the
// base and added stats, the rates and each element's damage bonus.
var apiProperties = []string{"hpBase", "hpDelta", "attackBase", "attackDelta", "defenseBase", "defenseDelta", "speedBase", "speedAdd",
	"criticalChance", "criticalDamage", "statusResistance", "statusProbability", "breakDamageAdded", "spRatio", "healRatio",
	"physicalAdded", "fireAdded", "iceAdded", "thunderAdded", "windAdded", "quantumAdded", "imaginaryAdded"}

var (
	apiKey    = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)
	apiAnchor = regexp.MustCompile(`^Point[0-9]{2}$`)
	apiIcon   = regexp.MustCompile(`^/?(?:[A-Za-z0-9_.-]+/)*[A-Za-z0-9_-]+\.(?:png|webp|jpg)$`)
)

func parseAPICharacter(item map[string]any) (APICharacter, bool) {
	character := APICharacter{AvatarID: apiInt(item["avatarId"]), Name: apiText(item["name"], 32), Level: apiInt(item["level"]), Rank: apiInt(item["rank"]),
		Properties: map[string]float64{}, Relics: []APIRelic{}, RelicSets: []APIRelicSet{}, Behaviors: []APIBehavior{}}
	if character.AvatarID <= 0 {
		return APICharacter{}, false
	}
	if character.Name == "{nickname}" || character.Name == "{NICKNAME}" {
		character.Name = "开拓者"
	}
	if kind := apiText(item["damage_type"], 16); apiKey.MatchString(kind) {
		character.DamageType = kind
	}
	properties := asObject(item["properties"])
	for _, key := range apiProperties {
		if value, ok := apiNumber(properties[key]); ok {
			character.Properties[key] = value
		}
	}
	if equipment := asObject(item["equipment"]); apiInt(equipment["id"]) > 0 {
		character.Equipment = APIEquipment{ID: apiInt(equipment["id"]), Name: apiText(equipment["name"], 32), Level: apiInt(equipment["level"]), Rank: apiInt(equipment["rank"]),
			Rarity: apiText(equipment["rarity"], 64)}
		character.Equipment.HP, _ = apiNumber(equipment["hp"])
		character.Equipment.Attack, _ = apiNumber(equipment["atk"])
		character.Equipment.Defence, _ = apiNumber(equipment["def"])
	}
	for _, raw := range limitList(item["relics"], 8) {
		relic := asObject(raw)
		parsed := APIRelic{ID: apiInt(relic["id"]), Name: apiText(relic["name"], 32), Level: apiInt(relic["level"]), MaxLevel: apiInt(relic["max_level"]),
			MainName: apiText(relic["main_affix_name"], 32), MainTag: apiTag(relic["main_affix_tag"]), Affixes: []APIAffix{}}
		parsed.MainValue, _ = apiNumber(relic["main_affix_value"])
		parsed.Score, _ = apiNumber(relic["score"])
		for _, raw := range limitList(relic["sub_affix_id"], 8) {
			affix := asObject(raw)
			sub := APIAffix{Name: apiText(affix["name"], 32), Tag: apiTag(affix["tag"]), Count: apiInt(affix["cnt"])}
			sub.Value, _ = apiNumber(affix["value"])
			sub.Weight, _ = apiNumber(affix["weight"])
			parsed.Affixes = append(parsed.Affixes, sub)
		}
		character.Relics = append(character.Relics, parsed)
	}
	for _, raw := range limitList(item["relic_sets"], 8) {
		set := asObject(raw)
		character.RelicSets = append(character.RelicSets, APIRelicSet{ID: apiInt(set["set_id"]), Name: apiText(set["set_name"], 16)})
	}
	for _, raw := range limitList(item["behaviorList"], 64) {
		behavior := asObject(raw)
		parsed := APIBehavior{ID: apiInt(behavior["id"]), Level: apiInt(behavior["level"]), Type: apiText(behavior["type"], 16), MaxLevel: apiInt(behavior["max_level"])}
		if anchor := apiText(behavior["anchor"], 16); apiAnchor.MatchString(anchor) {
			parsed.Anchor = anchor
		}
		if icon := apiText(behavior["icon"], 256); apiIcon.MatchString(icon) && !strings.Contains(icon, "..") {
			parsed.Icon = icon
		}
		character.Behaviors = append(character.Behaviors, parsed)
	}
	return character, true
}

func limitList(value any, limit int) []any {
	list := asList(value)
	return list[:min(len(list), limit)]
}

// apiInt reads a whole number sent as a number or numeric text; anything
// else, or a number out of range, reads as 0.
func apiInt(value any) int {
	number, ok := apiNumber(value)
	if !ok || number != math.Trunc(number) || math.Abs(number) > 1e12 {
		return 0
	}
	return int(number)
}

func apiNumber(value any) (float64, bool) {
	switch value.(type) {
	case json.Number, string:
		number, err := strconv.ParseFloat(asText(value), 64)
		return number, err == nil && !math.IsInf(number, 0) && !math.IsNaN(number)
	}
	return 0, false
}

func apiTag(value any) string {
	if tag := apiText(value, 64); apiKey.MatchString(tag) {
		return tag
	}
	return ""
}

// apiText reads a text field as plain text of at most limit characters,
// without control characters other than line breaks.
func apiText(value any, limit int) string {
	text, _ := value.(string)
	if number, ok := value.(json.Number); ok {
		text = number.String()
	}
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, text)
	if runes := []rune(text); len(runes) > limit {
		text = string(runes[:limit])
	}
	return text
}
