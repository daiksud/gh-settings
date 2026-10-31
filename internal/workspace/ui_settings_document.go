package workspace

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

// uiSettingsDocument is the only public Settings document. Family adapter models
// remain independent of YAML grouping; they are never decoded as a second format.
// Every scalar pointer preserves omission independently from explicit false/zero.
type uiSettingsDocument struct {
	General          *uiGeneralDocument
	Rulesets         *[]uimodel.Ruleset
	Actions          *uiActionsDocument
	AdvancedSecurity *uiAdvancedSecurityDocument
	Pages            *uimodel.Pages
	Environments     *[]uimodel.Environment
}

type uiGeneralDocument struct {
	TemplateRepository *bool
	DefaultBranch      *string
	Releases           *uiReleasesDocument
	Features           uimodel.Features
	PullRequests       uimodel.PullRequests
	Commits            uimodel.Commits
	DangerZone         uimodel.DangerZone
}

type uiReleasesDocument struct{ ImmutableRelease *bool }

type uiActionsDocument struct{ General *uiActionsGeneralDocument }

type uiActionsGeneralDocument struct {
	ActionsPermissions                                         *uiActionsPermissionsDocument
	CheckWorkflowRunStatusArtifactAndLogRetention              *int
	ApprovalForRunningForkPullRequestWorkflowsFromContributors *uimodel.ForkApprovalChoice
	WorkflowPermissions                                        *uiWorkflowPermissionsDocument
	ForkPullRequestWorkflows                                   *uimodel.ForkPullRequestWorkflows
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

type uiAdvancedSecurityDocument struct {
	PrivateVulnerabilityReporting *bool
	SecretProtection              *uiSecretProtectionDocument
}

type uiSecretProtectionDocument struct {
	Enabled        *bool
	PushProtection *bool
}

func (document uiSettingsDocument) model() uimodel.RepositorySettings {
	result := uimodel.RepositorySettings{
		Rulesets: document.Rulesets, Pages: document.Pages, Environments: document.Environments,
	}
	if general := document.General; general != nil {
		result.General = &uimodel.General{
			TemplateRepository: general.TemplateRepository, DefaultBranch: general.DefaultBranch,
			Features: general.Features, PullRequests: general.PullRequests,
			Commits: general.Commits, DangerZone: general.DangerZone,
		}
		if general.Releases != nil {
			result.General.Releases = &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: general.Releases.ImmutableRelease}
		}
	}
	if document.Actions != nil {
		result.Actions = &uimodel.ActionsSettings{}
		if document.Actions.General != nil {
			general := document.Actions.General.model()
			result.Actions.General = &general
		}
	}
	if security := document.AdvancedSecurity; security != nil {
		advanced := &uimodel.AdvancedSecurity{PrivateVulnerabilityReporting: security.PrivateVulnerabilityReporting}
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
		CheckWorkflowRunStatusArtifactAndLogRetention:              document.CheckWorkflowRunStatusArtifactAndLogRetention,
		ApprovalForRunningForkPullRequestWorkflowsFromContributors: document.ApprovalForRunningForkPullRequestWorkflowsFromContributors,
		ForkPullRequestWorkflows:                                   document.ForkPullRequestWorkflows,
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
	result := uiSettingsDocument{Rulesets: model.Rulesets, Pages: model.Pages, Environments: model.Environments}
	if general := model.General; general != nil {
		result.General = &uiGeneralDocument{
			TemplateRepository: general.TemplateRepository, DefaultBranch: general.DefaultBranch,
			Features: general.Features, PullRequests: general.PullRequests,
			Commits: general.Commits, DangerZone: general.DangerZone,
		}
		if general.Releases != nil {
			result.General.Releases = &uiReleasesDocument{ImmutableRelease: general.Releases.EnableReleaseImmutability}
		}
	}
	if model.Actions != nil {
		result.Actions = &uiActionsDocument{}
		if model.Actions.General != nil {
			general := uiActionsGeneralDocumentFromModel(*model.Actions.General)
			result.Actions.General = &general
		}
	}
	if model.SecurityAndQuality != nil && model.SecurityAndQuality.AdvancedSecurity != nil {
		security := model.SecurityAndQuality.AdvancedSecurity
		result.AdvancedSecurity = &uiAdvancedSecurityDocument{PrivateVulnerabilityReporting: security.PrivateVulnerabilityReporting}
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
		CheckWorkflowRunStatusArtifactAndLogRetention:              model.CheckWorkflowRunStatusArtifactAndLogRetention,
		ApprovalForRunningForkPullRequestWorkflowsFromContributors: model.ApprovalForRunningForkPullRequestWorkflowsFromContributors,
		ForkPullRequestWorkflows:                                   model.ForkPullRequestWorkflows,
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
