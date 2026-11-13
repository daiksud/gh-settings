package settings

import (
	"strings"
	"testing"
)

func TestEnvironmentModeChangePreservesOmittedPatterns(t *testing.T) {
	for _, mode := range []*EnvironmentBranchPolicy{nil, {ProtectedBranches: true}} {
		for _, count := range []int{0, 1} {
			before := environmentPolicy(t, "production")
			before.DeploymentBranchPolicy = &EnvironmentBranchPolicy{CustomBranchPolicies: true}
			patterns := []EnvironmentPatternState{}
			values := []EnvironmentPattern{}
			if count > 0 {
				values = append(values, EnvironmentPattern{Type: "branch", Name: "release/*"})
				patterns = append(patterns, EnvironmentPatternState{ID: 7, Pattern: values[0]})
			}
			before.DeploymentBranchPatterns = &values
			state := EnvironmentState{CustomRules: []EnvironmentCustomRule{}, ID: 1, Policy: before, Patterns: patterns}
			observed, err := NewEnvironmentObservation([]EnvironmentState{state})
			if err != nil {
				t.Fatal(err)
			}
			request := EnvironmentDeclaration{Name: "production", DeploymentBranchPolicyDeclared: true, DeploymentBranchPolicy: mode}
			desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{request})
			if err != nil {
				t.Fatal(err)
			}
			changes, err := ReconcileEnvironments(observed, &desired)
			if count > 0 {
				if err == nil || len(changes) != 0 || !strings.Contains(err.Error(), "omitted deployment patterns") {
					t.Fatalf("mode change can erase unowned patterns: changes=%v err=%v", changes, err)
				}
			} else if err != nil || len(changes) != 1 {
				t.Fatal("known-empty mode transition rejected", changes, err)
			}
			_, err = PrepareEnvironmentUpdate(state, request)
			if (err != nil) != (count > 0) {
				t.Fatal("send-time preservation differs", count, err)
			}
		}
	}
}

func TestEnvironmentModeGuardUsesRawPatternEvidence(t *testing.T) {
	state := normalizedEnvironmentModeState(t, 7, "production", true)
	request := EnvironmentDeclaration{Name: "production", DeploymentBranchPolicyDeclared: true}
	if _, err := PrepareEnvironmentUpdate(state, request); err == nil {
		t.Fatal("latent raw patterns erased by effective fallback")
	}
	state.Policy = environmentPolicy(t, "production")
	state.DeploymentModeNormalization = EnvironmentDeploymentNormalization{}
	state.Policy.DeploymentBranchPolicy = &EnvironmentBranchPolicy{CustomBranchPolicies: true}
	state.Patterns = []EnvironmentPatternState{
		{ID: 7, Pattern: EnvironmentPattern{Type: "branch", Name: "same"}, RefTypeNormalization: ObservationNormalization{Path: EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}},
		{ID: 8, Pattern: EnvironmentPattern{Type: "branch", Name: "same"}},
	}
	if _, err := PrepareEnvironmentUpdate(state, request); err == nil || !strings.Contains(err.Error(), "omitted deployment patterns") {
		t.Fatal("projection collision hid raw patterns", err)
	}
	request.DeploymentBranchPolicy = &EnvironmentBranchPolicy{CustomBranchPolicies: true}
	if _, err := PrepareEnvironmentUpdate(state, request); err != nil {
		t.Fatal("selected-to-selected preservation rejected", err)
	}
	request.DeploymentBranchPolicyDeclared, request.DeploymentBranchPolicy = false, nil
	if _, err := PrepareEnvironmentUpdate(state, request); err != nil {
		t.Fatal("omitted mode changed", err)
	}
	request.DeploymentBranchPolicyDeclared = true
	state.Patterns = nil
	if _, err := PrepareEnvironmentUpdate(state, request); err == nil {
		t.Fatal("missing pattern acquisition became empty")
	}
}
