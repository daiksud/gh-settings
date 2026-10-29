package settings

import (
	"fmt"
	"reflect"
)

// RawPattern returns the acquired identity, never the UI fallback identity.
// The enclosing state binds the evidence to its positive ID and exact name.
func (state EnvironmentPatternState) RawPattern() (EnvironmentPattern, error) {
	if state.ID <= 0 {
		return EnvironmentPattern{}, fmt.Errorf("environment deployment pattern requires a positive ID")
	}
	if _, err := normalizeEnvironmentPatterns([]EnvironmentPattern{state.Pattern}); err != nil {
		return EnvironmentPattern{}, err
	}
	raw := state.Pattern
	if record := state.RefTypeNormalization; record != (ObservationNormalization{}) {
		_, supported := EnvironmentRefTypeChoiceForValue(record.RawValue)
		if record.Path != EnvironmentRefTypePath || record.Fallback != string(DefaultEnvironmentRefType) || state.Pattern.Type != record.Fallback || supported {
			return EnvironmentPattern{}, fmt.Errorf("invalid environment deployment ref type observation provenance")
		}
		raw.Type = record.RawValue
	}
	return raw, nil
}

// Only normalized collections add acquisition invariants here. Canonical
// construction retains its existing adapter-level materialization checks.
func (state EnvironmentState) validateRefTypeNormalizations() error {
	normalized := false
	for _, pattern := range state.Patterns {
		normalized = normalized || pattern.RefTypeNormalization != (ObservationNormalization{})
	}
	if !normalized {
		return nil
	}
	if err := ValidateEnvironmentPatternObservation(state.Patterns); err != nil {
		return err
	}
	if mode := state.Policy.DeploymentBranchPolicy; mode != nil && mode.CustomBranchPolicies {
		effective, err := EffectiveEnvironmentPatterns(state.Patterns)
		if err != nil {
			// Raw acquisition is valid, but distinct identities project to a
			// collision. Keep all entries outside the active policy list.
			if state.Policy.DeploymentBranchPatterns != nil {
				return fmt.Errorf("colliding deployment ref types cannot form an effective policy list")
			}
			return nil
		}
		if state.Policy.DeploymentBranchPatterns == nil {
			return fmt.Errorf("normalized deployment patterns require their effective policy list")
		}
		policy, err := normalizeEnvironmentPatterns(*state.Policy.DeploymentBranchPatterns)
		if err != nil || !reflect.DeepEqual(policy, effective) {
			return fmt.Errorf("normalized deployment patterns differ from their effective policy list")
		}
		return nil
	}
	if !state.DeploymentModeNormalization.RawPolicy.CustomBranchPolicies || state.Policy.DeploymentBranchPatterns != nil {
		return fmt.Errorf("normalized deployment ref types require acquired active or latent custom mode")
	}
	return nil
}
