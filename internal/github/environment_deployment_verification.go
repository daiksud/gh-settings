package github

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/daiksud/gh-settings/internal/settings"
)

// verifyEnvironmentRequestedState confirms requested values against a completed
// acquisition. Prepared existing-update snapshots also select sent companions.
func verifyEnvironmentRequestedState(actual settings.EnvironmentState, requested settings.EnvironmentDeclaration) error {
	rawMode, err := actual.RawDeploymentBranchPolicy()
	if err != nil {
		return err
	}
	if requested.DeploymentBranchPolicyDeclared || requested.DeploymentBranchPolicy != nil {
		if !reflect.DeepEqual(rawMode, requested.DeploymentBranchPolicy) {
			return fmt.Errorf("environment raw deployment mode differs from requested policy")
		}
	}
	if requested.DeploymentBranchPatterns != nil {
		if rawMode == nil || rawMode.ProtectedBranches || !rawMode.CustomBranchPolicies {
			return fmt.Errorf("requested deployment patterns require acquired selected mode")
		}
		if err := settings.ValidateEnvironmentPatternObservation(actual.Patterns); err != nil {
			return err
		}
		changes, err := settings.ReconcileEnvironmentPatterns(actual.Patterns, requested.DeploymentBranchPatterns)
		if err != nil {
			return err
		}
		if len(changes) != 0 {
			return fmt.Errorf("environment raw deployment patterns differ from requested patterns")
		}
	}
	if !settings.EnvironmentMatchesRequest(actual.Policy, requested) {
		return fmt.Errorf("environment readback differs from requested policy")
	}
	return nil
}

// verifyEnvironmentUpdateResponse validates preliminary detail data without
// inventing pattern acquisition or certifying omitted creation outcomes.
func verifyEnvironmentUpdateResponse(raw map[string]json.RawMessage, requested settings.EnvironmentDeclaration) error {
	actual, record, err := decodeObservedEnvironment(raw)
	if err != nil {
		return err
	}
	if requested.DeploymentBranchPolicyDeclared || requested.DeploymentBranchPolicy != nil {
		rawMode := actual.Policy.DeploymentBranchPolicy
		if record != (settings.EnvironmentDeploymentNormalization{}) {
			rawMode = &record.RawPolicy
		}
		if !reflect.DeepEqual(rawMode, requested.DeploymentBranchPolicy) {
			return fmt.Errorf("environment acknowledgment raw deployment mode differs from requested policy")
		}
	}
	if !settings.EnvironmentMatchesRequest(actual.Policy, environmentBuiltins(requested)) {
		return fmt.Errorf("environment acknowledgment differs from requested controls or sent companions")
	}
	return nil
}

func environmentBuiltins(policy settings.EnvironmentDeclaration) settings.EnvironmentDeclaration {
	policy.DeploymentBranchPatterns = nil
	return policy
}

// Omitted patterns remain unowned. Verify their exact active raw identities when
// selected mode stays selected, without treating projections as declarations.
// Explicit mode transitions retain the separate inactive-pattern limitation.
func verifyPreservedEnvironmentPatterns(before, actual settings.EnvironmentState, intent settings.EnvironmentDeclaration) error {
	if intent.DeploymentBranchPatterns != nil {
		return nil
	}
	beforeMode, err := before.RawDeploymentBranchPolicy()
	if err != nil {
		return err
	}
	actualMode, err := actual.RawDeploymentBranchPolicy()
	if err != nil {
		return err
	}
	selected := settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
	if beforeMode == nil || *beforeMode != selected || actualMode == nil || *actualMode != selected {
		return nil
	}
	identities := func(patterns []settings.EnvironmentPatternState) (map[int64]settings.EnvironmentPattern, error) {
		if err := settings.ValidateEnvironmentPatternObservation(patterns); err != nil {
			return nil, err
		}
		values := make(map[int64]settings.EnvironmentPattern, len(patterns))
		for _, pattern := range patterns {
			raw, err := pattern.RawPattern()
			if err != nil {
				return nil, err
			}
			values[pattern.ID] = raw
		}
		return values, nil
	}
	want, err := identities(before.Patterns)
	if err != nil {
		return err
	}
	got, err := identities(actual.Patterns)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, got) {
		return fmt.Errorf("unowned active deployment pattern identities changed")
	}
	return nil
}
