package app

import (
	"context"
	"encoding/json"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-starrail/internal/gacha"
)

func TestGachaFilesRoundTripAndFillLegacyRecords(t *testing.T) {
	a := pluginApp(t)
	// A UIGF v4 file keeps long IDs exact, drops non-standard fields, and
	// takes names and ranks from the catalog when a record leaves them out.
	file := `{"info":{"version":"v4.0","authkey":"synthetic"},"hk4e":[{"uid":"100000009","timezone":8,"list":[]}],"hkrpg":[{"uid":100000001,"timezone":8,"lang":"zh-cn","list":[
		{"id":1844674407370955101,"gacha_type":"11","gacha_id":"2001","item_id":"1102","time":"2026-09-01 08:00:00","cookie_token":"synthetic"}]}]}`
	archives, err := a.parseGachaFile([]byte(file))
	if err != nil || len(archives) != 1 {
		t.Fatal(archives, err)
	}
	record := archives[0].Records[0]
	if archives[0].UID != "100000001" || record.ID != "1844674407370955101" || record.Name != "希儿" || record.ItemType != "角色" || record.Rank != "5" {
		t.Fatalf("%+v", archives[0])
	}
	archive := archives[0]
	archive.Region = "prod_gf_cn"
	for _, legacy := range []bool{false, true} {
		raw, _ := json.Marshal(a.gachaFile(archive, legacy))
		again, err := a.parseGachaFile(raw)
		if err != nil || len(again) != 1 || again[0].Records[0] != record || again[0].UID != archive.UID || again[0].Timezone != 8 {
			t.Fatalf("legacy %v: %+v %v", legacy, again, err)
		}
	}
	// SRGF files may name items only; the catalog gives their IDs.
	legacy, err := a.parseGachaFile([]byte(`{"info":{"uid":"100000001","srgf_version":"v1.0","region_time_zone":8},"list":[{"id":"1","gacha_type":"12","gacha_id":"3001","name":"于夜色中","time":"2026-09-01 08:00:00"}]}`))
	if err != nil || legacy[0].Records[0].ItemID != "23001" || legacy[0].Records[0].ItemType != "光锥" || legacy[0].Timezone != 8 {
		t.Fatal(legacy, err)
	}
	if other, err := a.parseGachaFile([]byte(`{"info":{"uid":"100000001","uigf_version":"v2.2"},"list":[{"id":"1"}]}`)); err != nil || other != nil {
		t.Fatal("a Genshin file was read", other, err)
	}
	for text, want := range map[string]string{
		`{"info":{}}`: "json文件内容错误：非统一祈愿记录标准",
		`{"info":{"srgf_version":"v1.0","uid":"1"},"list":[{"id":"1","gacha_type":"11","item_id":"1102","time":"2026-09-01 08:00:00"}]}`: "json文件内容错误：缺少必要字段 gacha_id",
		`not json`: "{name},json格式错误",
	} {
		if _, err := a.parseGachaFile([]byte(text)); friendlyError(err) != want {
			t.Errorf("%s: %v", text, err)
		}
	}
}

func TestGachaImportsGoUnderTheUIDsRegion(t *testing.T) {
	a := pluginApp(t)
	listed := Accounts{Items: []Account{{Roles: []Role{{Game: "starrail", UID: "600000001", Region: "prod_official_usa"}}}}}
	if _, _, err := a.Gacha.Import(gacha.Archive{UID: "500000001", Region: "prod_gf_cn", Timezone: 8}); err != nil {
		t.Fatal(err)
	}
	for uid, want := range map[string]string{"600000001": "prod_official_usa", "500000001": "prod_gf_cn", "500000002": "prod_qd_cn", "1800000001": "prod_official_asia", "100000001": "prod_gf_cn"} {
		if region := a.archiveRegion(listed, uid); region != want {
			t.Errorf("%s: %s", uid, region)
		}
	}
	if regionTimezone("prod_official_eur") != 1 || regionTimezone("prod_official_usa") != -5 || regionTimezone("prod_qd_cn") != 8 {
		t.Error("region time zones")
	}
}

func TestGachaImportFindsTheSentFile(t *testing.T) {
	event := &rayleabot.EventContext{}
	event.Event.Message.Segments = []rayleabot.Segment{{Type: "file", Data: map[string]any{"name": "100000001.json", "url": "https://files.example/abc"}}}
	if file, name, found := messageFile(context.Background(), event); !found || file != "https://files.example/abc" || name != "100000001.json" {
		t.Fatal(file, name, found)
	}
	event.Event.Message.Segments = nil
	event.Event.Message.PlainText = "导入记录"
	if _, _, found := messageFile(context.Background(), event); found {
		t.Fatal("a message without a file was taken")
	}
}
