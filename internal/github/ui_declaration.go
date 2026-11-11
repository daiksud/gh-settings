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
		if page := repository.Settings.InteractionLimits; page != nil && page.PullRequestLimits != nil {
			c := page.PullRequestLimits
			result.PullRequestLimits = settings.PullRequestLimitsDeclaration{Enabled: copyUIValue(c.LimitOpenPullRequestsFromUsersWithoutWriteAccess), Maximum: copyUIValue(c.MaximumOpenPullRequestsPerUser), IncludeDrafts: copyUIValue(c.CountDraftPullRequestsTowardTheLimit)}
			if err := result.PullRequestLimits.Validate(); err != nil {
				return settings.Declaration{}, err
			}
		}

		if repository.Settings.Copilot != nil && repository.Settings.Copilot.CodeReview != nil {
			result.CopilotInstructions = copyUIValue(repository.Settings.Copilot.CodeReview.UseCustomInstructions)
			result.CopilotMCP = copyUIValue(repository.Settings.Copilot.CodeReview.AllowCopilotToUseMCPToolsWhenReviewingPullRequests)
			result.CopilotReviewEffort = copyUIValue(repository.Settings.Copilot.CodeReview.ReviewEffortLevel)
			if result.CopilotReviewEffort != nil {
				if err := settings.ValidateCopilotReviewEffort(*result.CopilotReviewEffort); err != nil {
					return settings.Declaration{}, err
				}
			}
		}
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
		environments, err := EnvironmentCollectionDeclaration(repository.Settings.Environments)
		if err != nil {
			return settings.Declaration{}, fmt.Errorf("repository.settings.environments: %w", err)
		}
		result.Environments = environments
		environmentVariables, err := EnvironmentVariablesDeclaration(repository.Settings.Environments)
		if err != nil {
			return settings.Declaration{}, fmt.Errorf("repository.settings.environments: %w", err)
		}
		result.EnvironmentVariables = environmentVariables
		if repository.Settings.Pages != nil {
			pages, err := PagesDeclaration(*repository.Settings.Pages)
			if err != nil {
				return settings.Declaration{}, fmt.Errorf("repository.settings.pages: %w", err)
			}
			result.Pages = pages
		}
		rulesets, err := RulesetsDeclaration(repository.Settings.Rulesets)
		if err != nil {
			return settings.Declaration{}, fmt.Errorf("repository.settings.rulesets: %w", err)
		}
		result.Rulesets = rulesets
		if repository.Settings.Agents != nil {
			variables, err := VariablesDeclaration(repository.Settings.Agents.Variables)
			if err != nil {
				return settings.Declaration{}, fmt.Errorf("repository.settings.agents.variables: %w", err)
			}
			result.AgentsVariables = variables
		}
		if repository.Settings.Actions != nil {
			policies, err := ActionsPoliciesDeclaration(repository.Settings.Actions.Policies)
			if err != nil {
				return settings.Declaration{}, fmt.Errorf("repository.settings.actions.policies: %w", err)
			}
			result.ActionsPolicies = policies
			variables, err := VariablesDeclaration(repository.Settings.Actions.Variables)
			if err != nil {
				return settings.Declaration{}, fmt.Errorf("repository.settings.actions.variables: %w", err)
			}
			result.Variables = variables
		}
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
