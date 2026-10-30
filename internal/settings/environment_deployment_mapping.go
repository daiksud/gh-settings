package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const EnvironmentDeploymentModePath = "repository.settings.environments.deployment_branches_and_tags"
const DefaultEnvironmentDeploymentChoice = uimodel.DefaultEnvironmentDeploymentChoice

func EnvironmentDeploymentPolicyForChoice(choice uimodel.EnvironmentDeploymentChoice) (*EnvironmentBranchPolicy, bool) {
	switch choice {
	case uimodel.NoRestriction:
		return nil, true
	case uimodel.ProtectedBranchesOnly:
		return &EnvironmentBranchPolicy{ProtectedBranches: true}, true
	case uimodel.SelectedBranchesAndTags:
		return &EnvironmentBranchPolicy{CustomBranchPolicies: true}, true
	default:
		return nil, false
	}
}

func EnvironmentDeploymentChoiceForPolicy(policy *EnvironmentBranchPolicy) (uimodel.EnvironmentDeploymentChoice, bool) {
	if policy == nil {
		return uimodel.NoRestriction, true
	}
	switch {
	case policy.ProtectedBranches && !policy.CustomBranchPolicies:
		return uimodel.ProtectedBranchesOnly, true
	case !policy.ProtectedBranches && policy.CustomBranchPolicies:
		return uimodel.SelectedBranchesAndTags, true
	default:
		return "", false
	}
}
