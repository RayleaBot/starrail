package app

import (
	"strings"
	"testing"
)

func TestCloudOCRURLAndGearProjection(t *testing.T) {
	q := CloudInput{Mode: "ocr", Consent: true, ImageURL: "https://images.example.com/gear.png", Slot: 1, Forge: true}
	route, body, err := cloudRequest(q)
	if err != nil || route != "ocr/profilechange/sr" || body["forge"] != true {
		t.Fatal(route, body, err)
	}
	for _, url := range []string{"http://example.com/a.png", "https://127.0.0.1/a.png", "https://user:pass@example.com/a.png", "https://example.com/a.png?authkey=secret", "https://localhost/a.png"} {
		q.ImageURL = url
		if _, _, err = cloudRequest(q); err == nil {
			t.Fatal("invalid image URL accepted")
		}
	}
	data := cloudObject(t, `{"retcode":100,"data":[{"data":{"id":61011,"level":15,"mainId":1,"attrIds":["1,2,2"]}},{"id":61011,"level":15,"mainId":1,"attrIds":["1,2,2","5,1,0"],"token":"not-for-output"}]}`)
	result, err := projectCloud(testGame(t), q, data)
	if err != nil || result.OCR == nil || result.OCR.Replacement == nil || result.OCR.NeedsIdentity {
		t.Fatal(result, err)
	}
	if len(result.OCR.Replacement.Main) != 1 || result.OCR.Replacement.SetName != "云无留迹的过客" || !strings.Contains(result.View.Text(), "大攻击：3.46%") {
		t.Fatal(result)
	}
	if result.OCR.replacement["token"] != nil {
		t.Fatal("unknown OCR metadata persisted")
	}
	raw := asObject(asList(data["data"])[1])
	delete(raw, "id")
	result, err = projectCloud(testGame(t), q, data)
	if err != nil || !result.OCR.NeedsIdentity {
		t.Fatal("missing identity was invented", err)
	}
}
