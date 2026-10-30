package github

import (
	"encoding/json"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func observedEnvironmentModeState(t *testing.T, raw bool) settings.EnvironmentState {
	t.Helper()
	var response map[string]json.RawMessage
	json.Unmarshal([]byte(emptyEnvironmentFixture), &response)
	response["deployment_branch_policy"], _ = json.Marshal(settings.EnvironmentBranchPolicy{ProtectedBranches: raw, CustomBranchPolicies: raw})
	state, record, err := decodeObservedEnvironment(response)
	if err != nil {
		t.Fatal(err)
	}
	state.DeploymentModeNormalization = record
	if raw {
		state.Patterns = []settings.EnvironmentPatternState{}
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

func TestEnvironmentRequestedStateVerificationUsesRawExplicitModeAndIgnoresOmittedMode(t *testing.T) {
	var response map[string]json.RawMessage
	json.Unmarshal([]byte(emptyEnvironmentFixture), &response)
	canonical, err := decodeEnvironment(response)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []settings.EnvironmentState{canonical, observedEnvironmentModeState(t, false), observedEnvironmentModeState(t, true)} {
		requested := state.Policy
		err := verifyEnvironmentRequestedState(state, requested)
		if (err != nil) != (state.DeploymentModeNormalization != (settings.EnvironmentDeploymentNormalization{})) {
			t.Fatal("projected null confirmed raw mode", state, err)
		}
		modeOmitted := settings.EnvironmentDeclaration{Name: "PRODUCTION", WaitTimer: state.Policy.WaitTimer}
		if err := verifyEnvironmentRequestedState(state, modeOmitted); err != nil {
			t.Fatal("omitted mode gained preservation requirement", state, err)
		}
		// Existing nullable choice may change under an omitted PUT. Only explicit fields are verified.
		protected := canonical
		protected.Policy.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{ProtectedBranches: true}
		if err := verifyEnvironmentRequestedState(protected, modeOmitted); err != nil {
			t.Fatal("server-selected mode outcome was certified or constrained", err)
		}
		timer := 5
		modeOmitted.WaitTimer = &timer
		if err := verifyEnvironmentRequestedState(state, modeOmitted); err == nil {
			t.Fatal("omitted mode hid different requested timer")
		}
	}
	for _, policy := range []*settings.EnvironmentBranchPolicy{{ProtectedBranches: true}, {CustomBranchPolicies: true}} {
		actual := canonical
		actual.Policy.DeploymentBranchPolicy = policy
		requested := actual.Policy
		if err := verifyEnvironmentRequestedState(actual, requested); err != nil {
			t.Fatal("canonical requested mode failed", err)
		}
		requested.DeploymentBranchPolicyDeclared = false
		if err := verifyEnvironmentRequestedState(actual, requested); err != nil {
			t.Fatal("nonnull requested policy lost explicit ownership", err)
		}
		for _, mismatch := range []settings.EnvironmentState{canonical, observedEnvironmentModeState(t, false), observedEnvironmentModeState(t, true)} {
			if err := verifyEnvironmentRequestedState(mismatch, requested); err == nil {
				t.Fatal("nonnull policy with false declaration flag accepted a different raw mode", mismatch)
			}
		}
		requested.DeploymentBranchPolicy = nil
		requested.DeploymentBranchPolicyDeclared = true
		if err := verifyEnvironmentRequestedState(actual, requested); err == nil {
			t.Fatal("known different mode confirmed null")
		}
	}
	selected := canonical
	selected.Policy.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
	patterns := []settings.EnvironmentPattern{{Type: "branch", Name: "release/*"}}
	selected.Policy.DeploymentBranchPatterns = &patterns
	selected.Patterns = []settings.EnvironmentPatternState{{ID: 7, Pattern: patterns[0]}}
	requested := settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPatterns: &patterns}
	if err := verifyEnvironmentRequestedState(selected, requested); err != nil {
		t.Fatal("scoped existing pattern verification changed", err)
	}
	different := []settings.EnvironmentPattern{{Type: "branch", Name: "other/*"}}
	requested.DeploymentBranchPatterns = &different
	if err := verifyEnvironmentRequestedState(selected, requested); err == nil {
		t.Fatal("different requested patterns confirmed")
	}
}

func TestEnvironmentRequestedStateVerificationRejectsIncompleteAndForgedAcquisition(t *testing.T) {
	for _, mutate := range []func(*settings.EnvironmentState){
		func(s *settings.EnvironmentState) { s.ID = 0 }, func(s *settings.EnvironmentState) { s.Policy.WaitTimer = nil }, func(s *settings.EnvironmentState) { s.Policy.DeploymentBranchPolicyDeclared = false }, func(s *settings.EnvironmentState) { s.DeploymentModeNormalization.Path = "other" }, func(s *settings.EnvironmentState) { s.Patterns = nil },
	} {
		actual := observedEnvironmentModeState(t, true)
		mutate(&actual)
		if err := verifyEnvironmentRequestedState(actual, settings.EnvironmentDeclaration{Name: "production"}); err == nil {
			t.Fatal("invalid acquisition became API-controlled success", actual)
		}
	}
	if err := verifyEnvironmentRequestedState(settings.EnvironmentState{}, settings.EnvironmentDeclaration{Name: "production"}); err == nil {
		t.Fatal("unacquired became omission success")
	}
}
