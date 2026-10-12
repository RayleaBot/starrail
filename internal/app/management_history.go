package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/starrail/internal/localdata"
)

type historyRecord struct {
	Kind    string  `json:"kind"`
	Ref     string  `json:"ref"`
	Owner   Subject `json:"owner"`
	Label   string  `json:"label"`
	Version string  `json:"version"`
	Count   int     `json:"count"`
}

type historyFiles struct {
	kind, directory string
	mu              *sync.Mutex
}

func (a *App) historyFiles() []historyFiles {
	return []historyFiles{
		{"groups", a.Groups.Directory, &a.Groups.mu},
		{"tasks", a.Reminders.Directory, &a.Reminders.mu},
		{"sync", a.SyncTasks.Directory, &a.SyncTasks.mu},
		{"monthly", a.Monthly.Directory, &a.Monthly.mu},
		{"panels", a.PanelHistory.Directory, &a.PanelHistory.mu},
		{"billing", a.Billing.Directory, &a.Billing.mu},
		{"exchange", a.CloudArchive.Directory, &a.CloudArchive.mu},
		{"simulation", a.Simulation.Directory, &a.Simulation.mu},
	}
}

var historyFilename = regexp.MustCompile(`^[a-f0-9]{32}(?:[a-f0-9]{32})?\.json$`)

func historyVersion(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func describeHistory(kind, ref string, raw json.RawMessage) (historyRecord, error) {
	var header struct {
		Owner      Subject           `json:"owner"`
		Scope      GroupScope        `json:"scope"`
		Role       Role              `json:"role"`
		UID        string            `json:"uid"`
		GroupID    string            `json:"group_id"`
		Items      []json.RawMessage `json:"items"`
		Rank       []json.RawMessage `json:"rank"`
		Challenges []json.RawMessage `json:"challenges"`
	}
	item := historyRecord{Kind: kind, Ref: ref, Version: historyVersion(raw), Count: 1}
	if kind == "panels" {
		var snapshots []PanelSnapshot
		if err := json.Unmarshal(raw, &snapshots); err != nil {
			return item, err
		}
		item.Count = len(snapshots)
		if len(snapshots) > 0 {
			item.Label = snapshots[0].Panel.Name
		}
		return item, nil
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return item, err
	}
	item.Owner, item.Label = header.Owner, header.Role.UID
	if item.Label == "" {
		item.Label = header.UID
	}
	if item.Label == "" {
		item.Label = header.Owner.ActorID
	}
	if header.GroupID != "" {
		item.Label = "群 " + header.GroupID
	}
	if kind == "groups" {
		item.Owner = Subject{SourceProtocol: header.Scope.Protocol, BotID: header.Scope.BotID}
		item.Label = "群 " + header.Scope.GroupID
		item.Count += len(header.Rank) + len(header.Challenges)
	}
	if kind == "monthly" {
		item.Count = len(header.Items)
	}
	return item, nil
}

func (a *App) listHistory() ([]historyRecord, error) {
	items := []historyRecord{}
	for _, store := range a.historyFiles() {
		err := func() error {
			store.mu.Lock()
			defer store.mu.Unlock()
			files, err := os.ReadDir(store.directory)
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return err
			}
			for _, file := range files {
				if !file.Type().IsRegular() || !historyFilename.MatchString(file.Name()) {
					continue
				}
				if store.kind == "simulation" && file.Name() == filepath.Base(a.Simulation.filename("management")) {
					continue
				}
				raw, err := os.ReadFile(filepath.Join(store.directory, file.Name()))
				if err != nil {
					return err
				}
				item, err := describeHistory(store.kind, file.Name(), raw)
				if err != nil {
					return err
				}
				items = append(items, item)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}
	for _, store := range []struct {
		kind, path string
		mu         *sync.Mutex
	}{
		{"push", a.Subscriptions.Path, &a.Subscriptions.mu}, {"interaction", a.Interactions.Path, &a.Interactions.mu},
	} {
		err := func() error {
			store.mu.Lock()
			defer store.mu.Unlock()
			var rows []json.RawMessage
			if err := localdata.Read(store.path, &rows); err != nil {
				return err
			}
			for _, raw := range rows {
				var row struct {
					Ref string `json:"ref"`
				}
				if err := json.Unmarshal(raw, &row); err != nil {
					return err
				}
				item, err := describeHistory(store.kind, row.Ref, raw)
				if err != nil {
					return err
				}
				items = append(items, item)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

// Assignment arrives with a host-verified target account. The global history
// UI opens that scoped action through the SDK; business payload cannot name it.
func (a *App) managementHistory(ctx context.Context, event *rayleabot.EventContext, action string, input map[string]any) (map[string]any, error) {
	bot, scoped := event.Event.ManagementBot()
	if action == "history.assign" {
		var err error
		bot, err = managementAccount(event)
		if err != nil {
			return nil, err
		}
		if bot.SourceProtocol != "onebot11" {
			return nil, gameError("source_invalid", "仅支持归入 OneBot11 账号。")
		}
	} else if scoped {
		return nil, gameError("source_invalid", "请在全局历史数据入口操作。")
	}
	if action == "history.list" {
		items, err := a.listHistory()
		return map[string]any{"items": items, "total": len(items)}, err
	}
	if action != "history.get" && action != "history.remove" && action != "history.assign" {
		return nil, gameError("operation_denied", "历史数据操作不存在。")
	}
	if action != "history.get" && input["confirm"] != true {
		return nil, gameError("input_invalid", "请确认历史数据操作。")
	}
	kind, ref := asText(input["kind"]), asText(input["ref"])
	var raw json.RawMessage
	var taskID string
	edit := func(current json.RawMessage) (json.RawMessage, error) {
		item, err := describeHistory(kind, ref, current)
		if err != nil {
			return nil, err
		}
		if item.Version != asText(input["version"]) {
			return nil, gameError("history_changed", "记录已变化，请刷新后重试。")
		}
		raw = current
		if action == "history.get" {
			return current, nil
		}
		var data map[string]any
		if kind == "tasks" || kind == "sync" || kind == "push" {
			if err := json.Unmarshal(current, &data); err != nil {
				return nil, err
			}
			taskID = asText(data["ref"])
		}
		if action == "history.remove" {
			return nil, nil
		}
		if item.Owner.SourceProtocol != "onebot11" || item.Owner.BotID != "" {
			return nil, gameError("source_invalid", "仅可归属缺少账号的 OneBot11 记录。")
		}
		if data == nil {
			if err := json.Unmarshal(current, &data); err != nil {
				return nil, err
			}
		}
		if kind == "groups" {
			var group GroupData
			if err := json.Unmarshal(current, &group); err != nil {
				return nil, err
			}
			group.Scope.BotID, group.Scope.Adapter = bot.BotID, ""
			return json.Marshal(group)
		}
		owner := item.Owner
		owner.BotID, owner.SourceAdapter = bot.BotID, ""
		data["owner"] = owner
		switch kind {
		case "tasks":
			data["enabled"], data["last_code"] = false, "bot_assigned"
		case "sync":
			data["state"], data["last_code"] = "paused", "bot_assigned"
		case "interaction":
			data["ref"] = interactionRef(owner)
		case "push":
			data["ref"] = contentRef(a.Game.ID, owner, asText(data["group_id"]))
			previous := []string{}
			if err := decodeObject(data["previous_refs"], &previous); err != nil {
				return nil, err
			}
			data["previous_refs"] = append(previous, taskID)
		}
		return json.Marshal(data)
	}
	var err error
	if kind == "push" || kind == "interaction" {
		path, mu := a.Subscriptions.Path, &a.Subscriptions.mu
		if kind == "interaction" {
			path, mu = a.Interactions.Path, &a.Interactions.mu
		}
		err = func() error {
			mu.Lock()
			defer mu.Unlock()
			rows := []json.RawMessage{}
			if err := localdata.Read(path, &rows); err != nil {
				return err
			}
			index := -1
			for i, row := range rows {
				var header struct {
					Ref string `json:"ref"`
				}
				if err := json.Unmarshal(row, &header); err != nil {
					return err
				}
				if header.Ref == ref {
					index = i
				}
			}
			if index < 0 {
				return gameError("history_missing", "记录已不存在。")
			}
			next, err := edit(rows[index])
			if err != nil || action == "history.get" {
				return err
			}
			if next == nil {
				rows = slices.Delete(rows, index, index+1)
			} else {
				var changed struct {
					Ref string `json:"ref"`
				}
				if err := json.Unmarshal(next, &changed); err != nil {
					return err
				}
				for i, row := range rows {
					if i == index {
						continue
					}
					var other struct {
						Ref string `json:"ref"`
					}
					if err := json.Unmarshal(row, &other); err != nil {
						return err
					}
					if other.Ref == changed.Ref {
						return gameError("history_changed", "目标账号已有此记录，请先处理已有记录。")
					}
				}
				rows[index] = next
			}
			return localdata.Write(path, rows)
		}()
	} else {
		stores := a.historyFiles()
		i := slices.IndexFunc(stores, func(store historyFiles) bool { return store.kind == kind })
		if i < 0 || !historyFilename.MatchString(ref) || kind == "simulation" && ref == filepath.Base(a.Simulation.filename("management")) {
			return nil, gameError("input_invalid", "历史记录标识无效。")
		}
		store := stores[i]
		err = func() error {
			store.mu.Lock()
			defer store.mu.Unlock()
			path := filepath.Join(store.directory, ref)
			current, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			next, err := edit(current)
			if err != nil || action == "history.get" {
				return err
			}
			if kind == "groups" {
				var old GroupData
				if err := json.Unmarshal(current, &old); err != nil {
					return err
				}
				delete(a.Groups.configs, old.Scope.identity())
				if next != nil {
					var data, target GroupData
					if err := json.Unmarshal(next, &data); err != nil {
						return err
					}
					file := a.Groups.file(data.Scope)
					if err := localdata.Read(file, &target); err != nil {
						return err
					}
					if target.Scope.BotID != "" {
						mergeGroup(&target, data)
						data = target
					} else {
						data.Revision++
					}
					if err := localdata.Write(file, data); err != nil {
						return err
					}
					a.Groups.remember(data.Scope, data.Config)
					return os.Rename(path, path+".legacy")
				}
			}
			if next == nil {
				return os.Remove(path)
			}
			return localdata.Write(path, next)
		}()
	}
	if err != nil {
		return nil, err
	}
	if action == "history.get" {
		return map[string]any{"record": raw}, nil
	}
	if action == "history.remove" && taskID != "" {
		_, err = event.Actions().SchedulerDelete(ctx, taskID)
	}
	if action == "history.assign" && kind == "push" {
		a.addressMigration.mu.Lock()
		a.addressMigration.ready = false
		a.addressMigration.mu.Unlock()
		err = a.migrateSubscriptions(ctx, event)
	}
	return map[string]any{"updated": true}, err
}
