package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RulesetLine renders only declaration-owned values. Collection removals remain
// visible through their identities, while omitted attributes stay absent.
func RulesetLine(change settings.RulesetDisplayChange) string {
	if change.Operation() == "delete" {
		return fmt.Sprintf("settings.rulesets: delete %q (removes repository protection)", change.Name())
	}
	before, _ := json.Marshal(change.Before())
	after, _ := json.Marshal(change.After())
	line := fmt.Sprintf("settings.rulesets: %s %q: %s → %s", change.Operation(), change.Name(), before, after)
	return line
}
