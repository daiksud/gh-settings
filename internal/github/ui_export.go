package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// UIFromExportDeclaration projects values already acquired and validated by export.
// It does not establish acquisition or fill missing observation metadata.
func UIFromExportDeclaration(source settings.Declaration) (uimodel.Repository, error) {
	if source.Repository.Description == nil || source.Repository.Homepage == nil || source.Topics.Names == nil {
		return uimodel.Repository{}, fmt.Errorf("export requires acquired About description, website and topics")
	}
	topics := settings.SortedTopics(*source.Topics.Names)
	result := uimodel.Repository{About: &uimodel.About{Description: copyUIValue(source.Repository.Description), Website: copyUIValue(source.Repository.Homepage), Topics: &topics}}
	general, err := GeneralObservation(source.Repository)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export General: %w", err)
	}
	if general.General != (uimodel.General{}) {
		result.Settings = &uimodel.RepositorySettings{General: &general.General}
	}
	if source.Labels != nil {
		labels := make([]uimodel.Label, 0, source.Labels.Len())
		for _, policy := range settings.SortedLabels(source.Labels.Values()) {
			label, err := LabelPolicyUI(policy)
			if err != nil {
				return uimodel.Repository{}, fmt.Errorf("export Labels: %w", err)
			}
			labels = append(labels, label)
		}
		result.Labels = &labels
	}
	if source.Rulesets != nil {
		rulesets := make([]uimodel.Ruleset, 0, source.Rulesets.Len())
		for _, policy := range source.Rulesets.Values() {
			prepared, err := RulesetObservation(policy)
			if err != nil {
				return uimodel.Repository{}, fmt.Errorf("export Rulesets: %w", err)
			}
			rulesets = append(rulesets, prepared.Ruleset)
		}
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.Rulesets = &rulesets
	}
	actions, err := ActionsExecutionPolicyUI(source.Actions)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions execution: %w", err)
	}
	actions, err = WithActionsWorkflowPolicyUI(actions, source.Actions)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions workflow permissions: %w", err)
	}
	actions, err = WithActionsRetentionPolicyUI(actions, source.Actions.RetentionDays)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions retention: %w", err)
	}
	actions, err = WithActionsForkApprovalPolicyUI(actions, source.Actions.ForkApprovalPolicy)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions contributor approval: %w", err)
	}
	actions, err = WithActionsPrivateForkPolicyUI(actions, source.Actions.PrivateFork)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions fork workflows: %w", err)
	}
	if actions != (uimodel.ActionsGeneral{}) {
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.Actions = &uimodel.ActionsSettings{General: &actions}
	}
	if err := source.Security.Validate(); err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Security: %w", err)
	}
	if !source.Security.Empty() {
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		if !source.Security.DependabotEmpty() || !source.Security.SecretScanningEmpty() || source.Security.PrivateVulnerabilityReporting != nil {
			result.Settings.SecurityAndQuality = &uimodel.SecurityAndQuality{AdvancedSecurity: &uimodel.AdvancedSecurity{DependabotAlerts: copyUISecurityValue(source.Security.DependabotAlerts), DependabotSecurityUpdates: copyUISecurityValue(source.Security.DependabotSecurityUpdates), SecretProtection: copyUISecurityValue(source.Security.SecretScanning), PushProtection: copyUISecurityValue(source.Security.PushProtection), PrivateVulnerabilityReporting: copyUISecurityValue(source.Security.PrivateVulnerabilityReporting)}}
		}
		if source.Security.ImmutableReleases != nil {
			if result.Settings.General == nil {
				result.Settings.General = &uimodel.General{}
			}
			result.Settings.General.Releases = &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: copyUISecurityValue(source.Security.ImmutableReleases)}
		}
	}
	if !source.Pages.Empty() {
		pages, err := PagesPolicyUI(source.Pages)
		if err != nil {
			return uimodel.Repository{}, fmt.Errorf("export Pages: %w", err)
		}
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.Pages = &pages
	}
	if source.Environments != nil {
		environments := make([]uimodel.Environment, 0, source.Environments.Len())
		for _, policy := range source.Environments.Values() {
			environment, err := EnvironmentPolicyUI(policy)
			if err != nil {
				return uimodel.Repository{}, fmt.Errorf("export Environments: %w", err)
			}
			environments = append(environments, environment)
		}
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.Environments = &environments
	}
	return result, nil
}
