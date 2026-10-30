package github

import (
	"cmp"
	"slices"

	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *EnvironmentsClient) warnRefTypeNormalizations(state settings.EnvironmentState, requested settings.EnvironmentDeclaration) {
	if client.RefTypeNormalizationWarning == nil || requested.DeploymentBranchPatterns == nil || settings.EnvironmentIdentity(state.Policy.Name) != settings.EnvironmentIdentity(requested.Name) {
		return
	}
	mode, err := state.RawDeploymentBranchPolicy()
	if err != nil || mode == nil || *mode != (settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}) {
		return
	}
	if err := settings.ValidateEnvironmentPatternObservation(state.Patterns); err != nil {
		return
	}
	patterns := slices.Clone(state.Patterns)
	slices.SortFunc(patterns, func(a, b settings.EnvironmentPatternState) int { return cmp.Compare(a.ID, b.ID) })
	for _, pattern := range patterns {
		if pattern.RefTypeNormalization != (settings.ObservationNormalization{}) {
			client.RefTypeNormalizationWarning(state.Policy.Name, pattern)
		}
	}
}
