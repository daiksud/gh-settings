package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func EnvironmentLine(change settings.EnvironmentChange) string {
	before, _ := json.Marshal(change.Before())
	after, _ := json.Marshal(change.After())
	line := fmt.Sprintf("settings.environments: %s %q: effective before policy %s → request %s", change.Operation(), change.Name(), before, after)
	if requested := change.After(); requested != nil && settings.EnvironmentHasAPIOmissions(*requested) {
		line += " (omitted fields are API-controlled; resulting values are unknown until apply)"
	}
	if record := change.DeploymentModeNormalization(); record != (settings.EnvironmentDeploymentNormalization{}) {
		choice, _ := settings.EnvironmentDeploymentChoiceForPolicy(change.After().DeploymentBranchPolicy)
		decision := fmt.Sprintf("%s (Environment %q): %q (normalized observation) → %q (write required)", record.Path, change.Name(), record.Fallback, choice)
		return EnvironmentDeploymentNormalizationLine(change.Name(), record) + "\n" + decision + "\n" + line
	}
	return line
}

func EnvironmentDeploymentNormalizationLine(name string, record settings.EnvironmentDeploymentNormalization) string {
	raw, _ := json.Marshal(record.RawPolicy)
	return fmt.Sprintf("Warning: %s: observed %s is outside UI choices; using %q (projection only). Environment %q.", record.Path, raw, record.Fallback, name)
}
