package app

import (
	"testing"
)

func TestCloudExtendedRanksPreserveCorrespondenceAndTypes(t *testing.T) {
	q := CloudInput{Mode: "self_rank", UID: "100000001", CharacterIDs: []string{"1001", "1002"}, Consent: true}
	route, body, err := cloudRequest(q)
	if err != nil || route != "rank/self" || body["type"] != "sr" {
		t.Fatal(route, body, err)
	}
	result, err := projectCloud(Game{ID: "starrail"}, q, cloudObject(t, `{"retcode":100,"rank":[{"retcode":100,"rank":20,"score":2},{"retcode":404}]}`))
	if err != nil || len(result.View.Sections) != 2 || result.View.Sections[1].Text == "" {
		t.Fatal(result, err)
	}
	if _, err = projectCloud(Game{ID: "starrail"}, q, cloudObject(t, `{"retcode":100,"rank":[{"rank":20}]}`)); err == nil {
		t.Fatal("shifted association accepted")
	}
	q.Mode = "rank"
	q.Query = "all"
	q.CharacterID = "1001"
	result, err = projectCloud(Game{ID: "starrail"}, q, []any{cloudObject(t, `{"retcode":100,"rank":1,"percent":0,"score":100}`), cloudObject(t, `{"retcode":100,"rank":10,"percent":20,"score":50}`)})
	if err != nil || len(result.View.Sections) != 2 || result.View.Sections[1].Title != "装备评分排名" {
		t.Fatal(result, err)
	}
	q = CloudInput{Mode: "group_rank", CharacterID: "10000046", UIDs: []string{"100000001", "100000002"}, Query: "mark", Consent: true}
	route, body, err = cloudRequest(q)
	if err != nil || route != "rank/group" || body["query"] != "mark" || body["data"] != nil {
		t.Fatal(body, err)
	}
	q.UIDs = append(q.UIDs, q.UIDs[0])
	if _, _, err = cloudRequest(q); err == nil {
		t.Fatal("duplicate UID accepted")
	}
}
