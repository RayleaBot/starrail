package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/localdata"
)

func (s Subject) identity() Subject       { s.SourceAdapter = ""; return s }
func (s Subject) same(other Subject) bool { return s.identity() == other.identity() }
func (s GroupScope) identity() GroupScope { s.Adapter = ""; return s }

// source_adapter is read-only compatibility. Unresolved old records keep a
// separate migration hint so later writes cannot erase the information needed
// to recover their account. The hint never participates in an identity or route.
func (s Subject) MarshalJSON() ([]byte, error) {
	type plain Subject
	hint := ""
	if s.BotID == "" {
		hint = s.SourceAdapter
	}
	return json.Marshal(struct {
		plain
		LegacyAdapter string `json:"legacy_adapter,omitempty"`
	}{plain(s.identity()), hint})
}
func (s *Subject) UnmarshalJSON(raw []byte) error {
	type plain Subject
	var value struct {
		plain
		LegacyAdapter string `json:"legacy_adapter,omitempty"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*s = Subject(value.plain)
	if s.SourceAdapter == "" {
		s.SourceAdapter = value.LegacyAdapter
	}
	return nil
}
func (s GroupScope) MarshalJSON() ([]byte, error) {
	type plain GroupScope
	hint := ""
	if s.BotID == "" {
		hint = s.Adapter
	}
	return json.Marshal(struct {
		plain
		LegacyAdapter string `json:"legacy_adapter,omitempty"`
	}{plain(s.identity()), hint})
}
func (s *GroupScope) UnmarshalJSON(raw []byte) error {
	type plain GroupScope
	var value struct {
		plain
		LegacyAdapter string `json:"legacy_adapter,omitempty"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*s = GroupScope(value.plain)
	if s.Adapter == "" {
		s.Adapter = value.LegacyAdapter
	}
	return nil
}

func recoverBot(protocol, adapter, botID string, bots []rayleabot.Bot) string {
	if botID != "" || adapter == "" {
		return botID
	}
	for _, bot := range bots {
		if bot.SourceProtocol == protocol && bot.SourceAdapter == adapter {
			return bot.ID
		}
	}
	return ""
}

func (s *Subject) resolve(bots []rayleabot.Bot, record string) bool {
	before := *s
	s.BotID = recoverBot(s.SourceProtocol, s.SourceAdapter, s.BotID, bots)
	if s.BotID != "" {
		s.SourceAdapter = ""
	} else {
		log.Printf("机器人账号未确定，保留记录 %s；请恢复原连接身份或重新选择机器人账号。", record)
	}
	return *s != before
}

// Migration runs before the first event, then again when the host's bot
// snapshot changes. Store locks also protect it from running task writes.
// A failure leaves the snapshot unmarked so the next event can resume.
type addressMigration struct {
	mu    sync.Mutex
	bots  string
	ready bool
}

func (a *App) migrateAddresses(ctx context.Context, event *rayleabot.EventContext) error {
	m := &a.addressMigration
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, _ := json.Marshal(event.Bots)
	if m.ready && m.bots == string(raw) {
		return nil
	}
	steps := []func() error{
		func() error {
			return migrateTaskOwners(&a.Reminders.taskFiles, event.Bots, func(t *Reminder) *Subject { return &t.Owner })
		},
		func() error {
			return migrateTaskOwners(&a.SyncTasks.taskFiles, event.Bots, func(t *SyncTask) *Subject { return &t.Owner })
		},
		func() error { return a.Groups.migrate(event.Bots) },
		func() error { return a.Interactions.migrate(event.Bots) },
		func() error { return a.migrateSubscriptions(ctx, event) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			log.Printf("机器人账号迁移未完成，保留原记录并在下次事件重试：%v", err)
			return err
		}
	}
	m.ready, m.bots = true, string(raw)
	return nil
}

func migrateTaskOwners[T storedTask](s *taskFiles[T], bots []rayleabot.Bot, owner func(*T) *Subject) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	for _, task := range items {
		if owner(&task).resolve(bots, task.taskRef()) {
			if err := localdata.Write(s.file(task.taskRef()), task); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *GroupStore) migrate(bots []rayleabot.Bot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := os.ReadDir(s.Directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, file := range files {
		if !file.Type().IsRegular() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.Directory, file.Name())
		var old GroupData
		if err := localdata.Read(path, &old); err != nil {
			return err
		}
		old.Scope.BotID = recoverBot(old.Scope.Protocol, old.Scope.Adapter, old.Scope.BotID, bots)
		if old.Scope.BotID == "" {
			log.Printf("群记录 %s 缺少机器人账号，保留原文件；请恢复原连接身份或重新选择机器人账号。", file.Name())
			continue
		}
		old.Scope = old.Scope.identity()
		target := s.file(old.Scope)
		if path == target {
			continue
		}
		var data GroupData
		if _, err := os.Stat(target); err == nil {
			if err := localdata.Read(target, &data); err != nil {
				return err
			}
			mergeGroup(&data, old)
		} else {
			if !os.IsNotExist(err) {
				return err
			}
			data = old
		}
		if err := localdata.Write(target, data); err != nil {
			return err
		}
		// Backups are outside the active .json set, including conflicting
		// scalar settings and older versions of the same ranking entry.
		if err := os.Rename(path, path+".legacy"); err != nil {
			return err
		}
		delete(s.configs, old.Scope)
	}
	return nil
}

// The canonical file wins conflicting settings. Fill missing settings and
// aliases; rankings keep each participant's newest submission.
func mergeGroup(dst *GroupData, old GroupData) {
	if dst.Config.Enabled == nil {
		dst.Config.Enabled = old.Config.Enabled
	}
	if dst.Config.ImageReplies == nil {
		dst.Config.ImageReplies = old.Config.ImageReplies
	}
	if dst.Config.Aliases == nil {
		dst.Config.Aliases = map[string]string{}
	}
	for key, value := range old.Config.Aliases {
		if _, ok := dst.Config.Aliases[key]; !ok {
			dst.Config.Aliases[key] = value
		}
	}
	for _, entry := range old.Rank {
		i := slices.IndexFunc(dst.Rank, func(v RankEntry) bool { return v.UID == entry.UID && v.CharacterID == entry.CharacterID })
		if i < 0 {
			dst.Rank = append(dst.Rank, entry)
		} else if entry.UpdatedAtMS > dst.Rank[i].UpdatedAtMS {
			dst.Rank[i] = entry
		}
	}
	for _, entry := range old.Challenges {
		i := slices.IndexFunc(dst.Challenges, func(v ChallengeEntry) bool {
			return v.Kind == entry.Kind && v.Season == entry.Season && v.Region == entry.Region && v.ActorID == entry.ActorID
		})
		if i < 0 {
			dst.Challenges = append(dst.Challenges, entry)
		} else {
			first := min(dst.Challenges[i].FirstMS, entry.FirstMS)
			if entry.UpdatedMS > dst.Challenges[i].UpdatedMS {
				dst.Challenges[i] = entry
			}
			dst.Challenges[i].FirstMS = first
		}
	}
	if dst.RankSinceMS == 0 || old.RankSinceMS > 0 && old.RankSinceMS < dst.RankSinceMS {
		dst.RankSinceMS = old.RankSinceMS
	}
	dst.Revision = max(dst.Revision, old.Revision) + 1
}

func contentRef(game string, owner Subject, group string) string {
	identity := strings.Join([]string{owner.SourceProtocol, owner.BotID, group}, "\x00")
	sum := sha256.Sum256([]byte(identity))
	return "game.content." + game + "." + hex.EncodeToString(sum[:16])
}

func (a *App) migrateSubscriptions(ctx context.Context, event *rayleabot.EventContext) error {
	s := a.Subscriptions
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	merged := []ContentSubscription{}
	changed := false
	for _, item := range items {
		changed = item.Owner.resolve(event.Bots, item.Ref) || changed
		if item.Owner.BotID == "" {
			merged = append(merged, item)
			continue
		}
		ref := contentRef(a.Game.ID, item.Owner, item.GroupID)
		if ref != item.Ref {
			item.PreviousRefs = append(item.PreviousRefs, item.Ref)
			item.Ref, changed = ref, true
		}
		i := slices.IndexFunc(merged, func(v ContentSubscription) bool { return v.Ref == item.Ref })
		if i < 0 {
			merged = append(merged, item)
			continue
		}
		changed = true
		dst := &merged[i]
		for _, kind := range item.Kinds {
			if !slices.Contains(dst.Kinds, kind) {
				dst.Kinds = append(dst.Kinds, kind)
			}
		}
		if dst.Sent == nil {
			dst.Sent = map[string]int64{}
		}
		for post, at := range item.Sent {
			dst.Sent[post] = max(dst.Sent[post], at)
		}
		if item.WarnedOn > dst.WarnedOn {
			dst.WarnedOn = item.WarnedOn
		}
		dst.PreviousRefs = append(dst.PreviousRefs, item.PreviousRefs...)
	}
	if changed {
		if err := localdata.Write(s.Path, merged); err != nil {
			return err
		}
	}
	cleaned := false
	for i := range merged {
		item := &merged[i]
		if len(item.PreviousRefs) == 0 {
			continue
		}
		// Persist the canonical record first. Old triggers can no longer
		// push; a failed create/delete is resumed from PreviousRefs.
		if _, err := event.Actions().SchedulerCreate(ctx, a.contentJob(item.Ref)); err != nil {
			return err
		}
		for _, ref := range item.PreviousRefs {
			if _, err := event.Actions().SchedulerDelete(ctx, ref); err != nil {
				return err
			}
		}
		item.PreviousRefs, cleaned = nil, true
	}
	if cleaned {
		return localdata.Write(s.Path, merged)
	}
	return nil
}

func (s *InteractionStore) migrate(bots []rayleabot.Bot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	merged := []InteractionProfile{}
	changed := false
	for _, item := range items {
		changed = item.Owner.resolve(bots, "interaction:"+item.Ref) || changed
		if item.Owner.BotID == "" {
			merged = append(merged, item)
			continue
		}
		ref := interactionRef(item.Owner)
		changed = changed || ref != item.Ref
		item.Ref = ref
		i := slices.IndexFunc(merged, func(v InteractionProfile) bool { return v.Ref == ref })
		if i < 0 {
			merged = append(merged, item)
		} else {
			changed = true
			if item.LastImageMS > merged[i].LastImageMS {
				merged[i] = item
			}
		}
	}
	if changed {
		// Preserve previous image preferences when old instances collapse.
		backup := s.Path + ".legacy"
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			if err := localdata.Write(backup, items); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		return localdata.Write(s.Path, merged)
	}
	return nil
}

// Simulation filenames contain only a hash, with no stored source identity.
// Recover the legacy scopes we can identify on the first chat access. Other
// old files remain untouched. The canonical archive wins a collision.
func (s *SimulationStore) migrateScope(event *rayleabot.EventContext, scope string) error {
	if event.Event.EventType == "management.action" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, adapter := range legacyAdapters(event) {
		raw, _ := json.Marshal([]string{event.Event.SourceProtocol, adapter, event.Event.BotID, event.Event.Target.Type, event.Event.Target.ID, event.Event.Actor.ID})
		old, target := s.filename(string(raw)), s.filename(scope)
		if _, err := os.Stat(old); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if _, err := os.Stat(target); os.IsNotExist(err) {
			var state SimulationState
			if err := localdata.Read(old, &state); err != nil {
				return err
			}
			if err := localdata.Write(target, state); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			log.Printf("模拟档案合并时保留账号档案，旧档案备份为 %s.legacy。", filepath.Base(old))
		}
		if err := os.Rename(old, old+".legacy"); err != nil {
			return fmt.Errorf("migrate simulation: %w", err)
		}
	}
	return nil
}

func legacyAdapters(event *rayleabot.EventContext) []string {
	adapters := []string{event.Event.SourceAdapter}
	for _, bot := range event.Bots {
		if bot.SourceProtocol == event.Event.SourceProtocol && bot.ID == event.Event.BotID && !slices.Contains(adapters, bot.SourceAdapter) {
			adapters = append(adapters, bot.SourceAdapter)
		}
	}
	return adapters
}
