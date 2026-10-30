package github

import (
	"encoding/json"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func observedEnvironmentRefTypeState(t *testing.T, rawType string, collision bool) settings.EnvironmentState {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(emptyEnvironmentFixture), &raw); err != nil {
		t.Fatal(err)
	}
	raw["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":false,"custom_branch_policies":true}`)
	state, err := decodeEnvironment(raw)
	if err != nil {
		t.Fatal(err)
	}
	state.Patterns = []settings.EnvironmentPatternState{{ID: 7, Pattern: settings.EnvironmentPattern{Type: rawType, Name: "main"}}}
	if rawType != "branch" && rawType != "tag" {
		state.Patterns[0].Pattern.Type = "branch"
		state.Patterns[0].RefTypeNormalization = settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: rawType, Fallback: "branch"}
	}
	if collision {
		state.Patterns = append(state.Patterns, settings.EnvironmentPatternState{ID: 8, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}})
	}
	if !collision {
		patterns, err := settings.EffectiveEnvironmentPatterns(state.Patterns)
		if err != nil {
			t.Fatal(err)
		}
		state.Policy.DeploymentBranchPatterns = &patterns
	}
	observed, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{state})
	if err != nil {
		t.Fatal(err)
	}
	values, err := observed.Values()
	if err != nil {
		t.Fatal(err)
	}
	return values[0]
}

func TestEnvironmentRequestedPatternVerificationUsesRawAcquiredIdentity(t *testing.T) {
	for _, raw := range []string{"branch", "tag", "future", "", " \t"} {
		state := observedEnvironmentRefTypeState(t, raw, false)
		for _, desired := range [][]settings.EnvironmentPattern{{{Type: "branch", Name: "main"}}, {{Type: "tag", Name: "main"}}, {}} {
			requested := settings.EnvironmentDeclaration{Name: "PRODUCTION", DeploymentBranchPatterns: &desired}
			err := verifyEnvironmentRequestedState(state, requested)
			wantSuccess := len(desired) == 1 && desired[0].Type == raw
			if (err == nil) != wantSuccess {
				t.Fatal("fallback projection confirmed requested raw type", raw, desired, err)
			}
		}
		if err := verifyEnvironmentRequestedState(state, settings.EnvironmentDeclaration{Name: "PRODUCTION"}); err != nil {
			t.Fatal("unowned pattern outcome constrained", raw, err)
		}
	}
	collision := observedEnvironmentRefTypeState(t, "future", true)
	desired := []settings.EnvironmentPattern{{Type: "branch", Name: "main"}}
	if err := verifyEnvironmentRequestedState(collision, settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPatterns: &desired}); err == nil {
		t.Fatal("canonical companion hid obsolete unknown raw identity")
	}
	if err := verifyEnvironmentRequestedState(collision, settings.EnvironmentDeclaration{Name: "production"}); err != nil {
		t.Fatal("unowned collision blocked requested fields", err)
	}
}

func TestEnvironmentRequestedPatternVerificationRequiresAcquisitionAndActiveApplicability(t *testing.T) {
	empty := []settings.EnvironmentPattern{}
	requested := settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPatterns: &empty}
	canonical := observedEnvironmentRefTypeState(t, "branch", false)
	canonical.Patterns, canonical.Policy.DeploymentBranchPatterns = []settings.EnvironmentPatternState{}, &empty
	if err := verifyEnvironmentRequestedState(canonical, requested); err != nil {
		t.Fatal("known active empty rejected", err)
	}
	for _, mutate := range []func(*settings.EnvironmentState){
		func(s *settings.EnvironmentState) { s.Patterns = nil },
		func(s *settings.EnvironmentState) {
			s.Policy.DeploymentBranchPolicy = nil
			s.Policy.DeploymentBranchPatterns = nil
		},
		func(s *settings.EnvironmentState) {
			s.Policy.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{ProtectedBranches: true}
			s.Policy.DeploymentBranchPatterns = nil
		},
		func(s *settings.EnvironmentState) {
			s.Patterns = []settings.EnvironmentPatternState{{ID: 0, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}}}
		},
	} {
		state := canonical
		mutate(&state)
		if err := verifyEnvironmentRequestedState(state, requested); err == nil {
			t.Fatal("missing acquisition or inactive mode confirmed empty intent", state)
		}
	}
	latent := observedEnvironmentModeState(t, true)
	if err := verifyEnvironmentRequestedState(latent, requested); err == nil {
		t.Fatal("known empty latent TT proved active Selected applicability")
	}
	invalid := observedEnvironmentRefTypeState(t, "future", false)
	invalid.Patterns[0].RefTypeNormalization.Path = "wrong"
	if err := verifyEnvironmentRequestedState(invalid, settings.EnvironmentDeclaration{Name: "production"}); err == nil {
		t.Fatal("malformed unowned acquisition ignored")
	}
}
