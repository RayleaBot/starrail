package app

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/RayleaBot/plugin-starrail/internal/pluginmeta"
)

// commandSet maps the command word the host delivered back to this plugin's
// declaration. The host matched the word against the same triggers and only
// passes the word on, so the plugin repeats the lookup in the host's order:
// declarations in manifest order, exact names compared as is, patterns with
// MatchString. Handlers dispatch on the command ID, so display names and
// trigger words can follow upstream without touching them.
type commandSet struct {
	commands []declaredCommand
}

type declaredCommand struct {
	id      string
	names   []string
	pattern *regexp.Regexp
}

func newCommandSet(manifest pluginmeta.Manifest) (commandSet, error) {
	set := commandSet{}
	for _, command := range manifest.Commands {
		declared := declaredCommand{id: command.ID}
		switch command.Trigger.Type {
		case "exact":
			declared.names = command.Trigger.Names
		case "pattern":
			expression, err := regexp.Compile(command.Trigger.Pattern)
			if err != nil {
				return set, fmt.Errorf("command %s has an invalid pattern: %w", command.ID, err)
			}
			declared.pattern = expression
		default:
			continue
		}
		set.commands = append(set.commands, declared)
	}
	return set, nil
}

// resolve returns the command ID and arguments for a delivered command word.
// The named groups of a pattern trigger become the leading arguments, so
// "雷神面板 1000" reaches the panel handler with ["雷神", "1000"].
func (s commandSet) resolve(word string, args []string) (string, []string, bool) {
	word = strings.TrimSpace(word)
	if word == "" {
		return "", args, false
	}
	for _, command := range s.commands {
		if command.pattern == nil {
			for _, name := range command.names {
				if strings.TrimSpace(name) == word {
					return command.id, args, true
				}
			}
			continue
		}
		match := command.pattern.FindStringSubmatch(word)
		if match == nil {
			continue
		}
		leading := []string{}
		for i, group := range command.pattern.SubexpNames() {
			if group != "" && match[i] != "" {
				leading = append(leading, match[i])
			}
		}
		return command.id, append(leading, args...), true
	}
	return "", args, false
}

// matches is whether a word matches a declared command's trigger.
func (s commandSet) matches(id, word string) bool {
	for _, command := range s.commands {
		if command.id == id {
			return command.pattern != nil && command.pattern.MatchString(word) || slices.Contains(command.names, word)
		}
	}
	return false
}

// starRailGacha is StarRail-plugin's gatcha rule, which upstream runs before
// miao's and Yunzai's gacha rules. The 跃迁记录 trigger adds Yunzai's words to
// it, so it is kept here for miaoGacha.
var starRailGacha = regexp.MustCompile(`^(?:角色|光锥|武器|常驻|新手)?(?:跃迁|抽卡)?(?:记录|分析|统计)$`)

// miaoGacha is whether a gacha word draws miao's page: its 喵喵 words, and
// with miao's setting on the plain words StarRail-plugin's rule leaves.
func (s commandSet) miaoGacha(word string, setting bool) bool {
	if !s.matches("gacha-detail", word) && !s.matches("gacha-stat", word) {
		return false
	}
	return strings.HasPrefix(word, "喵喵") || setting && !starRailGacha.MatchString(word)
}

// usage writes a declared usage with the prefix replies use. Manifests start
// usages with the placeholder "#", which the host help menu also replaces.
func (a *App) usage(text string) string {
	if strings.HasPrefix(text, "#") {
		return a.Game.Prefix + text[1:]
	}
	return text
}
