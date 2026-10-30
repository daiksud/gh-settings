package github

import (
	"encoding/json"
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

// verifyEnvironmentUpdateResponse validates preliminary detail data without
// inventing pattern acquisition or certifying API-controlled omitted outcomes.
func verifyEnvironmentUpdateResponse(raw map[string]json.RawMessage, requested settings.EnvironmentDeclaration) error {
	actual, record, err := decodeObservedEnvironment(raw)
	if err != nil {
		return err
	}
	if requested.DeploymentBranchPolicyDeclared || requested.DeploymentBranchPolicy != nil {
		rawMode := actual.Policy.DeploymentBranchPolicy
		if record != (settings.EnvironmentDeploymentNormalization{}) {
			rawMode = &record.RawPolicy
		}
		if !reflect.DeepEqual(rawMode, requested.DeploymentBranchPolicy) {
			return fmt.Errorf("environment acknowledgment raw deployment mode differs from requested policy")
		}
	}
	return nil
}
