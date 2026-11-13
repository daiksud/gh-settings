package github

import (
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
