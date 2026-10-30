package github

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestObservedEnvironmentDetailSeparatesModeEvidenceFromPatternAcquisition(t *testing.T) {
	for _, test := range []struct {
		raw    string
		policy *settings.EnvironmentBranchPolicy
		record settings.EnvironmentDeploymentNormalization
	}{
		{`null`, nil, settings.EnvironmentDeploymentNormalization{}},
		{`{"protected_branches":true,"custom_branch_policies":false}`, &settings.EnvironmentBranchPolicy{ProtectedBranches: true}, settings.EnvironmentDeploymentNormalization{}},
		{`{"protected_branches":false,"custom_branch_policies":true}`, &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}, settings.EnvironmentDeploymentNormalization{}},
		{`{"protected_branches":false,"custom_branch_policies":false}`, nil, settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{}, Fallback: uimodel.NoRestriction}},
		{`{"protected_branches":true,"custom_branch_policies":true}`, nil, settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{ProtectedBranches: true, CustomBranchPolicies: true}, Fallback: uimodel.NoRestriction}},
	} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(emptyEnvironmentFixture), &raw)
		raw["deployment_branch_policy"] = json.RawMessage(test.raw)
		original := map[string]json.RawMessage{}
		for k, v := range raw {
			original[k] = bytes.Clone(v)
		}
		state, record, err := decodeObservedEnvironment(raw)
		if err != nil || record != test.record || state.ID != 1 || state.Policy.Name != "production" || !state.Policy.DeploymentBranchPolicyDeclared || !reflect.DeepEqual(state.Policy.DeploymentBranchPolicy, test.policy) || state.Policy.DeploymentBranchPatterns != nil || state.Patterns != nil || state.DeploymentModeNormalization != (settings.EnvironmentDeploymentNormalization{}) || !reflect.DeepEqual(raw, original) {
			t.Fatal("detail proof/ownership boundary", state, record, err)
		}
		// The detail helper returns separate evidence; it has not read latent pattern pages.
		if record.RawPolicy.CustomBranchPolicies && record != (settings.EnvironmentDeploymentNormalization{}) {
			state.DeploymentModeNormalization = record
			if _, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{state}); err == nil {
				t.Fatal("preliminary detail asserted complete latent acquisition")
			}
		}
		_, strictErr := decodeEnvironment(raw)
		if (strictErr != nil) != (record != (settings.EnvironmentDeploymentNormalization{})) {
			t.Fatal("strict native decoder changed", strictErr)
		}
		if state.Policy.DeploymentBranchPolicy != nil {
			state.Policy.DeploymentBranchPolicy.ProtectedBranches = !state.Policy.DeploymentBranchPolicy.ProtectedBranches
		}
		if !reflect.DeepEqual(raw, original) {
			t.Fatal("decoded state aliases raw bytes")
		}
	}
}

func TestObservedEnvironmentDetailRejectsMalformedWithoutPartialEvidence(t *testing.T) {
	for _, mutate := range []func(map[string]json.RawMessage){
		func(r map[string]json.RawMessage) { delete(r, "deployment_branch_policy") },
		func(r map[string]json.RawMessage) { r["deployment_branch_policy"] = json.RawMessage(`{}`) },
		func(r map[string]json.RawMessage) { r["deployment_branch_policy"] = json.RawMessage(`false`) },
		func(r map[string]json.RawMessage) { r["deployment_branch_policy"] = json.RawMessage(`[]`) },
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":false}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"custom_branch_policies":true}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":null,"custom_branch_policies":false}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":false,"custom_branch_policies":null}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":false,"custom_branch_policies":"false"}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":0,"custom_branch_policies":false}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":false,"custom_branch_policies":false,"future":true}`)
		},
		func(r map[string]json.RawMessage) { r["future"] = json.RawMessage(`true`) },
		func(r map[string]json.RawMessage) { delete(r, "can_admins_bypass") },
		func(r map[string]json.RawMessage) { r["protection_rules"] = json.RawMessage(`null`) },
		func(r map[string]json.RawMessage) { r["id"] = json.RawMessage(`0`) },
		func(r map[string]json.RawMessage) { r["name"] = json.RawMessage(`null`) },
	} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(emptyEnvironmentFixture), &raw)
		raw["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":true,"custom_branch_policies":true}`)
		mutate(raw)
		original := map[string]json.RawMessage{}
		for k, v := range raw {
			original[k] = bytes.Clone(v)
		}
		state, record, err := decodeObservedEnvironment(raw)
		if err == nil || !reflect.DeepEqual(state, settings.EnvironmentState{}) || record != (settings.EnvironmentDeploymentNormalization{}) || !reflect.DeepEqual(raw, original) {
			t.Fatal("malformed detail returned partial proof", state, record, err)
		}
	}
}
