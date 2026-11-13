package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// UIFromExportDeclaration projects acquired values and valid retained intent.
// It does not establish acquisition or fill missing observation metadata.
func UIFromExportDeclaration(source settings.Declaration) (uimodel.Repository, error) {
	var topics *[]string
	if source.Topics.Names != nil {
		v := settings.SortedTopics(*source.Topics.Names)
		topics = &v
	}
	var result uimodel.Repository
	if source.Repository.Description != nil || source.Repository.Homepage != nil || topics != nil {
		result.About = &uimodel.About{Description: copyUIValue(source.Repository.Description), Website: copyUIValue(source.Repository.Homepage), Topics: topics}
	}
	general, err := GeneralObservation(source.Repository)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export General: %w", err)
	}
	if general.General != (uimodel.General{}) {
		result.Settings = &uimodel.RepositorySettings{General: &general.General}
	}
	if source.Collaborators != nil {
		values := make([]uimodel.Collaborator, 0, source.Collaborators.Len())
		for _, person := range source.Collaborators.Values() {
			values = append(values, uimodel.Collaborator{Username: person.Username, ID: person.ID})
		}
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.Collaborators = &values
	}
	if source.Labels != nil {
		labels := make([]uimodel.Label, 0, source.Labels.Len())
		for _, policy := range settings.SortedLabels(source.Labels.Values()) {
			label, err := labelExportDeclarationUI(policy)
			if err != nil {
				return uimodel.Repository{}, fmt.Errorf("export Labels: %w", err)
			}
			labels = append(labels, label)
		}
		result.Labels = &labels
	}
	if source.BranchProtectionRules != nil {
		if err := settings.ValidateClassicRules(*source.BranchProtectionRules); err != nil {
			return uimodel.Repository{}, err
		}
		rules := make([]uimodel.BranchProtectionRule, 0, len(*source.BranchProtectionRules))
		for _, v := range *source.BranchProtectionRules {
			rules = append(rules, settings.CloneClassicRule(v))
		}
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.BranchProtectionRules = &rules
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
	actions, err := actionsExportDeclarationUI(source.Actions)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions execution: %w", err)
	}
	actions, err = WithActionsCachePolicyUI(actions, source.Actions)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions cache: %w", err)
	}
	actions, err = WithActionsRetentionPolicyUI(actions, source.Actions.RetentionDays)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions retention: %w", err)
	}
	actions, err = WithActionsForkApprovalPolicyUI(actions, source.Actions.ForkApprovalPolicy)
	if err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Actions contributor approval: %w", err)
	}
	if actions != (uimodel.ActionsGeneral{}) {
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.Actions = &uimodel.ActionsSettings{General: &actions}
	}
	if source.ActionsPolicies != nil {
		values := make([]uimodel.ActionsEventPolicy, 0, source.ActionsPolicies.Len())
		for _, entry := range source.ActionsPolicies.Values() {
			value, err := ActionsEventPolicyUI(entry)
			if err != nil {
				return uimodel.Repository{}, fmt.Errorf("export Actions policies: %w", err)
			}
			values = append(values, value)
		}
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		if result.Settings.Actions == nil {
			result.Settings.Actions = &uimodel.ActionsSettings{}
		}
		result.Settings.Actions.Policies = &values
	}
	if source.Variables != nil {
		values := variableValuesUI(*source.Variables)
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		if result.Settings.Actions == nil {
			result.Settings.Actions = &uimodel.ActionsSettings{}
		}
		result.Settings.Actions.Variables = &values
	}
	if source.OIDCSubject != nil {
		if err := settings.ValidateOIDCSubject(*source.OIDCSubject); err != nil {
			return uimodel.Repository{}, err
		}
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		if result.Settings.Actions == nil {
			result.Settings.Actions = &uimodel.ActionsSettings{}
		}
		copy := settings.CloneOIDCSubject(*source.OIDCSubject)
		result.Settings.Actions.OIDC = &uimodel.OIDCSettings{SubjectClaim: &copy}
	}
	if source.AgentsVariables != nil {
		values := variableValuesUI(*source.AgentsVariables)
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.Agents = &uimodel.AgentsSettings{Variables: &values}
	}
	if source.CopilotReviewEffort != nil {
		if err := settings.ValidateCopilotReviewEffort(*source.CopilotReviewEffort); err != nil {
			return uimodel.Repository{}, err
		}
	}
	if !source.PullRequestLimits.Empty() {
		if err := source.PullRequestLimits.Validate(); err != nil {
			return uimodel.Repository{}, err
		}
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		c := source.PullRequestLimits
		result.Settings.InteractionLimits = &uimodel.InteractionLimits{PullRequestLimits: &uimodel.PullRequestLimits{LimitOpenPullRequestsFromUsersWithoutWriteAccess: copyUIValue(c.Enabled), MaximumOpenPullRequestsPerUser: copyUIValue(c.Maximum), CountDraftPullRequestsTowardTheLimit: copyUIValue(c.IncludeDrafts)}}
	}

	if source.PullRequestExemptions != nil {
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		if result.Settings.InteractionLimits == nil {
			result.Settings.InteractionLimits = &uimodel.InteractionLimits{}
		}
		if result.Settings.InteractionLimits.PullRequestLimits == nil {
			result.Settings.InteractionLimits.PullRequestLimits = &uimodel.PullRequestLimits{}
		}
		logins := source.PullRequestExemptions.Logins()
		result.Settings.InteractionLimits.PullRequestLimits.ExemptionList = &logins
	}

	if source.CopilotMCP != nil || source.CopilotInstructions != nil || source.CopilotReviewEffort != nil {
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		result.Settings.Copilot = &uimodel.CopilotSettings{CodeReview: &uimodel.CopilotCodeReview{AllowCopilotToUseMCPToolsWhenReviewingPullRequests: copyUIValue(source.CopilotMCP), UseCustomInstructions: copyUIValue(source.CopilotInstructions), ReviewEffortLevel: copyUIValue(source.CopilotReviewEffort)}}
	}
	if err := source.Security.Validate(); err != nil {
		return uimodel.Repository{}, fmt.Errorf("export Security: %w", err)
	}
	if !source.Security.Empty() {
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		if !source.Security.DependabotEmpty() || !source.Security.SecretScanningEmpty() || source.Security.PrivateVulnerabilityReporting != nil || source.Security.CodeQLDefaultSetup != nil || (source.Security.AIScanPullRequests != nil || source.Security.AIScanRepositories != nil) || !source.Security.CodeQLConfiguration.Empty() {
			result.Settings.SecurityAndQuality = &uimodel.SecurityAndQuality{AdvancedSecurity: &uimodel.AdvancedSecurity{DependabotAlerts: copyUISecurityValue(source.Security.DependabotAlerts), DependabotSecurityUpdates: copyUISecurityValue(source.Security.DependabotSecurityUpdates), SecretProtection: copyUISecurityValue(source.Security.SecretScanning), PushProtection: copyUISecurityValue(source.Security.PushProtection), PrivateVulnerabilityReporting: copyUISecurityValue(source.Security.PrivateVulnerabilityReporting)}}
		}
		if source.Security.CodeQLDefaultSetup != nil || !source.Security.CodeQLConfiguration.Empty() {
			result.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity = &uimodel.CodeSecurity{CodeQLAnalysis: &uimodel.CodeQLAnalysis{DefaultSetup: copyUISecurityValue(source.Security.CodeQLDefaultSetup), Languages: source.Security.CodeQLConfiguration.Languages, QuerySuite: source.Security.CodeQLConfiguration.QuerySuite, ThreatModel: source.Security.CodeQLConfiguration.ThreatModel}}
			if runner := source.Security.CodeQLConfiguration.Runner; runner != nil {
				ui := &uimodel.CodeQLRunner{Type: runner.Type}
				if runner.Type == "labeled_runner" {
					ui.Label = copyUISecurityValue(&runner.Label)
				}
				result.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity.CodeQLAnalysis.Runner = ui
			}
		}
		if source.Security.AIScanPullRequests != nil || source.Security.AIScanRepositories != nil {
			advanced := result.Settings.SecurityAndQuality.AdvancedSecurity
			if advanced.CodeSecurity == nil {
				advanced.CodeSecurity = &uimodel.CodeSecurity{}
			}
			advanced.CodeSecurity.AIScan = &uimodel.AIScan{AIScanForPullRequests: copyUISecurityValue(source.Security.AIScanPullRequests), AIScanForRepositories: copyUISecurityValue(source.Security.AIScanRepositories)}
		}

		if source.Security.ImmutableReleases != nil {
			if result.Settings.General == nil {
				result.Settings.General = &uimodel.General{}
			}
			result.Settings.General.Releases = &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: copyUISecurityValue(source.Security.ImmutableReleases)}
		}
	}
	if !source.Pages.Empty() {
		pages, err := pagesExportDeclarationUI(source.Pages)
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
			environment, err := EnvironmentReportUI(policy)
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
	if len(source.EnvironmentVariables) != 0 {
		if result.Settings == nil {
			result.Settings = &uimodel.RepositorySettings{}
		}
		if result.Settings.Environments == nil {
			entries := []uimodel.Environment{}
			result.Settings.Environments = &entries
		}
		for _, scoped := range source.EnvironmentVariables {
			if _, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: scoped.Environment}}); err != nil {
				return uimodel.Repository{}, err
			}
			values := variableValuesUI(scoped.Variables)
			index := -1
			for i, entry := range *result.Settings.Environments {
				if settings.EnvironmentIdentity(entry.Name) == settings.EnvironmentIdentity(scoped.Environment) {
					index = i
					break
				}
			}
			if index < 0 {
				*result.Settings.Environments = append(*result.Settings.Environments, uimodel.Environment{Name: scoped.Environment, Variables: &values})
			} else {
				if (*result.Settings.Environments)[index].Variables != nil {
					return uimodel.Repository{}, fmt.Errorf("duplicate environment variable export scope %q", scoped.Environment)
				}
				(*result.Settings.Environments)[index].Variables = &values
			}
		}
	}
	return result, nil
}

// actionsExportDeclarationUI projects validated intent, which may contain only
// one control from an unavailable family. API observations are checked before
// becoming declarations; projecting retained intent must not invent acquisition.
func actionsExportDeclarationUI(policy settings.ActionsPolicyDeclaration) (uimodel.ActionsGeneral, error) {
	if err := policy.Validate(); err != nil {
		return uimodel.ActionsGeneral{}, err
	}
	ui := uimodel.ActionsGeneral{RequireActionsToBePinnedToAFullLengthCommitSHA: copyUIValue(policy.SHAPinningRequired), AllowGitHubActionsToCreateAndApprovePullRequests: copyUIValue(policy.CanApprovePullRequestReviews)}
	if policy.Enabled != nil {
		allowed := ""
		if policy.AllowedActions != nil {
			allowed = *policy.AllowedActions
		}
		choice, valid := settings.ActionsPermissionsChoiceForValue(*policy.Enabled, allowed)
		if !valid {
			return uimodel.ActionsGeneral{}, fmt.Errorf("Actions permissions declaration is outside UI choices")
		}
		ui.ActionsPermissions = &choice
	} else if policy.AllowedActions != nil {
		return uimodel.ActionsGeneral{}, fmt.Errorf("Actions permissions declaration requires an explicit UI choice")
	}
	// Observed disabled policies may retain API-only latent selected values.
	// A declaration with no permission choice can independently manage companions.
	if policy.Enabled == nil || *policy.Enabled {
		ui.AllowActionsCreatedByGitHub = copyUIValue(policy.SelectedActions.GitHubOwnedAllowed)
		ui.AllowActionsByMarketplaceVerifiedCreators = copyUIValue(policy.SelectedActions.VerifiedAllowed)
		if policy.SelectedActions.PatternsAllowed != nil {
			patterns := append([]string{}, (*policy.SelectedActions.PatternsAllowed)...)
			ui.AllowOrBlockSpecifiedActionsAndReusableWorkflows = &patterns
		}
	}
	if policy.DefaultWorkflowPermissions != nil {
		choice := uimodel.ReadRepositoryContentsAndPackagesPermissions
		if *policy.DefaultWorkflowPermissions == "write" {
			choice = uimodel.ReadAndWritePermissions
		}
		ui.WorkflowPermissions = &choice
	}
	return ui, nil
}

func labelExportDeclarationUI(policy settings.LabelDeclaration) (uimodel.Label, error) {
	if err := settings.ValidateLabels([]settings.LabelDeclaration{policy}); err != nil {
		return uimodel.Label{}, err
	}
	if policy.Color == nil {
		return uimodel.Label{}, fmt.Errorf("label declaration requires color")
	}
	color := "#" + *policy.Color
	return uimodel.Label{ID: copyUIValue(policy.ID), Name: policy.Name, Color: &color, Description: copyUIValue(policy.Description)}, nil
}

func pagesExportDeclarationUI(policy settings.PagesDeclaration) (uimodel.Pages, error) {
	if err := policy.Validate(); err != nil {
		return uimodel.Pages{}, err
	}
	ui := uimodel.Pages{EnforceHTTPS: copyUIValue(policy.HTTPSEnforced)}
	if policy.BuildType != nil || policy.Source != nil {
		b := &uimodel.PagesBuildAndDeployment{}
		if policy.BuildType != nil {
			choice, valid := settings.PagesModeChoiceForValue(*policy.BuildType)
			if !valid {
				return uimodel.Pages{}, fmt.Errorf("Pages source is outside UI choices")
			}
			b.Source = &choice
		}
		if policy.Source != nil {
			b.Branch = copyUIValue(&policy.Source.Branch)
			if !policy.Delete() {
				folder, valid := settings.PagesFolderChoiceForValue(policy.Source.Path)
				if !valid {
					return uimodel.Pages{}, fmt.Errorf("Pages folder is outside UI choices")
				}
				b.SelectFolder = &folder
			}
		}
		ui.BuildAndDeployment = b
	}
	if policy.CNAME != nil {
		ui.CustomDomain = &uimodel.PagesCustomDomain{Value: copyUIValue(policy.CNAME.Value)}
	}
	return ui, nil
}
