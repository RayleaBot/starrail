package app

import (
	"reflect"
	"testing"
)

func TestParseCustomRankFollowsArk(t *testing.T) {
	// A super administrator with the token: every column, by Chinese name,
	// the words around the filters joined into the name, and a row count.
	name, query, err := parseCustomRank("丹恒 等级>=80 饮月 精炼!=1 数量:10", customRankColumns, "dmg_avg", "")
	want := []customRankFilter{{Type: "level", Rule: 0, Value: 80.0, RawValue: "80"}, {Type: "weapon_affix", Rule: 5, Value: 1.0, RawValue: "1"}}
	if err != nil || name != "丹恒饮月" || query.Nums != 10 || query.Rank.Col != "dmg_avg" || query.Rank.Order != "desc" || !reflect.DeepEqual(query.Filter, want) {
		t.Fatalf("parsed %q %+v %v", name, query, err)
	}
	// Sets may be named, slots keep their stat, counts stay within 1–50.
	_, query, _ = parseCustomRank("希儿 套装=快枪手4 部位4=速度 NUMS：0", customRankColumns, "mark_score", "")
	want = []customRankFilter{{Type: "artis_sets", Rule: 1, Value: "快枪手4", RawValue: "快枪手4"}, {Type: "pos4", Rule: 1, Value: "速度", RawValue: "速度"}}
	if query.Nums != 1 || query.Rank.Col != "mark_score" || !reflect.DeepEqual(query.Filter, want) {
		t.Fatalf("parsed %+v", query)
	}
	for raw, nums := range map[string]int{"希儿": 20, "希儿 数量=70": 50, "希儿 nums:99999999999999999999": 50} {
		if _, query, _ := parseCustomRank(raw, customRankColumns, "dmg_avg", ""); query.Nums != nums {
			t.Errorf("%s asks for %d rows", raw, query.Nums)
		}
	}
	cases := []struct {
		raw     string
		columns []string
		denied  string
		want    string
	}{
		{"希儿 升序", customRankColumns, "", "当前命令仅支持默认降序排序"},
		{"希儿 排序:dmg", customRankColumns, "", "当前命令仅支持默认降序排序"},
		{"希儿 等级=八十", customRankColumns, "", "等级 的值必须是数字"},
		{"希儿 foo=1", customRankColumns, "", "不支持筛选列 foo"},
		// Without the token only the Eidolon filters, whoever asks; the value
		// is checked before the column, as upstream.
		{"希儿 部位4=速度", []string{"cons"}, "除 cons 之外的筛选仅主人可用", "除 cons 之外的筛选仅主人可用"},
		{"希儿 等级>=abc", []string{"cons"}, "除 cons 之外的筛选仅主人可用", "等级 的值必须是数字"},
	}
	for _, tc := range cases {
		if _, _, err := parseCustomRank(tc.raw, tc.columns, "dmg_avg", tc.denied); err == nil || err.Error() != tc.want {
			t.Errorf("%s: %v, want %s", tc.raw, err, tc.want)
		}
	}
	if name, query, err := parseCustomRank("希儿 命座<=2 cons>0", []string{"cons"}, "dmg_avg", "除 cons 之外的筛选仅主人可用"); err != nil || name != "希儿" || len(query.Filter) != 2 {
		t.Fatalf("Eidolon filters refused: %v %+v", err, query)
	}
}

// ark reads a slot's main stat by its Chinese names or its key and sends the
// stat's number within the slot.
func TestCustomRankMainStatFollowsArk(t *testing.T) {
	cases := []struct {
		slot, name string
		id         int
		ok         bool
	}{
		{"4", "速度", 4, true}, {"3", "暴击", 4, true}, {"3", "暴伤", 5, true}, {"3", "cdmg", 5, true},
		{"5", "量子伤", 9, true}, {"5", "虚数属性伤害提高", 10, true}, {"6", "充能", 2, true}, {"6", "击破特攻", 1, true},
		{"1", "小生命", 1, true}, {"2", "atkPlus", 1, true},
		{"4", "暴击", 0, false}, {"1", "生命", 0, false}, {"5", "治疗", 0, false},
	}
	for _, tc := range cases {
		if id, ok := customRankMainStat(tc.slot, tc.name); id != tc.id || ok != tc.ok {
			t.Errorf("部位%s=%s: %d %v, want %d %v", tc.slot, tc.name, id, ok, tc.id, tc.ok)
		}
	}
}

func TestCustomRankLimitWritesArkNote(t *testing.T) {
	_, query, _ := parseCustomRank("希儿", customRankColumns, "dmg_avg", "")
	if got := customRankLimit(query); got != "排序: 伤害均值 降序 / 筛选: 无" {
		t.Fatal(got)
	}
	// ark notes the resolved column with the value as written.
	_, query, _ = parseCustomRank("希儿 命=6 部位3=暴击", customRankColumns, "mark_score", "")
	if got := customRankLimit(query); got != "排序: 遗器评分 降序 / 筛选: cons=6 部位3=暴击" {
		t.Fatal(got)
	}
}

// Replies write 遗器 where ark writes Genshin's 圣遗物; the command and the
// column names take both, so the usage the help writes works as sent.
func TestCustomRankTakesRelicWords(t *testing.T) {
	for _, message := range []string{"ark符玄遗器排行 部位4=速度", "ark符玄圣遗物排名 部位4=速度"} {
		if match := customRankWord.FindStringSubmatch(message); match == nil || match[1] != "符玄" || match[2] != "遗器" && match[2] != "圣遗物" || match[4] != "部位4=速度" {
			t.Errorf("%s = %q", message, match)
		}
	}
	for _, raw := range []string{"希儿 遗器评分>=200 遗器=快枪手4", "希儿 圣遗物评分>=200 圣遗物=快枪手4"} {
		_, query, err := parseCustomRank(raw, customRankColumns, "dmg_avg", "")
		if err != nil || len(query.Filter) != 2 || query.Filter[0].Type != "mark_score" || query.Filter[1].Type != "artis_sets" {
			t.Errorf("%s = %+v %v", raw, query, err)
		}
	}
}

func TestCustomRankEntriesAddEidolonTalents(t *testing.T) {
	a := pluginApp(t)
	entry, _ := a.Catalog.Get("1407")
	record, err := findReferenceCharacter(a.Game.Calc, CharacterPanel{ID: entry.ID, Element: entry.Element})
	if err != nil {
		t.Fatal(err)
	}
	rows := []any{
		map[string]any{"uid": "10****543", "cons": 5, "talent_a": 6, "talent_e": 10, "talent_q": 10, "talent_t": 10, "talent_me": 6, "talent_mt": 6, "weapon_id": 23039, "weapon_level": 80, "weapon_affix": 1,
			"mark_score": 281.6, "dmg_avg": 64248.5, "artisSet": map[string]any{"names": []any{"荡除蠹灾的铁骑", "不老者的仙舟"}, "name": "铁骑4+仙舟2"}},
		map[string]any{"uid": "15****910", "talent_a": 6},
	}
	entries := customRankEntries(record, rows)
	first, second := entries[0], entries[1]
	// 遐蝶's third Eidolon raises the basic attack, ultimate and memosprite
	// talent, the fifth the skill, talent and memosprite skill.
	want := map[string]PanelTalent{"a": {Level: 7, Original: 6}, "e": {Level: 12, Original: 10}, "q": {Level: 12, Original: 10}, "t": {Level: 12, Original: 10}, "me": {Level: 7, Original: 6}, "mt": {Level: 7, Original: 6}}
	if first.UID != "10****543" || first.Weapon != "23039" || first.WeaponAffix != 1 || !reflect.DeepEqual(first.Talents, want) {
		t.Fatalf("first row %+v", first)
	}
	if first.Damage == nil || *first.Damage != 64248.5 || first.SetName != "铁骑4+仙舟2" || !reflect.DeepEqual(first.Sets, []string{"荡除蠹灾的铁骑", "不老者的仙舟"}) {
		t.Fatalf("first row %+v", first)
	}
	// A talent ark sends as 0 or leaves out has no level; no damage without
	// dmg_avg.
	if len(second.Talents) != 1 || second.Damage != nil || second.Mark != 0 {
		t.Fatalf("second row %+v", second)
	}
}

func TestCustomRankAvatarReadsArkPlayerData(t *testing.T) {
	answer := cloudObject(t, `{"retcode":0,"data":{"info":{"uid":"10@@@@543","avatars":{"1102":{"name":"希儿","id":1102,"elem":"quantum","level":80,"promote":6,"cons":2,
		"talent":{"a":6,"e":10,"t":10,"q":10},"trees":["1102101","1102201"],"weapon":{"id":23001,"level":80,"promote":6,"affix":1},
		"artis":{"1":{"level":15,"star":5,"id":61011,"mainId":1,"attrIds":["5,2,2","6,1,0"]}},"_source":"homo"}}}}}`)
	uid, avatar, failure := customRankAvatar(answer, "1102")
	if failure != "" || uid != "10@@@@543" || avatar["name"] != "希儿" {
		t.Fatalf("read %q %v %q", uid, avatar, failure)
	}
	// A character the answer does not hold falls back to its first one.
	if _, other, _ := customRankAvatar(answer, "1208"); other["name"] != "希儿" {
		t.Fatalf("fallback %v", other)
	}
	_, raw, err := cleanCloudAvatar(avatar)
	if err != nil {
		t.Fatal(err)
	}
	panel, err := pluginApp(t).cloudAvatarPanel(t.Context(), raw)
	if err != nil || panel.ID != "1102" || panel.Rank != 2 || panel.Weapon == nil || panel.Weapon.Name != "于夜色中" || len(panel.Equipment) != 1 {
		t.Fatalf("panel %+v %v", panel, err)
	}
	for raw, want := range map[string]string{
		`{"retcode":-1}`: "云服务请求失败，请稍后重试。",
		`{"retcode":1,"message":"query expired"}`:         "云服务请求失败，请稍后重试。",
		`{"retcode":0,"data":{"playerData":{"uid":"1"}}}`: "返回数据异常",
		`{"retcode":0,"data":{"uid":"1","avatars":{}}}`:   "未找到对应角色面板",
		`{"retcode":0,"data":{"avatars":{"1":{"id":1}}}}`: "返回数据异常",
	} {
		if _, _, failure := customRankAvatar(cloudObject(t, raw), "1102"); failure != want {
			t.Errorf("%s: %q, want %q", raw, failure, want)
		}
	}
}
