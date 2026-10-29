package settings

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// EnvironmentDeploymentNormalization owns an acquired, out-of-UI boolean pair.
// It never supplies omitted desired mode or active pattern intent.
type EnvironmentDeploymentNormalization struct {
	Path      string                              `json:"path"`
	RawPolicy EnvironmentBranchPolicy             `json:"raw_policy"`
	Fallback  uimodel.EnvironmentDeploymentChoice `json:"fallback"`
}

func (state EnvironmentState) validateDeploymentNormalization() error {
	if err := validateEnvironment(state.Policy, true); err != nil {
		return err
	}
	if state.ID <= 0 {
		return fmt.Errorf("environment observation requires a positive ID")
	}
	record := state.DeploymentModeNormalization
	if record == (EnvironmentDeploymentNormalization{}) {
		return nil
	}
	_, supported := EnvironmentDeploymentChoiceForPolicy(&record.RawPolicy)
	if record.Path != EnvironmentDeploymentModePath || record.Fallback != DefaultEnvironmentDeploymentChoice || supported || state.Policy.DeploymentBranchPolicy != nil || !state.Policy.DeploymentBranchPolicyDeclared || state.Policy.DeploymentBranchPatterns != nil {
		return fmt.Errorf("invalid environment deployment mode observation provenance")
	}
	if record.RawPolicy.CustomBranchPolicies {
		if state.Patterns == nil {
			return fmt.Errorf("normalized raw custom mode requires acquired deployment patterns")
		}
		empty := []EnvironmentPattern{}
		if _, err := ReconcileEnvironmentPatterns(state.Patterns, &empty); err != nil {
			return err
		}
	}
	return nil
}

// RawDeploymentBranchPolicy distinguishes acquired null from an acquired pair.
// It returns an owned pointer after complete validation of the scoped evidence.
func (state EnvironmentState) RawDeploymentBranchPolicy() (*EnvironmentBranchPolicy, error) {
	if err := state.validateDeploymentNormalization(); err != nil {
		return nil, err
	}
	if record := state.DeploymentModeNormalization; record != (EnvironmentDeploymentNormalization{}) {
		raw := record.RawPolicy
		return &raw, nil
	}
	return copyRuleValue(state.Policy.DeploymentBranchPolicy), nil
}
