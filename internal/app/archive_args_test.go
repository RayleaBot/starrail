package app

import "testing"

func TestArchiveArgs(t *testing.T) {
	for _, tc := range []struct {
		text  string
		index int
		uid   string
	}{
		{"常规演算", 1, ""}, {"本期演算3", 3, ""}, {"货币战绩10", 10, ""}, {"常规演算0", 1, ""},
		{"常规演算123456789", 1, "123456789"}, {"常规演算1 123456789", 1, "123456789"}, {"货币战绩10 123456789", 10, "123456789"},
	} {
		index, uid := ArchiveArgs(tc.text)
		if index != tc.index || uid != tc.uid {
			t.Errorf("%s = %d %q", tc.text, index, uid)
		}
	}
}

func TestArchiveCommandRejectsOtherWords(t *testing.T) {
	host := newSDKHost(t, pluginApp(t), nil)
	end, _ := host.message("*本期演算 上周 12", "本期演算", "上周", "12")
	if asObject(end["data"])["handled"] != false || len(host.sent) != 0 {
		t.Fatal(end, host.sent)
	}
}
