package settings

import (
	"reflect"
	"testing"
)

func TestEnvironmentEffectivePolicyProjectsActiveRefTypesAndExcludesLatents(t *testing.T) {
	state := normalizedPatternEnvironment(t, false)
	got, err := state.EffectivePolicy()
	if err != nil || !reflect.DeepEqual(got, state.Policy) {
		t.Fatal(got, err)
	}
	*got.WaitTimer = 99
	(*got.DeploymentBranchPatterns)[0].Name = "changed"
	again, err := state.EffectivePolicy()
	if err != nil || *again.WaitTimer != 0 || (*again.DeploymentBranchPatterns)[0].Name != "main" {
		t.Fatal("projection aliases state", again, err)
	}
	latent := normalizedEnvironmentModeState(t, 9, "staging", true)
	latent.Patterns = normalizedPatternEnvironment(t, true).Patterns
	got, err = latent.EffectivePolicy()
	if err != nil || got.DeploymentBranchPatterns != nil || got.DeploymentBranchPolicy != nil {
		t.Fatal("latent colliding patterns activated", got, err)
	}
}

func TestEnvironmentEffectivePolicyFailsCollisionAndInvalidStateWithoutPartialPolicy(t *testing.T) {
	bad := normalizedPatternEnvironment(t, false)
	bad.Patterns[0].ID = 0
	for _, state := range []EnvironmentState{{}, bad, normalizedPatternEnvironment(t, true)} {
		if got, err := state.EffectivePolicy(); err == nil || !reflect.DeepEqual(got, EnvironmentDeclaration{}) {
			t.Fatal("invalid or colliding projection returned policy", got, err)
		}
	}
}

func TestEnvironmentEffectivePolicyPreservesCanonicalProjectionBoundary(t *testing.T) {
	for _, knownEmpty := range []bool{false, true} {
		state := EnvironmentState{ID: 7, Policy: environmentPolicy(t, "production")}
		state.Policy.DeploymentBranchPolicy = &EnvironmentBranchPolicy{CustomBranchPolicies: true}
		if knownEmpty {
			patterns := []EnvironmentPattern{}
			state.Policy.DeploymentBranchPatterns = &patterns
		}
		got, err := state.EffectivePolicy()
		if err != nil || !reflect.DeepEqual(got, state.Policy) || state.Patterns != nil {
			t.Fatal("projection invented acquisition or changed canonical behavior", got, err)
		}
	}
}
