package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func normalizedEnvironmentModeState(t *testing.T, id int64, name string, raw bool) EnvironmentState {
	t.Helper()
	state := EnvironmentState{ID: id, Policy: environmentPolicy(t, name), DeploymentModeNormalization: EnvironmentDeploymentNormalization{Path: EnvironmentDeploymentModePath, RawPolicy: EnvironmentBranchPolicy{ProtectedBranches: raw, CustomBranchPolicies: raw}, Fallback: uimodel.NoRestriction}}
	if raw {
		state.Patterns = []EnvironmentPatternState{{ID: 31, Pattern: EnvironmentPattern{Type: "branch", Name: "release/*"}}, {ID: 32, Pattern: EnvironmentPattern{Type: "tag", Name: "v*"}}}
	}
	return state
}

func TestEnvironmentDeploymentObservationOwnsRawPairAndLatentPatterns(t *testing.T) {
	for _, raw := range []bool{false, true} {
		state := normalizedEnvironmentModeState(t, 7, "production", raw)
		other := normalizedEnvironmentModeState(t, 9, "staging", !raw)
		observed, err := NewEnvironmentObservation([]EnvironmentState{state, other})
		if err != nil {
			t.Fatal(err)
		}
		state.DeploymentModeNormalization.RawPolicy = EnvironmentBranchPolicy{ProtectedBranches: true}
		*state.Policy.WaitTimer = 999
		if raw {
			state.Patterns[0].Pattern.Name = "mutated input"
		}
		values, err := observed.Values()
		if err != nil || len(values) != 2 {
			t.Fatal(values, err)
		}
		for i, want := range []bool{raw, !raw} {
			got, err := values[i].RawDeploymentBranchPolicy()
			if err != nil || got == nil || *got != (EnvironmentBranchPolicy{ProtectedBranches: want, CustomBranchPolicies: want}) {
				t.Fatal("per-name/ID raw pair lost", got, err)
			}
			*got = EnvironmentBranchPolicy{ProtectedBranches: true}
			again, err := values[i].RawDeploymentBranchPolicy()
			if err != nil || again == nil || *again != (EnvironmentBranchPolicy{ProtectedBranches: want, CustomBranchPolicies: want}) {
				t.Fatal("raw policy pointer alias", again, err)
			}
			if values[i].Policy.DeploymentBranchPolicy != nil || !values[i].Policy.DeploymentBranchPolicyDeclared || values[i].Policy.DeploymentBranchPatterns != nil {
				t.Fatal("latent policy became active intent", values[i])
			}
			if want && (len(values[i].Patterns) != 2 || values[i].Patterns[0].Pattern.Name != "release/*") {
				t.Fatal("latent acquisition lost", values[i])
			}
		}
		values[0].DeploymentModeNormalization.RawPolicy.ProtectedBranches = !raw
		*values[0].Policy.WaitTimer = 50
		if raw {
			values[0].Patterns[0].Pattern.Name = "mutated output"
		}
		fresh, err := observed.Values()
		if err != nil || fresh[0].DeploymentModeNormalization.RawPolicy.ProtectedBranches != raw || *fresh[0].Policy.WaitTimer != 0 || raw && fresh[0].Patterns[0].Pattern.Name != "release/*" {
			t.Fatal("observation clone alias", fresh, err)
		}
		if got, err := fresh[1].RawDeploymentBranchPolicy(); err != nil || got == nil || got.ProtectedBranches == raw {
			t.Fatal("other identity corrupted", got, err)
		}
	}
	canonical := EnvironmentState{ID: 7, Policy: environmentPolicy(t, "production")}
	if got, err := canonical.RawDeploymentBranchPolicy(); err != nil || got != nil {
		t.Fatal("known null invented pair", got, err)
	}
	for _, policy := range []*EnvironmentBranchPolicy{{ProtectedBranches: true}, {CustomBranchPolicies: true}} {
		expected := *policy
		canonical.Policy.DeploymentBranchPolicy = policy
		got, err := canonical.RawDeploymentBranchPolicy()
		if err != nil || got == nil || *got != *policy {
			t.Fatal("canonical pair changed", got, err)
		}
		*got = EnvironmentBranchPolicy{}
		if *canonical.Policy.DeploymentBranchPolicy != expected {
			t.Fatal("canonical pointer alias")
		}
	}
	if got, err := (EnvironmentState{}).RawDeploymentBranchPolicy(); err == nil || got != nil {
		t.Fatal("unacquired became known null", got, err)
	}
	knownEmpty := normalizedEnvironmentModeState(t, 7, "empty", true)
	knownEmpty.Patterns = []EnvironmentPatternState{}
	if _, err := NewEnvironmentObservation([]EnvironmentState{knownEmpty}); err != nil {
		t.Fatal("acquired empty patterns rejected", err)
	}
}

func TestEnvironmentDeploymentObservationRejectsForgedOrIncompleteEvidence(t *testing.T) {
	for _, mutate := range []func(*EnvironmentState){
		func(s *EnvironmentState) { s.ID = 0 }, func(s *EnvironmentState) { s.Policy.Name = "" }, func(s *EnvironmentState) { s.Policy.WaitTimer = nil }, func(s *EnvironmentState) { s.Policy.DeploymentBranchPolicyDeclared = false }, func(s *EnvironmentState) {
			s.Policy.DeploymentBranchPolicy = &EnvironmentBranchPolicy{ProtectedBranches: true}
		},
		func(s *EnvironmentState) { s.DeploymentModeNormalization.Path = "other" }, func(s *EnvironmentState) { s.DeploymentModeNormalization.Fallback = uimodel.SelectedBranchesAndTags }, func(s *EnvironmentState) { s.DeploymentModeNormalization.RawPolicy.CustomBranchPolicies = false },
		func(s *EnvironmentState) { s.Patterns = nil }, func(s *EnvironmentState) { s.Patterns[0].ID = 0 }, func(s *EnvironmentState) { s.Patterns[1].ID = s.Patterns[0].ID }, func(s *EnvironmentState) { s.Patterns[1].Pattern = s.Patterns[0].Pattern }, func(s *EnvironmentState) { s.Patterns[0].Pattern.Type = "future" },
		func(s *EnvironmentState) {
			patterns := []EnvironmentPattern{{Type: "branch", Name: "release/*"}}
			s.Policy.DeploymentBranchPatterns = &patterns
		},
	} {
		state := normalizedEnvironmentModeState(t, 7, "production", true)
		mutate(&state)
		if got, err := state.RawDeploymentBranchPolicy(); err == nil || got != nil {
			t.Fatal("forged raw policy escaped", got, err)
		}
		observed, err := NewEnvironmentObservation([]EnvironmentState{{ID: 9, Policy: environmentPolicy(t, "valid")}, state})
		if err == nil || !reflect.DeepEqual(observed, EnvironmentObservation{}) {
			t.Fatal("partial collection escaped", observed, err)
		}
	}
}
