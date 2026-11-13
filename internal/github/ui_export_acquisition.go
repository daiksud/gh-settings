package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// CheckExportAcquisition checks an isolated, acquired family before adoption.
// Unlike retained declarations, observations must contain all required values.
func CheckExportAcquisition(source settings.Declaration) error {
	if source.Topics.Names != nil {
		if err := settings.ValidateTopics(*source.Topics.Names); err != nil {
			return err
		}
	}
	if source.Labels != nil {
		for _, label := range source.Labels.Values() {
			if _, err := LabelPolicyUI(label); err != nil {
				return err
			}
		}
	}
	if _, err := ActionsExecutionPolicyUI(source.Actions); err != nil {
		return err
	}
	if _, err := WithActionsWorkflowPolicyUI(uimodel.ActionsGeneral{}, source.Actions); err != nil {
		return err
	}
	if !source.Pages.Empty() {
		if _, err := PagesPolicyUI(source.Pages); err != nil {
			return err
		}
	}
	if source.Autolinks != nil {
		for _, v := range *source.Autolinks {
			if v.ID == nil {
				return fmt.Errorf("autolink export lacks acquired identity")
			}
		}
	}
	if source.BranchProtectionRules != nil {
		for _, r := range *source.BranchProtectionRules {
			if !settings.ClassicForcePushSelected(r.AllowForcePushes) {
				return fmt.Errorf("classic export lacks acquired force-push control")
			}
			if !settings.ClassicReviewsManaged(r.RequireAPullRequestBeforeMerging) {
				return fmt.Errorf("classic export lacks acquired PR policy")
			}
			if !settings.ClassicDeploymentsSelected(r.RequireDeploymentsToSucceedBeforeMerging) {
				return fmt.Errorf("classic export lacks acquired deployment control")
			}
			if r.ID == nil {
				return fmt.Errorf("classic export lacks acquired identity")
			}
			for _, v := range settings.ClassicFlags(r) {
				if v == nil {
					return fmt.Errorf("classic export lacks acquired control")
				}
			}
		}
	}
	if source.Environments != nil {
		for _, policy := range source.Environments.Values() {
			if _, err := EnvironmentPolicyUI(policy); err != nil {
				return err
			}
		}
	}
	_, err := UIFromExportDeclaration(source)
	return err
}
