package app

import (
	"reflect"
	"testing"
)

func TestPanelChangeReadsMiaoWording(t *testing.T) {
	catalog, err := ParseCatalog([]byte(`{"version":"1","entries":[
{"id":"1102","name":"希儿","kind":"character"},{"id":"1006","name":"银狼","kind":"character"},
{"id":"23001","name":"于夜色中","kind":"weapon","aliases":["夜色"]}],
"artifact_sets":{"61011":"野穗伴行的快枪手","63055":"太空封印站"},
"artifact_pieces":{"野穗伴行的快枪手":["快枪手的野穗毡帽","快枪手的粗革手套","快枪手的猎风披肩","快枪手的铆钉马靴"],"太空封印站":["","","","","「日月」的封印站","「日月」的漂泊飘带"]},
"set_abbrs":{"野穗伴行的快枪手":"快枪手","太空封印站":"封印站"}}`))
	if err != nil {
		t.Fatal(err)
	}
	change, ok := parsePanelChange(catalog, nil, "希儿换80级5叠影夜色换快枪手4+封印站2换天赋6 10 10 10")
	if !ok || change.CharacterID != "1102" || change.Weapon == nil || change.Weapon.ID != "23001" || change.Weapon.Level != 80 || change.Weapon.Refinement != 5 {
		t.Fatalf("light cone: %+v", change)
	}
	// The relic set fills the first four slots, the planar set the last two.
	if !reflect.DeepEqual(change.Sets, []string{"野穗伴行的快枪手", "野穗伴行的快枪手", "太空封印站"}) || !reflect.DeepEqual(change.Talents, map[string]int{"a": 6, "e": 10, "t": 10, "q": 10}) {
		t.Fatalf("sets and talents: %+v", change)
	}
	change, _ = parsePanelChange(catalog, nil, "希儿换银狼遗器+光锥")
	if change.Artifacts == nil || change.Artifacts.CharacterID != "1006" || change.WeaponFrom == nil || change.WeaponFrom.CharacterID != "1006" {
		t.Fatalf("parts of another character: %+v", change)
	}
	change, _ = parsePanelChange(catalog, nil, "100000001希儿换银狼头")
	if change.UID != "100000001" || change.Pieces[1] != (changeSource{CharacterID: "1006"}) {
		t.Fatalf("a piece: %+v", change)
	}
	change, _ = parsePanelChange(catalog, nil, "希儿换满命换满行迹换70级")
	if change.Cons == nil || *change.Cons != 6 || len(change.Trees) != 15 || change.Level != 70 {
		t.Fatalf("eidolons, traces and level: %+v", change)
	}
	if change, _ = parsePanelChange(catalog, nil, "希儿换银狼"); change.Swap != "1006" {
		t.Fatalf("another character: %+v", change)
	}
	if _, ok = parsePanelChange(catalog, nil, "希儿换一个好心情"); ok {
		t.Fatal("a word without a change was read")
	}
	if _, ok = parsePanelChange(catalog, nil, "兑换码"); ok {
		t.Fatal("a word without a character was read")
	}
}
