package github

import (
	"fmt"
	"reflect"

	"github.com/daiksud/gh-settings/internal/settings"
)

// verifyEnvironmentRequestedState confirms requested values against a completed
// acquisition. Omitted mode outcomes remain controlled by the API.
func verifyEnvironmentRequestedState(actual settings.EnvironmentState, requested settings.EnvironmentDeclaration) error {
	rawMode, err := actual.RawDeploymentBranchPolicy()
	if err != nil {
		return err
	}
	if requested.DeploymentBranchPolicyDeclared || requested.DeploymentBranchPolicy != nil {
		if !reflect.DeepEqual(rawMode, requested.DeploymentBranchPolicy) {
			return fmt.Errorf("environment raw deployment mode differs from requested policy")
		}
	}
	if !settings.EnvironmentMatchesRequest(actual.Policy, requested) {
		return fmt.Errorf("environment readback differs from requested policy")
	}
	return nil
}
