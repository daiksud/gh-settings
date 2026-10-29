package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func DeclarationFromUI(repository uimodel.Repository) (settings.Declaration, error) {
	var result settings.Declaration
	if repository.About != nil {
		about, err := AboutDeclaration(*repository.About)
		if err != nil {
			return settings.Declaration{}, fmt.Errorf("repository.about: %w", err)
		}
		result = about
	}
	if repository.Settings != nil && repository.Settings.General != nil {
		general, err := GeneralDeclaration(*repository.Settings.General)
		if err != nil {
			return settings.Declaration{}, fmt.Errorf("repository.settings: %w", err)
		}
		general.Description = result.Repository.Description
		general.Homepage = result.Repository.Homepage
		result.Repository = general
	}
	labels, err := LabelsDeclaration(repository.Labels)
	if err != nil {
		return settings.Declaration{}, fmt.Errorf("repository.labels: %w", err)
	}
	result.Labels = labels
	if repository.Settings != nil {
		securityUI := uimodel.SecuritySettings{}
		if repository.Settings.General != nil {
			securityUI.GeneralReleases = repository.Settings.General.Releases
		}
		if repository.Settings.SecurityAndQuality != nil {
			securityUI.AdvancedSecurity = repository.Settings.SecurityAndQuality.AdvancedSecurity
		}
		security, err := SecurityDeclaration(securityUI)
		if err != nil {
			return settings.Declaration{}, fmt.Errorf("repository.settings: %w", err)
		}
		result.Security = security
		rulesets, err := RulesetsDeclaration(repository.Settings.Rulesets)
		if err != nil {
			return settings.Declaration{}, fmt.Errorf("repository.settings.rulesets: %w", err)
		}
		result.Rulesets = rulesets
		if repository.Settings.Actions != nil && repository.Settings.Actions.General != nil {
			actions, err := ActionsDeclaration(*repository.Settings.Actions.General)
			if err != nil {
				return settings.Declaration{}, fmt.Errorf("repository.settings.actions.general: %w", err)
			}
			result.Actions = actions
		}
	}
	return result, nil
}
