package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestEnvironmentDeploymentMappingPreservesThreeClosedChoicesAndOwnsPolicies(t *testing.T) {
	if EnvironmentDeploymentModePath != "repository.settings.environments.deployment_branches_and_tags" || DefaultEnvironmentDeploymentChoice != uimodel.EnvironmentDeploymentChoice("no_restriction") {
		t.Fatal("wrong mode path/default")
	}
	for _, test := range []struct {
		choice uimodel.EnvironmentDeploymentChoice
		policy *EnvironmentBranchPolicy
	}{
		{"no_restriction", nil}, {"protected_branches_only", &EnvironmentBranchPolicy{ProtectedBranches: true}}, {"selected_branches_and_tags", &EnvironmentBranchPolicy{CustomBranchPolicies: true}},
	} {
		policy, ok := EnvironmentDeploymentPolicyForChoice(test.choice)
		if !ok || !reflect.DeepEqual(policy, test.policy) {
			t.Fatal("forward mapping", test, policy, ok)
		}
		choice, ok := EnvironmentDeploymentChoiceForPolicy(test.policy)
		if !ok || choice != test.choice {
			t.Fatal("reverse mapping", test, choice, ok)
		}
		if policy != nil {
			*policy = EnvironmentBranchPolicy{ProtectedBranches: true, CustomBranchPolicies: true}
			fresh, ok := EnvironmentDeploymentPolicyForChoice(test.choice)
			if !ok || !reflect.DeepEqual(fresh, test.policy) {
				t.Fatal("returned policy alias", fresh, ok)
			}
		}
	}
	for _, policy := range []*EnvironmentBranchPolicy{{}, {ProtectedBranches: true, CustomBranchPolicies: true}} {
		if choice, ok := EnvironmentDeploymentChoiceForPolicy(policy); ok || choice != "" {
			t.Fatal("equal pair gained UI choice", policy, choice, ok)
		}
	}
	for _, choice := range []uimodel.EnvironmentDeploymentChoice{"", "future", "branch", "No restriction"} {
		if policy, ok := EnvironmentDeploymentPolicyForChoice(choice); ok || policy != nil {
			t.Fatal("unknown desired choice accepted", choice, policy, ok)
		}
	}
}
