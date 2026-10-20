package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RulesetLine renders complete before/after policy so nested removals stay visible.
func RulesetLine(change settings.RulesetChange) string {
	before, _ := json.Marshal(change.Before())
	after, _ := json.Marshal(change.After())
	line := fmt.Sprintf("settings.rulesets: %s %q: %s → %s", change.Operation(), change.Name(), before, after)
	if change.Operation() == "delete" {
		line += " (removes repository protection)"
	}
	return line
}
