package settings

import (
	"bytes"
	"encoding/json"
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
	if err := validateEnvironmentDeploymentRecord(state.Policy, record); err != nil {
		return err
	}
	if record.RawPolicy.CustomBranchPolicies {
		if state.Patterns == nil {
			return fmt.Errorf("normalized raw custom mode requires acquired deployment patterns")
		}
		if err := ValidateEnvironmentPatternObservation(state.Patterns); err != nil {
			return err
		}
	}
	return state.validateRefTypeNormalizations()
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

// validateEnvironmentDeploymentRecord checks scalar presentation evidence only;
// it does not reconstruct pattern acquisition from a plan report.
func validateEnvironmentDeploymentRecord(policy EnvironmentDeclaration, record EnvironmentDeploymentNormalization) error {
	if record == (EnvironmentDeploymentNormalization{}) {
		return nil
	}
	_, supported := EnvironmentDeploymentChoiceForPolicy(&record.RawPolicy)
	if record.Path != EnvironmentDeploymentModePath || record.Fallback != DefaultEnvironmentDeploymentChoice || supported || policy.DeploymentBranchPolicy != nil || !policy.DeploymentBranchPolicyDeclared || policy.DeploymentBranchPatterns != nil {
		return fmt.Errorf("invalid environment deployment mode observation provenance")
	}
	return nil
}

func (change EnvironmentChange) DeploymentModeNormalization() EnvironmentDeploymentNormalization {
	return change.deploymentModeNormalization
}

func NewEnvironmentChangeWithDeploymentNormalization(operation string, before, after *EnvironmentDeclaration, record EnvironmentDeploymentNormalization) (EnvironmentChange, error) {
	if record == (EnvironmentDeploymentNormalization{}) {
		return EnvironmentChange{}, fmt.Errorf("normalized environment change requires provenance")
	}
	return newEnvironmentChange(operation, before, after, record, nil)
}

// UnmarshalJSON requires both acquired booleans, retaining false separately from
// omission or null, and assigns only after the complete structure is decoded.
func (record *EnvironmentDeploymentNormalization) UnmarshalJSON(data []byte) error {
	var fields struct {
		Path      *string `json:"path"`
		RawPolicy *struct {
			ProtectedBranches    *bool `json:"protected_branches"`
			CustomBranchPolicies *bool `json:"custom_branch_policies"`
		} `json:"raw_policy"`
		Fallback *uimodel.EnvironmentDeploymentChoice `json:"fallback"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("decode environment deployment normalization: %w", err)
	}
	if fields.Path == nil || fields.Fallback == nil || fields.RawPolicy == nil || fields.RawPolicy.ProtectedBranches == nil || fields.RawPolicy.CustomBranchPolicies == nil {
		return fmt.Errorf("environment deployment normalization requires string path/fallback and complete boolean raw_policy")
	}
	*record = EnvironmentDeploymentNormalization{Path: *fields.Path, RawPolicy: EnvironmentBranchPolicy{ProtectedBranches: *fields.RawPolicy.ProtectedBranches, CustomBranchPolicies: *fields.RawPolicy.CustomBranchPolicies}, Fallback: *fields.Fallback}
	return nil
}
