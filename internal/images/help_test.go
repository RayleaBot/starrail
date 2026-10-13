package images

import (
	"encoding/json"
	"testing"
)

// Each written form is drawn on its own, with its placeholders set apart
// from the words to type.
func TestHelpFormsSetPlaceholdersApart(t *testing.T) {
	text := func(s string) map[string]any { return map[string]any{"text": s} }
	arg := func(s string) map[string]any { return map[string]any{"text": s, "arg": true} }
	want, _ := json.Marshal([]any{
		[]any{text("*"), arg("<角色>"), text("面板 "), arg("[UID]")},
		[]any{text("*面板列表")},
		[]any{text("*十连"), arg("[角色|光锥|常驻]")},
	})
	got, _ := json.Marshal(helpForms("*<角色>面板 [UID] / *面板列表 / *十连[角色|光锥|常驻]"))
	if string(got) != string(want) {
		t.Errorf("forms = %s", got)
	}
}
