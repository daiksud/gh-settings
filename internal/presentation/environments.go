package presentation

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

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
	if patterns := change.PatternObservations(); patterns != nil {
		slices.SortFunc(patterns, func(a, b settings.EnvironmentPatternState) int {
			return cmp.Compare(a.ID, b.ID)
		})
		var details []string
		for _, pattern := range patterns {
			if pattern.RefTypeNormalization != (settings.ObservationNormalization{}) {
				details = append(details, EnvironmentRefTypeNormalizationLine(change.Name(), pattern))
			}
		}
		type requestedPattern struct {
			RefType     string `json:"ref_type"`
			NamePattern string `json:"name_pattern"`
		}
		requested := make([]requestedPattern, 0, len(*change.After().DeploymentBranchPatterns))
		for _, pattern := range *change.After().DeploymentBranchPatterns {
			requested = append(requested, requestedPattern{pattern.Type, pattern.Name})
		}
		request, _ := json.Marshal(requested)
		details = append(details, fmt.Sprintf("%s (Environment %q): normalized pattern observation → request %s (write required; raw identities differ)", settings.EnvironmentRefTypePath, change.Name(), request))
		if change.Before().DeploymentBranchPatterns == nil {
			details = append(details, "effective deployment pattern list unavailable (normalized ref types collide; acquired raw identities retained)")
		}
		return strings.Join(details, "\n") + "\n" + line
	}
	return line
}

func EnvironmentDeploymentNormalizationLine(name string, record settings.EnvironmentDeploymentNormalization) string {
	raw, _ := json.Marshal(record.RawPolicy)
	return fmt.Sprintf("Warning: %s: observed %s is outside UI choices; using %q (projection only). Environment %q.", record.Path, raw, record.Fallback, name)
}

func EnvironmentRefTypeNormalizationLine(name string, pattern settings.EnvironmentPatternState) string {
	return fmt.Sprintf("%s Environment %q, pattern %d name_pattern %q.", NormalizationLine(pattern.RefTypeNormalization), name, pattern.ID, pattern.Pattern.Name)
}
