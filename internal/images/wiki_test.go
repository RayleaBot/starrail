package images

import "testing"

// The expected strings are what JavaScript's toFixed and miao's Format print.
func TestMiaoFormatsFollowJavaScript(t *testing.T) {
	for _, tc := range []struct {
		value  float64
		digits int
		want   string
	}{{106.25, 1, "106.3"}, {178.75, 1, "178.8"}, {1.005, 2, "1.00"}, {0.5, 0, "1"}, {931.392, 1, "931.4"}, {-100, 1, "-100.0"}, {-0.04, 1, "-0.0"}} {
		if got := jsFixed(tc.value, tc.digits); got != tc.want {
			t.Errorf("jsFixed(%v, %d) = %s, want %s", tc.value, tc.digits, got, tc.want)
		}
	}
	if got := miaoComma(115, 1); got != "115.0" {
		t.Errorf("miaoComma(115) = %s", got)
	}
	if got := miaoComma(12345.6, 0); got != "12,346" {
		t.Errorf("miaoComma(12345.6) = %s", got)
	}
}

func TestWikiFillFollowsMiaoGetDesc(t *testing.T) {
	tables := map[string]struct {
		Name   string
		IsSame bool
		Values []float64
	}{"1": {Name: "技能伤害", Values: []float64{1.1, 1.21, 1.32, 1.43, 1.54, 1.65, 1.7875, 1.925, 2.0625}}, "2": {Name: "回合数", Values: []float64{2, 2, 2, 2, 2, 2, 2, 2, 2}}}
	desc, out := wikiFill("造成<nobr>$1[i]%</nobr>攻击力的伤害，持续<nobr>$2[i]</nobr>回合。<br>第二行", tables, 8)
	if len(desc) != 2 || desc[0] != "造成<nobr>192.5% (lv8)</nobr>攻击力的伤害，持续<nobr>2 (lv8)</nobr>回合。" {
		t.Errorf("desc = %q", desc)
	}
	if out[0].values[6] != "178.8%" || out[1].values[0] != "2" {
		t.Errorf("tables = %v", out)
	}
}
