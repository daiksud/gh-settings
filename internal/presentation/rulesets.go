package presentation

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RulesetLine renders selected whole objects, including IDs and fixed defaults.
func RulesetLine(change settings.RulesetDisplayChange) string {
	prior, desired := change.Before(), change.After()
	before, _ := json.Marshal(prior)
	after, _ := json.Marshal(desired)
	line := fmt.Sprintf("settings.rulesets: %s %q: %s → %s", change.Operation(), change.Name(), before, after)
	if record := change.EnforcementNormalization(); record != (settings.ObservationNormalization{}) {
		id := *prior.ID
		decision := fmt.Sprintf("%s (Ruleset %d %q): %s (normalized observation) → %s (write required)", record.Path, id, change.Name(), strconv.Quote(record.Fallback), strconv.Quote(*desired.Enforcement))
		return RulesetNormalizationLine(id, change.Name(), record) + "\n" + decision + "\nEffective policy: " + line
	}
	return line
}

func RulesetNormalizationLine(id int64, name string, record settings.ObservationNormalization) string {
	return fmt.Sprintf("%s Ruleset %d %q.", NormalizationLine(record), id, name)
}
