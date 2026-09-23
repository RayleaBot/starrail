package app

import "testing"

func TestCardQueryFollowsMiaoCheck(t *testing.T) {
	for text, want := range map[string][2]string{
		"希儿":            {"希儿", ""},
		"希儿卡片100000001": {"希儿", "100000001"},
		"老婆卡芙卡":         {"卡芙卡", ""},
		"希儿 187654321":  {"希儿", "187654321"},
		"希儿1876543210":  {"希儿", "1876543210"},
	} {
		if name, uid := cardQuery(text); name != want[0] || uid != want[1] {
			t.Errorf("cardQuery(%q) = %q, %q; want %q, %q", text, name, uid, want[0], want[1])
		}
	}
}
