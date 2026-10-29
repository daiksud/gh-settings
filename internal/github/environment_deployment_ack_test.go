package github

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentUpdateAcknowledgmentKeepsModeOwnershipSeparateFromPatternAcquisition(t *testing.T) {
	for _, rawPolicy := range []string{`null`, `{"protected_branches":true,"custom_branch_policies":false}`, `{"protected_branches":false,"custom_branch_policies":true}`, `{"protected_branches":false,"custom_branch_policies":false}`, `{"protected_branches":true,"custom_branch_policies":true}`} {
		for _, requestedPolicy := range []*settings.EnvironmentBranchPolicy{nil, {ProtectedBranches: true}, {CustomBranchPolicies: true}} {
			for _, declared := range []bool{false, true} {
				requested := settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPolicy: requestedPolicy, DeploymentBranchPolicyDeclared: declared}
				// Pattern intent belongs to later CRUD and independent fresh read, not this acknowledgment.
				patterns := []settings.EnvironmentPattern{{Type: "branch", Name: "release/*"}}
				if requestedPolicy != nil && requestedPolicy.CustomBranchPolicies || requestedPolicy == nil && !declared {
					requested.DeploymentBranchPatterns = &patterns
				}
				timer := 123
				requested.WaitTimer = &timer // The acknowledgment still has timer zero: final verification owns value confirmation.
				var raw map[string]json.RawMessage
				json.Unmarshal([]byte(emptyEnvironmentFixture), &raw)
				raw["deployment_branch_policy"] = json.RawMessage(rawPolicy)
				original := map[string]json.RawMessage{}
				for k, v := range raw {
					original[k] = bytes.Clone(v)
				}
				owned := declared || requestedPolicy != nil
				canonicalMatch := rawPolicy == "null" && requestedPolicy == nil || rawPolicy == `{"protected_branches":true,"custom_branch_policies":false}` && requestedPolicy != nil && requestedPolicy.ProtectedBranches || rawPolicy == `{"protected_branches":false,"custom_branch_policies":true}` && requestedPolicy != nil && requestedPolicy.CustomBranchPolicies
				err := verifyEnvironmentUpdateResponse(raw, requested)
				if (err != nil) != (owned && !canonicalMatch) || !reflect.DeepEqual(raw, original) {
					t.Fatal("ack certifies projection, patterns or omitted mode", rawPolicy, requestedPolicy, declared, err)
				}
			}
		}
	}
}

func TestEnvironmentUpdateAcknowledgmentRejectsIncompleteAndWrongTypedPolicy(t *testing.T) {
	for _, mutate := range []func(map[string]json.RawMessage){
		func(r map[string]json.RawMessage) { delete(r, "deployment_branch_policy") },
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":false}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":false,"custom_branch_policies":null}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":"false","custom_branch_policies":false}`)
		},
		func(r map[string]json.RawMessage) {
			r["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":false,"custom_branch_policies":false,"future":true}`)
		},
		func(r map[string]json.RawMessage) { delete(r, "can_admins_bypass") }, func(r map[string]json.RawMessage) { r["protection_rules"] = json.RawMessage(`null`) }, func(r map[string]json.RawMessage) { r["id"] = json.RawMessage(`0`) },
	} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(emptyEnvironmentFixture), &raw)
		raw["deployment_branch_policy"] = json.RawMessage(`{"protected_branches":true,"custom_branch_policies":true}`)
		mutate(raw)
		original := map[string]json.RawMessage{}
		for k, v := range raw {
			original[k] = bytes.Clone(v)
		}
		if err := verifyEnvironmentUpdateResponse(raw, settings.EnvironmentDeclaration{Name: "production"}); err == nil || !reflect.DeepEqual(raw, original) {
			t.Fatal("malformed acknowledgment supplied omitted values", raw, err)
		}
	}
}
