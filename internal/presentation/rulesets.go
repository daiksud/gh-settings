package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RulesetLine renders selected whole objects, including IDs and fixed defaults.
func RulesetLine(change settings.RulesetDisplayChange) string {
	before, _ := json.Marshal(change.Before())
	after, _ := json.Marshal(change.After())
	line := fmt.Sprintf("settings.rulesets: %s %q: %s → %s", change.Operation(), change.Name(), before, after)
	return line
}
