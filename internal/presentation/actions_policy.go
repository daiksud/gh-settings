package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func ActionsPolicyLine(change settings.ActionsPolicyChange) string {
	before, after := change.Values()
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	value := string(b)
	if before == nil {
		value = "(unavailable while policy is not selected)"
	}
	return fmt.Sprintf("settings.%s: %s → %s", change.Field(), value, a)
}
