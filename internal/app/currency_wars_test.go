package app

import "testing"

func TestCurrencyWarsRecordReadsTheClosingNumeral(t *testing.T) {
	for word, want := range map[string]int{"货币战绩": 1, "币战回顾一": 1, "货币战争记录三": 3, "货币战报十": 10} {
		if got := CurrencyWarsRecord(word); got != want {
			t.Errorf("%s = %d, want %d", word, got, want)
		}
	}
}
