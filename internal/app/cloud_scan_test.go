package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
)

func TestScanGearReadsSlotsOfAReforge(t *testing.T) {
	a := App{Game: testGame(t)}
	a.Cloud.HTTP = cloudDoer(func(req *http.Request) (*http.Response, error) {
		var body map[string]any
		if req.URL.String() != "https://ark.ivny.cn/ocr/profilechange/sr" || json.NewDecoder(req.Body).Decode(&body) != nil || body["forge"] != true {
			t.Fatal("unexpected OCR request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"retcode":100,"data":[{"type":"arti1","data":{"id":61011,"level":15,"mainId":1,"attrIds":["1,2,2"]}},{"type":"arti1","data":{"id":61011,"level":15,"mainId":1,"attrIds":["1,2,2","5,1,0"]}}]}`))}, nil
	})
	gears, err := a.scanGear(t.Context(), "https://multimedia.nt.qq.com.cn/download?rkey=1", true)
	if err != nil || len(gears) != 2 || gears[0].Slot != 1 || gears[1].Slot != 1 {
		t.Fatal(gears, err)
	}
	if _, err = a.scanGear(t.Context(), "http://gchat.qpic.cn/a.png", true); err == nil {
		t.Fatal("plain HTTP image accepted")
	}
}

func TestReforgeMatchesKeptPieceBySubstats(t *testing.T) {
	game := testGame(t)
	// Without a slot in the answer, the piece's ID gives it.
	read := readGear(game, scannedGear{Gear: cloudObject(t, `{"id":61011,"level":15,"mainId":1,"attrIds":["5,1,0"]}`)})
	if read == nil || read.Slot != 1 {
		t.Fatal(read)
	}
	kept := PanelEquipment{Slot: 1, Sub: []PanelStat{{Key: "atk", Value: "3.4%"}}}
	if !sameGear(kept, *read) {
		t.Fatal("the game's shown value did not match")
	}
	for _, other := range []PanelStat{{Key: "atk", Value: "4.3%"}, {Key: "hp", Value: "3.4%"}} {
		if sameGear(PanelEquipment{Slot: 1, Sub: []PanelStat{other}}, *read) {
			t.Fatal("another piece matched", other)
		}
	}
	// Speed may be a few rolls apart, as ark-plugin allows.
	relic := PanelEquipment{Sub: []PanelStat{{Key: "speed", Value: "4.6"}, {Key: "cpct", Value: "5.8%"}}}
	if !sameGear(PanelEquipment{Sub: []PanelStat{{Key: "speed", Value: "6"}, {Key: "cpct", Value: "5.8%"}}}, relic) {
		t.Fatal("speed within ark-plugin's range did not match")
	}
	if sameGear(PanelEquipment{Sub: []PanelStat{{Key: "speed", Value: "7"}, {Key: "cpct", Value: "5.8%"}}}, relic) {
		t.Fatal("distant speed matched")
	}
}

func TestDistributionKeepsArkPercentileScores(t *testing.T) {
	data := cloudObject(t, `{"retcode":100,"data":{"name":"重击伤害","total":1234,"scores":[98000,90000,85000.5,80000,76000,70000,62000,50000,40000,30000]}}`)
	result, err := projectCloud(testGame(t), CloudInput{Mode: "distribution", CharacterID: "1102"}, data)
	if err != nil || result.Stats == nil || result.Stats.Title != "重击伤害" || result.Stats.Total != "1234" || len(result.Stats.Scores) != 10 || result.Stats.Scores[2] != 85000.5 {
		t.Fatal(result.Stats, err)
	}
	if text := result.View.Text(); !strings.Contains(text, "TOP 1%：98000") || !strings.Contains(text, "TOP 99%：30000") {
		t.Fatal(text)
	}
}

// The host hands a quoted message's ID over as message_id, OneBot's id
// renamed.
func TestQuotedMessageReadsTheHostReplySegment(t *testing.T) {
	event := &rayleabot.EventContext{Event: rayleabot.Event{Message: rayleabot.Message{Segments: []rayleabot.Segment{
		{Type: "reply", Data: map[string]any{"message_id": "98765"}}, rayleabot.Text("面板换装"),
	}}}}
	if got := quotedMessage(event); got != "98765" {
		t.Fatalf("quoted %q", got)
	}
	event.Event.Message.Segments = event.Event.Message.Segments[1:]
	if got := quotedMessage(event); got != "" {
		t.Fatalf("quoted %q without a reply", got)
	}
}
