package app

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestAliasOwnerPrefersCustomAliases(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"1","entries":[{"id":"1102","name":"希儿","aliases":["Seele"],"kind":"character"},{"id":"1101","name":"布洛妮娅","aliases":["鸭鸭"],"kind":"character"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Catalog: catalog}
	custom := map[string]string{"小鸭": "1101", "SEELE": "1101"}
	if entry, alias, ok := a.aliasOwner("seele", custom); !ok || entry.ID != "1101" || alias != "SEELE" {
		t.Fatal(entry, alias, ok)
	}
	if entry, alias, ok := a.aliasOwner("鸭鸭", custom); !ok || entry.ID != "1101" || alias != "" {
		t.Fatal(entry, alias, ok)
	}
	if _, _, ok := a.aliasOwner("不存在", custom); ok {
		t.Fatal("found a missing alias")
	}
	if list := aliasesOf(catalog.Entries[1], custom); !slices.Equal(list, []string{"鸭鸭", "SEELE", "小鸭"}) {
		t.Fatal(list)
	}
}

func TestAliasWordsWithoutACharacterListTheAliases(t *testing.T) {
	host := newSDKHost(t, pluginApp(t), nil)
	for _, word := range []string{"别名列表", "喵喵别名帮助"} {
		end, _ := host.groupMessage("*"+word, word)
		if !strings.Contains(terminalText(end), "*删除别名") {
			t.Fatalf("%s: %+v", word, end)
		}
	}
}

func TestChatAliasChangesLeaveGroupAliasesOut(t *testing.T) {
	a := pluginApp(t)
	if err := a.Groups.Update(GroupScope{"onebot11", "bot", "g"}, func(data *GroupData) error {
		data.Config.Aliases = map[string]string{"老婆": "1102", "小鸭": "1102"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	host := newSDKHost(t, a, nil)
	_, actions := host.groupMessage("*喵喵别名设置 布洛妮娅 小鸭", "喵喵别名设置", "布洛妮娅", "小鸭")
	wrote := false
	for _, action := range actions {
		if action.Name == "config.write" {
			var aliases map[string]string
			if err := decodeObject(asObject(action.Data["values"])["custom_aliases"], &aliases); err != nil {
				t.Fatal(err)
			}
			if !maps.Equal(aliases, map[string]string{"小鸭": "1101"}) {
				t.Fatalf("wrote group aliases to global config: %v", aliases)
			}
			wrote = true
		}
	}
	if !wrote {
		t.Fatal("no config.write action", actions)
	}
	end, _ := host.groupMessage("*别名列表", "别名列表")
	global, group, found := strings.Cut(terminalText(end), "【本群别名（群管理员用群设置增删）】")
	if !found || !strings.Contains(global, "布洛妮娅：小鸭") || strings.Contains(global, "老婆") || strings.Contains(global, "希儿：") || !strings.Contains(group, "希儿：小鸭，老婆") {
		t.Fatalf("global: %s\ngroup: %s", global, group)
	}
	end, actions = host.groupMessage("*删除别名 老婆", "删除别名", "老婆")
	if !strings.Contains(terminalText(end), "未识别到角色") {
		t.Fatal(end)
	}
	for _, action := range actions {
		if action.Name == "config.write" {
			t.Fatal("deleting a group alias wrote global config", action)
		}
	}
	end, _ = host.groupMessage("*删除别名 小鸭", "删除别名", "小鸭")
	if !strings.Contains(terminalText(end), "删除布洛妮娅别名成功") {
		t.Fatal(end)
	}
	end, _ = host.groupMessage("*别名列表", "别名列表")
	global, group, found = strings.Cut(terminalText(end), "【本群别名（群管理员用群设置增删）】")
	if !found || !strings.Contains(global, "暂无自定义别名") || !strings.Contains(group, "希儿：小鸭，老婆") {
		t.Fatalf("global: %s\ngroup: %s", global, group)
	}
}
