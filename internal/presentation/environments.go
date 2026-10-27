package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func EnvironmentLine(change settings.EnvironmentChange) string {
	before, _ := json.Marshal(change.Before())
	after, _ := json.Marshal(change.After())
	line := fmt.Sprintf("settings.environments: %s %q: %s → request %s", change.Operation(), change.Name(), before, after)
	if requested := change.After(); requested != nil && settings.EnvironmentHasAPIOmissions(*requested) {
		line += " (omitted fields are API-controlled; resulting values are unknown until apply)"
	}
	return line
}
