package app

import (
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/RayleaBot/plugin-starrail/internal/localdata"
)

// APICharacter is a character StarRail-plugin's panel service returned, as its
// data/panel store keeps it: the fields its panel page, character list and
// damage module read, under the service's names, which the damage module
// reads them by. IsNew and LastUpdateTime are what its updateData marks.
type APICharacter struct {
	AvatarID   int    `json:"avatarId"`
	Name       string `json:"name"`
	Level      int    `json:"level"`
	Rank       int    `json:"rank"`
	DamageType string `json:"damage_type"`
	// Properties are the panel's totals by the service's keys, as hpBase
	// and criticalChance.
	Properties     map[string]float64 `json:"properties"`
	Equipment      APIEquipment       `json:"equipment"`
	Relics         []APIRelic         `json:"relics"`
	RelicSets      []APIRelicSet      `json:"relic_sets"`
	Behaviors      []APIBehavior      `json:"behaviorList"`
	IsNew          bool               `json:"is_new"`
	LastUpdateTime int64              `json:"lastUpdateTime,omitempty"`
}

// APIEquipment is the light cone; its zero value, which encodes as {}, is
// none.
type APIEquipment struct {
	ID    int    `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Level int    `json:"level,omitempty"`
	Rank  int    `json:"rank,omitempty"`
	// Rarity is a name ending in the rarity, as Rarity5.
	Rarity  string  `json:"rarity,omitempty"`
	HP      float64 `json:"hp,omitempty"`
	Attack  float64 `json:"atk,omitempty"`
	Defence float64 `json:"def,omitempty"`
}

type APIRelic struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
	// MaxLevel is three times the rarity.
	MaxLevel  int     `json:"max_level"`
	MainName  string  `json:"main_affix_name"`
	MainTag   string  `json:"main_affix_tag"`
	MainValue float64 `json:"main_affix_value"`
	// Affixes are the substats, a fraction when below 1.
	Affixes []APIAffix `json:"sub_affix_id"`
	Score   float64    `json:"score,omitempty"`
}

type APIAffix struct {
	Name  string  `json:"name"`
	Tag   string  `json:"tag"`
	Value float64 `json:"value"`
	// Count is how often the substat rolled; Weight how much the service's
	// scoring values it.
	Count  int     `json:"cnt"`
	Weight float64 `json:"weight,omitempty"`
}

// APIRelicSet is a set bonus in effect, named 2件套 or 4件套.
type APIRelicSet struct {
	ID   int    `json:"set_id"`
	Name string `json:"set_name"`
}

// APIBehavior is a trace: the first ones are the skills, in the order basic
// attack, skill, ultimate, talent and technique.
type APIBehavior struct {
	ID       int    `json:"id"`
	Level    int    `json:"level"`
	Type     string `json:"type"`
	Anchor   string `json:"anchor"`
	Icon     string `json:"icon"`
	MaxLevel int    `json:"max_level"`
}

// PanelAPIStore keeps the characters each UID's panel service queries
// returned, as StarRail-plugin's readData and saveData keep data/panel.
type PanelAPIStore struct {
	mu        sync.Mutex
	Directory string
}

func (s *PanelAPIStore) file(uid string) (string, error) {
	if !uidPattern.MatchString(uid) {
		return "", gameError("input_invalid", "UID 格式不正确。")
	}
	return filepath.Join(s.Directory, uid+".json"), nil
}

func (s *PanelAPIStore) read(uid string) ([]APICharacter, error) {
	file, err := s.file(uid)
	if err != nil {
		return nil, err
	}
	characters := []APICharacter{}
	return characters, localdata.Read(file, &characters)
}

func (s *PanelAPIStore) Read(uid string) ([]APICharacter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(uid)
}

// Keep merges a query's characters into the UID's as updateData does: the
// returned ones, marked new and stamped, go first and replace their older
// entries; the others stay, no longer new.
func (s *PanelAPIStore) Keep(uid string, fresh []APICharacter, now time.Time) ([]APICharacter, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept, err := s.read(uid)
	if err != nil {
		return nil, err
	}
	merged := make([]APICharacter, 0, len(fresh)+len(kept))
	for _, character := range fresh {
		character.IsNew, character.LastUpdateTime = true, now.UnixMilli()
		merged = append(merged, character)
	}
	for _, character := range kept {
		if !slices.ContainsFunc(fresh, func(item APICharacter) bool { return item.AvatarID == character.AvatarID }) {
			character.IsNew = false
			merged = append(merged, character)
		}
	}
	file, _ := s.file(uid)
	return merged, localdata.Write(file, merged)
}
