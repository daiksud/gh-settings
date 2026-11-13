package settings

import (
	"fmt"
	"reflect"
)

// EnvironmentBuiltinsMatchRequest compares only declared built-in controls,
// keeping raw mode equality separate from its effective UI projection.
func EnvironmentBuiltinsMatchRequest(state EnvironmentState, requested EnvironmentDeclaration) (bool, error) {
	raw, err := state.RawDeploymentBranchPolicy()
	if err != nil {
		return false, err
	}
	requested = cloneEnvironment(requested)
	requested.DeploymentBranchPatterns = nil
	requested.CustomRules = nil
	if requested.DeploymentBranchPolicyDeclared && !reflect.DeepEqual(raw, requested.DeploymentBranchPolicy) {
		return false, nil
	}
	return EnvironmentMatchesRequest(state.Policy, requested), nil
}

// PrepareEnvironmentUpdate copies complete supported built-ins from the latest
// observation and overlays original intent. It is send-time state, never a new
// declaration or a source of pattern ownership or normalization warnings.
func PrepareEnvironmentUpdate(state EnvironmentState, requested EnvironmentDeclaration) (EnvironmentDeclaration, error) {
	raw, err := state.RawDeploymentBranchPolicy()
	if err != nil {
		return EnvironmentDeclaration{}, err
	}
	if EnvironmentIdentity(state.Policy.Name) != EnvironmentIdentity(requested.Name) {
		return EnvironmentDeclaration{}, fmt.Errorf("environment preservation identity differs from requested name")
	}
	if requested.DeploymentBranchPolicyDeclared && raw != nil && raw.CustomBranchPolicies && (requested.DeploymentBranchPolicy == nil || !requested.DeploymentBranchPolicy.CustomBranchPolicies) {
		if state.Patterns == nil {
			return EnvironmentDeclaration{}, fmt.Errorf("environment %q requires complete deployment pattern acquisition before changing mode", requested.Name)
		}
		if err := ValidateEnvironmentPatternObservation(state.Patterns); err != nil {
			return EnvironmentDeclaration{}, err
		}
		if requested.DeploymentBranchPatterns == nil && len(state.Patterns) > 0 {
			return EnvironmentDeclaration{}, fmt.Errorf("environment %q cannot change deployment mode without deleting omitted deployment patterns; first apply deployment_branch_or_tag_rules: [] while Selected, then change mode", requested.Name)
		}
	}
	requested = cloneEnvironment(requested)
	expected := cloneEnvironment(state.Policy)
	expected.DeploymentBranchPatterns = nil
	if requested.WaitTimer != nil {
		expected.WaitTimer = requested.WaitTimer
	}
	if requested.PreventSelfReview != nil {
		expected.PreventSelfReview = requested.PreventSelfReview
	}
	if requested.CanAdminsBypass != nil {
		expected.CanAdminsBypass = requested.CanAdminsBypass
	}
	if requested.Reviewers != nil {
		expected.Reviewers = requested.Reviewers
	}
	if requested.DeploymentBranchPolicyDeclared {
		expected.DeploymentBranchPolicy = requested.DeploymentBranchPolicy
	} else {
		if _, supported := EnvironmentDeploymentChoiceForPolicy(raw); !supported {
			return EnvironmentDeclaration{}, fmt.Errorf("environment %q cannot preserve omitted raw deployment mode in a policy update; declare a supported mode explicitly", requested.Name)
		}
		expected.DeploymentBranchPolicy = raw
	}
	expected.DeploymentBranchPolicyDeclared = true
	if err := validateEnvironment(expected, true); err != nil {
		return EnvironmentDeclaration{}, fmt.Errorf("environment %q requested controls conflict with preserved companions: %w", requested.Name, err)
	}
	return expected, nil
}
