package settings

import (
	"reflect"
	"testing"
)

func normalizedPatternEnvironment(t *testing.T, collision bool) EnvironmentState {
	t.Helper()
	state := EnvironmentState{ID: 7, Policy: environmentPolicy(t, "production"), Patterns: []EnvironmentPatternState{normalizedPattern(31, "future", "main")}}
	state.Policy.DeploymentBranchPolicy = &EnvironmentBranchPolicy{CustomBranchPolicies: true}
	if collision {
		state.Patterns = append(state.Patterns, EnvironmentPatternState{ID: 32, Pattern: EnvironmentPattern{Type: "branch", Name: "main"}})
	} else {
		patterns := []EnvironmentPattern{{Type: "branch", Name: "main"}}
		state.Policy.DeploymentBranchPatterns = &patterns
	}
	return state
}

func TestEnvironmentRefTypeStateRetainsOwnedActiveAndLatentEvidence(t *testing.T) {
	for _, collision := range []bool{false, true} {
		state := normalizedPatternEnvironment(t, collision)
		latent := normalizedEnvironmentModeState(t, 9, "staging", true)
		latent.Patterns = append([]EnvironmentPatternState{}, state.Patterns...)
		observed, err := NewEnvironmentObservation([]EnvironmentState{state, latent})
		if err != nil {
			t.Fatal("valid raw collection rejected", collision, err)
		}
		state.Patterns[0].RefTypeNormalization.RawValue = "changed input"
		values, err := observed.Values()
		if err != nil || values[0].Patterns[0].RefTypeNormalization.RawValue != "future" || values[1].Policy.DeploymentBranchPatterns != nil {
			t.Fatal("evidence lost or latent projection activated", values, err)
		}
		if _, err := values[0].RawDeploymentBranchPolicy(); err != nil {
			t.Fatal("valid companion mode rejected", err)
		}
		if _, err := values[1].RawDeploymentBranchPolicy(); err != nil {
			t.Fatal("latent raw types rejected", err)
		}
		values[0].Patterns[0].RefTypeNormalization.RawValue = "changed output"
		fresh, err := observed.Values()
		if err != nil || fresh[0].Patterns[0].RefTypeNormalization.RawValue != "future" {
			t.Fatal("output aliases evidence", fresh, err)
		}
	}
}

func TestEnvironmentRefTypeStateRejectsForgedCollectionsAndProjection(t *testing.T) {
	for _, mutate := range []func(*EnvironmentState){
		func(s *EnvironmentState) { s.Patterns[0].ID = 0 },
		func(s *EnvironmentState) { s.Patterns[0].RefTypeNormalization.Path = "wrong" },
		func(s *EnvironmentState) { s.Patterns = append(s.Patterns, s.Patterns[0]) },
		func(s *EnvironmentState) { s.Policy.DeploymentBranchPatterns = nil },
		func(s *EnvironmentState) { *s.Policy.DeploymentBranchPatterns = []EnvironmentPattern{} },
		func(s *EnvironmentState) { (*s.Policy.DeploymentBranchPatterns)[0].Name = "other" },
		func(s *EnvironmentState) { s.Policy.DeploymentBranchPolicy = nil },
		func(s *EnvironmentState) {
			s.Policy.DeploymentBranchPolicy = &EnvironmentBranchPolicy{ProtectedBranches: true}
		},
	} {
		state := normalizedPatternEnvironment(t, false)
		mutate(&state)
		got, err := NewEnvironmentObservation([]EnvironmentState{{ID: 9, Policy: environmentPolicy(t, "valid")}, state})
		if err == nil || !reflect.DeepEqual(got, EnvironmentObservation{}) {
			t.Fatal("invalid state accepted", state, got, err)
		}
		if raw, err := state.RawDeploymentBranchPolicy(); err == nil || raw != nil {
			t.Fatal("invalid companion provenance escaped", raw, err)
		}
	}
	collision := normalizedPatternEnvironment(t, true)
	patterns := []EnvironmentPattern{{Type: "branch", Name: "main"}}
	collision.Policy.DeploymentBranchPatterns = &patterns
	if _, err := NewEnvironmentObservation([]EnvironmentState{collision}); err == nil {
		t.Fatal("colliding entries collapsed into policy list")
	}
}
