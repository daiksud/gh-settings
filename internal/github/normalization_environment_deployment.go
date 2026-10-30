package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// decodeObservedEnvironment projects complete detail policy only. The separate
// record must not be attached to a final observation before latent pattern
// acquisition and custom-protection checks complete.
func decodeObservedEnvironment(raw map[string]json.RawMessage) (settings.EnvironmentState, settings.EnvironmentDeploymentNormalization, error) {
	branch, present := raw["deployment_branch_policy"]
	if !present {
		return settings.EnvironmentState{}, settings.EnvironmentDeploymentNormalization{}, fmt.Errorf("environment response missing deployment_branch_policy")
	}
	projected := make(map[string]json.RawMessage, len(raw))
	for key, value := range raw {
		projected[key] = value
	}
	record := settings.EnvironmentDeploymentNormalization{}
	if !bytes.Equal(bytes.TrimSpace(branch), []byte("null")) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(branch, &fields); err != nil {
			return settings.EnvironmentState{}, settings.EnvironmentDeploymentNormalization{}, err
		}
		if err := environmentKnownFields(fields, "protected_branches", "custom_branch_policies"); err != nil {
			return settings.EnvironmentState{}, settings.EnvironmentDeploymentNormalization{}, err
		}
		var policy settings.EnvironmentBranchPolicy
		if err := requiredEnvironmentJSON(fields, "protected_branches", &policy.ProtectedBranches); err != nil {
			return settings.EnvironmentState{}, settings.EnvironmentDeploymentNormalization{}, err
		}
		if err := requiredEnvironmentJSON(fields, "custom_branch_policies", &policy.CustomBranchPolicies); err != nil {
			return settings.EnvironmentState{}, settings.EnvironmentDeploymentNormalization{}, err
		}
		if _, supported := settings.EnvironmentDeploymentChoiceForPolicy(&policy); !supported {
			projected["deployment_branch_policy"] = json.RawMessage("null")
			record = settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: policy, Fallback: settings.DefaultEnvironmentDeploymentChoice}
		}
	}
	state, err := decodeEnvironment(projected)
	if err != nil {
		return settings.EnvironmentState{}, settings.EnvironmentDeploymentNormalization{}, err
	}
	return state, record, nil
}
