package workspace

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

// uiSettingsDocument is the only public Settings document. Family adapter models
// remain independent of YAML grouping; they are never decoded as a second format.
// Every scalar pointer preserves omission independently from explicit false/zero.
type uiSettingsDocument struct {
	General               *uimodel.General
	Collaborators         *[]uimodel.Collaborator
	Moderation            *uiModerationDocument
	Rulesets              *[]uimodel.Ruleset
	BranchProtectionRules *[]uimodel.BranchProtectionRule
	Actions               *uiActionsDocument
	Webhooks              *[]uimodel.Webhook
	Copilot               *uimodel.CopilotSettings
	Environments          *[]uimodel.Environment
	Pages                 *uimodel.Pages
	AdvancedSecurity      *uiAdvancedSecurityDocument
	DeployKeys            *[]uimodel.DeployKey
	Agents                *uimodel.AgentsSettings
	Autolinks             *[]uimodel.Autolink
}

type uiModerationDocument struct{ InteractionLimits *uimodel.InteractionLimits }

type uiActionsDocument struct {
	Runners   *[]uimodel.RunnerLabels
	OIDC      *uimodel.OIDCSettings
	Policies  *[]uimodel.ActionsEventPolicy
	General   *uiActionsGeneralDocument
	Variables *[]uimodel.RepositoryVariable
}

type uiActionsGeneralDocument struct {
	Cache                                                      *uimodel.ActionsCache
	ActionsPermissions                                         *uiActionsPermissionsDocument
	CheckWorkflowRunStatusArtifactAndLogRetention              *int
	ApprovalForRunningForkPullRequestWorkflowsFromContributors *uimodel.ForkApprovalChoice
	WorkflowPermissions                                        *uiWorkflowPermissionsDocument
}

type uiActionsPermissionsDocument struct {
	Policy                                           *uimodel.ActionsPermissionChoice
	RequireActionsToBePinnedToAFullLengthCommitSHA   *bool
	AllowActionsCreatedByGitHub                      *bool
	AllowActionsByMarketplaceVerifiedCreators        *bool
	AllowOrBlockSpecifiedActionsAndReusableWorkflows *[]string
}

type uiWorkflowPermissionsDocument struct {
	DefaultPermissions                               *uimodel.WorkflowPermissionChoice
	AllowGitHubActionsToCreateAndApprovePullRequests *bool
}

type uiDependabotDocument struct {
	Alerts          *bool
	SecurityUpdates *bool
}

type uiAdvancedSecurityDocument struct {
	CodeSecurity                  *uimodel.CodeSecurity
	Dependabot                    *uiDependabotDocument
	PrivateVulnerabilityReporting *bool
	SecretProtection              *uiSecretProtectionDocument
}

type uiSecretProtectionDocument struct {
	Enabled        *bool
	PushProtection *bool
}

func (document uiSettingsDocument) model() uimodel.RepositorySettings {
	result := uimodel.RepositorySettings{
		General:               document.General,
		Collaborators:         document.Collaborators,
		Rulesets:              document.Rulesets,
		BranchProtectionRules: document.BranchProtectionRules,
		Webhooks:              document.Webhooks,
		Copilot:               document.Copilot,
		Environments:          document.Environments,
		Pages:                 document.Pages,
		DeployKeys:            document.DeployKeys,
		Agents:                document.Agents,
		Autolinks:             document.Autolinks,
	}
	if document.Moderation != nil {
		result.InteractionLimits = document.Moderation.InteractionLimits
	}
	if document.Actions != nil {
		result.Actions = &uimodel.ActionsSettings{Runners: document.Actions.Runners, OIDC: document.Actions.OIDC, Variables: document.Actions.Variables, Policies: document.Actions.Policies}
		if document.Actions.General != nil {
			general := document.Actions.General.model()
			result.Actions.General = &general
		}
	}
	if security := document.AdvancedSecurity; security != nil {
		advanced := &uimodel.AdvancedSecurity{CodeSecurity: security.CodeSecurity, PrivateVulnerabilityReporting: security.PrivateVulnerabilityReporting}
		if security.Dependabot != nil {
			advanced.DependabotAlerts = security.Dependabot.Alerts
			advanced.DependabotSecurityUpdates = security.Dependabot.SecurityUpdates
		}
		if protection := security.SecretProtection; protection != nil {
			advanced.SecretProtection = protection.Enabled
			advanced.PushProtection = protection.PushProtection
		}
		result.SecurityAndQuality = &uimodel.SecurityAndQuality{AdvancedSecurity: advanced}
	}
	return result
}

func (document uiActionsGeneralDocument) model() uimodel.ActionsGeneral {
	result := uimodel.ActionsGeneral{
		Cache: document.Cache,
		CheckWorkflowRunStatusArtifactAndLogRetention:              document.CheckWorkflowRunStatusArtifactAndLogRetention,
		ApprovalForRunningForkPullRequestWorkflowsFromContributors: document.ApprovalForRunningForkPullRequestWorkflowsFromContributors,
	}
	if permissions := document.ActionsPermissions; permissions != nil {
		result.ActionsPermissions = permissions.Policy
		result.RequireActionsToBePinnedToAFullLengthCommitSHA = permissions.RequireActionsToBePinnedToAFullLengthCommitSHA
		result.AllowActionsCreatedByGitHub = permissions.AllowActionsCreatedByGitHub
		result.AllowActionsByMarketplaceVerifiedCreators = permissions.AllowActionsByMarketplaceVerifiedCreators
		result.AllowOrBlockSpecifiedActionsAndReusableWorkflows = permissions.AllowOrBlockSpecifiedActionsAndReusableWorkflows
	}
	if permissions := document.WorkflowPermissions; permissions != nil {
		result.WorkflowPermissions = permissions.DefaultPermissions
		result.AllowGitHubActionsToCreateAndApprovePullRequests = permissions.AllowGitHubActionsToCreateAndApprovePullRequests
	}
	return result
}

func uiSettingsDocumentFromModel(model uimodel.RepositorySettings) uiSettingsDocument {
	result := uiSettingsDocument{
		General:               model.General,
		Collaborators:         model.Collaborators,
		Rulesets:              model.Rulesets,
		BranchProtectionRules: model.BranchProtectionRules,
		Webhooks:              model.Webhooks,
		Copilot:               model.Copilot,
		Environments:          model.Environments,
		Pages:                 model.Pages,
		DeployKeys:            model.DeployKeys,
		Agents:                model.Agents,
		Autolinks:             model.Autolinks,
	}
	if model.InteractionLimits != nil {
		result.Moderation = &uiModerationDocument{InteractionLimits: model.InteractionLimits}
	}
	if model.Actions != nil {
		result.Actions = &uiActionsDocument{Runners: model.Actions.Runners, OIDC: model.Actions.OIDC, Variables: model.Actions.Variables, Policies: model.Actions.Policies}
		if model.Actions.General != nil {
			general := uiActionsGeneralDocumentFromModel(*model.Actions.General)
			result.Actions.General = &general
		}
	}
	if model.SecurityAndQuality != nil && model.SecurityAndQuality.AdvancedSecurity != nil {
		security := model.SecurityAndQuality.AdvancedSecurity
		result.AdvancedSecurity = &uiAdvancedSecurityDocument{CodeSecurity: security.CodeSecurity, PrivateVulnerabilityReporting: security.PrivateVulnerabilityReporting}
		if security.DependabotAlerts != nil || security.DependabotSecurityUpdates != nil {
			result.AdvancedSecurity.Dependabot = &uiDependabotDocument{Alerts: security.DependabotAlerts, SecurityUpdates: security.DependabotSecurityUpdates}
		}
		if security.SecretProtection != nil || security.PushProtection != nil {
			result.AdvancedSecurity.SecretProtection = &uiSecretProtectionDocument{
				Enabled: security.SecretProtection, PushProtection: security.PushProtection,
			}
		}
	}
	return result
}

func uiActionsGeneralDocumentFromModel(model uimodel.ActionsGeneral) uiActionsGeneralDocument {
	result := uiActionsGeneralDocument{
		Cache: model.Cache,
		CheckWorkflowRunStatusArtifactAndLogRetention:              model.CheckWorkflowRunStatusArtifactAndLogRetention,
		ApprovalForRunningForkPullRequestWorkflowsFromContributors: model.ApprovalForRunningForkPullRequestWorkflowsFromContributors,
	}
	if model.ActionsPermissions != nil || model.RequireActionsToBePinnedToAFullLengthCommitSHA != nil ||
		model.AllowActionsCreatedByGitHub != nil || model.AllowActionsByMarketplaceVerifiedCreators != nil ||
		model.AllowOrBlockSpecifiedActionsAndReusableWorkflows != nil {
		result.ActionsPermissions = &uiActionsPermissionsDocument{
			Policy: model.ActionsPermissions,
			RequireActionsToBePinnedToAFullLengthCommitSHA:   model.RequireActionsToBePinnedToAFullLengthCommitSHA,
			AllowActionsCreatedByGitHub:                      model.AllowActionsCreatedByGitHub,
			AllowActionsByMarketplaceVerifiedCreators:        model.AllowActionsByMarketplaceVerifiedCreators,
			AllowOrBlockSpecifiedActionsAndReusableWorkflows: model.AllowOrBlockSpecifiedActionsAndReusableWorkflows,
		}
	}
	if model.WorkflowPermissions != nil || model.AllowGitHubActionsToCreateAndApprovePullRequests != nil {
		result.WorkflowPermissions = &uiWorkflowPermissionsDocument{
			DefaultPermissions: model.WorkflowPermissions,
			AllowGitHubActionsToCreateAndApprovePullRequests: model.AllowGitHubActionsToCreateAndApprovePullRequests,
		}
	}
	return result
}
