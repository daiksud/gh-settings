package github

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentDeploymentWarningsRequireOwnedCompleteSelectedState(t *testing.T) {
	type event struct {
		name   string
		record settings.EnvironmentDeploymentNormalization
	}
	for _, raw := range []bool{false, true} {
		for _, request := range []settings.EnvironmentDeclaration{
			{Name: "production", DeploymentBranchPolicyDeclared: true},
			{Name: "PRODUCTION", DeploymentBranchPolicy: &settings.EnvironmentBranchPolicy{ProtectedBranches: true}},
			{Name: "production", DeploymentBranchPolicy: &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}},
			{Name: "production"}, {Name: "other", DeploymentBranchPolicyDeclared: true},
		} {
			for _, sink := range []bool{false, true} {
				state := observedEnvironmentModeState(t, raw)
				var got []event
				client := &EnvironmentsClient{}
				if sink {
					client.NormalizationWarning = func(name string, record settings.EnvironmentDeploymentNormalization) {
						got = append(got, event{name, record})
					}
				}
				client.warnDeploymentNormalization(state, request)
				var want []event
				if sink && settings.EnvironmentIdentity(request.Name) == settings.EnvironmentIdentity(state.Policy.Name) && (request.DeploymentBranchPolicyDeclared || request.DeploymentBranchPolicy != nil) {
					want = []event{{"production", state.DeploymentModeNormalization}}
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatal("warning scope/typed record/context", raw, request, sink, got, want)
				}
			}
		}
	}
}

func TestEnvironmentDeploymentWarningsSuppressCanonicalAndIncompleteEvidence(t *testing.T) {
	for _, mutate := range []func(*settings.EnvironmentState){
		func(s *settings.EnvironmentState) {
			s.DeploymentModeNormalization = settings.EnvironmentDeploymentNormalization{}
		},
		func(s *settings.EnvironmentState) { s.ID = 0 }, func(s *settings.EnvironmentState) { s.Policy.Name = "" }, func(s *settings.EnvironmentState) { s.Policy.WaitTimer = nil }, func(s *settings.EnvironmentState) { s.Policy.DeploymentBranchPolicyDeclared = false },
		func(s *settings.EnvironmentState) { s.DeploymentModeNormalization.Path = "other" }, func(s *settings.EnvironmentState) { s.Patterns = nil }, func(s *settings.EnvironmentState) {
			s.Patterns = []settings.EnvironmentPatternState{{ID: 0, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "release/*"}}}
		},
	} {
		state := observedEnvironmentModeState(t, true)
		mutate(&state)
		client := &EnvironmentsClient{NormalizationWarning: func(string, settings.EnvironmentDeploymentNormalization) { t.Fatal("invalid/unacquired state warned") }}
		client.warnDeploymentNormalization(state, settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPolicyDeclared: true})
	}
	var response map[string]json.RawMessage
	json.Unmarshal([]byte(emptyEnvironmentFixture), &response)
	canonical, err := decodeEnvironment(response)
	if err != nil {
		t.Fatal(err)
	}
	client := &EnvironmentsClient{NormalizationWarning: func(string, settings.EnvironmentDeploymentNormalization) { t.Fatal("canonical mode warned") }}
	client.warnDeploymentNormalization(canonical, settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPolicyDeclared: true})
	client.warnDeploymentNormalization(settings.EnvironmentState{}, settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPolicyDeclared: true})
}
