package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// PreWriteReadFailure confirms that a writer's prerequisite read failed before
// any write attempt. PUT and post-write verification errors must not mark this.
type PreWriteReadFailure interface {
	error
	ReadFailedBeforeWrite()
}

// RepositoryWriteEffectsFailure marks a composite repository operation whose
// earlier write substep completed before a later failure. It takes precedence
// over an inner substep's PreWriteReadFailure without discarding that cause.
type RepositoryWriteEffectsFailure interface {
	error
	RepositoryWritesMayHaveApplied()
}

// EnvironmentWriteEffectsFailure confirms an independently verified policy
// update before a later pattern or readback failure in the same environment.
type EnvironmentWriteEffectsFailure interface {
	error
	EnvironmentWritesCompleted()
}

// Reader defers each API client until the operation needs that resource.
type Reader struct {
	OIDCSubject           func(repository.Repository) (settings.OIDCSubjectObservation, error)
	AIScanRepositories    func(repository.Repository) (settings.AIScanRepositoriesObservation, error)
	BranchProtectionRules func(repository.Repository) (settings.ClassicObservation, error)
	EnvironmentPrune      func(repository.Repository, []string) (settings.EnvironmentPruneObservation, error)
	Collaborators         func(repository.Repository) (settings.CollaboratorObservation, error)
	ResolveCollaborator   func(repository.Repository, settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error)
	EnvironmentNames      func(repository.Repository) ([]string, error)
	CodeQLForExport       func(repository.Repository) (settings.CodeQLDefaultSetupObservation, settings.CodeQLConfigurationObservation, error)
	Variables             func(repository.Repository) (settings.VariableInventory, error)
	AgentVariables        func(repository.Repository) (settings.VariableInventory, error)
	EnvironmentVariables  func(repository.Repository, string) (settings.VariableInventory, error)
	CollaboratorPrune     func(repository.Repository, settings.CollaboratorDeclarations) (settings.CollaboratorPruneObservation, error)

	PullRequestExemptions func(repository.Repository) (settings.PullRequestExemptionsObservation, error)
	EnvironmentVariable   func(repository.Repository, string, string) (settings.VariableObservation, error)
	ActionsPolicies       func(repository.Repository) (settings.ActionsPolicyObservation, error)
	CopilotReviewEffort   func(repository.Repository) (settings.CopilotReviewEffortObservation, error)
	CopilotInstructions   func(repository.Repository) (settings.CopilotInstructionsObservation, error)
	CopilotMCP            func(repository.Repository) (settings.CopilotMCPObservation, error)
	PullRequestLimits     func(repository.Repository) (settings.PullRequestLimitsObservation, error)
	AgentVariable         func(repository.Repository, string) (settings.VariableObservation, error)
	Variable              func(repository.Repository, string) (settings.VariableObservation, error)
	CodeQLConfiguration   func(repository.Repository) (settings.CodeQLConfigurationObservation, error)
	ActionsAccess         func(repository.Repository) (settings.ActionsAccessObservation, error)
	CacheStorage          func(repository.Repository) (settings.CacheStorageObservation, error)
	CacheRetention        func(repository.Repository) (settings.CacheRetentionObservation, error)
	Dependabot            func(repository.Repository) (settings.DependabotObservation, error)
	ImmutableReleases     func(repository.Repository) (settings.ImmutableReleasesObservation, error)
	Pages                 func(repository.Repository) (settings.PagesObservation, error)
	Archive               func(repository.Repository) (settings.ArchiveState, error)
	ForkApproval          func(repository.Repository) (settings.ForkApprovalObservation, error)
	PrivateFork           func(repository.Repository) (settings.PrivateForkObservation, error)
	Retention             func(repository.Repository) (settings.RetentionObservation, error)
	CodeQLDefaultSetup    func(repository.Repository) (settings.CodeQLDefaultSetupObservation, error)
	AIScanPullRequests    func(repository.Repository) (settings.AIScanPullRequestsObservation, error)
	PrivateReporting      func(repository.Repository) (settings.PrivateReportingObservation, error)
	SecretScanning        func(repository.Repository) (settings.SecretScanningObservation, error)
	WorkflowPermissions   func(repository.Repository) (settings.WorkflowPermissionsState, error)
	Actions               func(repository.Repository, bool) (settings.ActionsPolicyState, error)
	ObserveRepository     func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error)
	Topics                func(repository.Repository) ([]string, error)
	Labels                func(repository.Repository) (settings.LabelObservation, error)
	Rulesets              func(repository.Repository) (settings.RulesetObservation, error)
	Environments          func(repository.Repository, []string) (settings.EnvironmentObservation, error)
}

// Writer sends each resource independently; successful earlier writes are not rolled back.
type Writer struct {
	OIDCSubject                func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error)
	ReadAIScanRepositories     func(repository.Repository) (settings.AIScanRepositoriesObservation, error)
	AIScanRepositories         func(repository.Repository, settings.SecurityChange) (bool, error)
	ReadCollaboratorPrune      func(repository.Repository, settings.CollaboratorDeclarations) (settings.CollaboratorPruneObservation, error)
	RefreshCollaboratorPrune   func(repository.Repository, settings.CollaboratorPruneObservation, *settings.CollaboratorWriteResult) (settings.CollaboratorPruneObservation, error)
	DeleteCollaborator         func(repository.Repository, settings.CollaboratorPruneObservation, settings.CollaboratorDeletion) (settings.CollaboratorPruneObservation, settings.CollaboratorDeleteResult, error)
	ReadEnvironmentPrune       func(repository.Repository, []string) (settings.EnvironmentPruneObservation, error)
	DeleteEnvironment          func(repository.Repository, settings.EnvironmentPruneCandidate) error
	ReadBranchProtectionRules  func(repository.Repository) (settings.ClassicObservation, error)
	BranchProtectionRule       func(repository.Repository, settings.ClassicSelection) (settings.ClassicWriteResult, error)
	DeleteBranchProtectionRule func(repository.Repository, settings.ClassicPruneCandidate) error
	ReadCollaborators          func(repository.Repository) (settings.CollaboratorObservation, error)
	ResolveCollaborator        func(repository.Repository, settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error)
	Collaborator               func(repository.Repository, settings.CollaboratorSelection) (settings.CollaboratorWriteResult, error)
	BindCollaboratorID         func(string, int64) error
	ReadVariables              func(repository.Repository) (settings.VariableInventory, error)
	ReadAgentVariables         func(repository.Repository) (settings.VariableInventory, error)
	ReadEnvironmentVariables   func(repository.Repository, string) (settings.VariableInventory, error)
	DeleteVariable             func(repository.Repository, settings.VariableChange) error
	DeleteAgentVariable        func(repository.Repository, settings.VariableChange) error
	DeleteEnvironmentVariable  func(repository.Repository, string, int64, settings.VariableChange) error

	PullRequestExemptions   func(repository.Repository, settings.PullRequestExemptionsDeclaration) (bool, error)
	EnvironmentVariable     func(repository.Repository, string, settings.RepositoryVariable) (bool, error)
	ReadActionsPolicies     func(repository.Repository) (settings.ActionsPolicyObservation, error)
	ActionsEventPolicy      func(repository.Repository, settings.ActionsEventPolicyChange) (int64, error)
	CopilotReviewEffort     func(repository.Repository, string) (bool, error)
	CopilotInstructions     func(repository.Repository, bool) (bool, error)
	CopilotMCP              func(repository.Repository, bool) (bool, error)
	PullRequestLimits       func(repository.Repository, settings.PullRequestLimitsDeclaration) (bool, error)
	AgentVariable           func(repository.Repository, settings.RepositoryVariable) (bool, error)
	Variable                func(repository.Repository, settings.RepositoryVariable) (bool, error)
	ReadCodeQLConfiguration func(repository.Repository) (settings.CodeQLConfigurationObservation, error)
	CodeQLConfiguration     func(repository.Repository, settings.CodeQLConfigurationDeclaration) (bool, error)
	ReadActionsAccess       func(repository.Repository) (settings.ActionsAccessObservation, error)
	ActionsAccess           func(repository.Repository, string) (bool, error)
	ReadCacheStorage        func(repository.Repository) (settings.CacheStorageObservation, error)
	CacheStorage            func(repository.Repository, int) (bool, error)
	ReadCacheRetention      func(repository.Repository) (settings.CacheRetentionObservation, error)
	CacheRetention          func(repository.Repository, int) (bool, error)
	ReadDependabot          func(repository.Repository) (settings.DependabotObservation, error)
	Dependabot              func(repository.Repository, settings.SecurityChange) (settings.DependabotWriteResult, error)
	ReadImmutableReleases   func(repository.Repository) (settings.ImmutableReleasesObservation, error)
	ImmutableReleases       func(repository.Repository, settings.SecurityChange) (bool, error)
	ReadPages               func(repository.Repository) (settings.PagesObservation, error)
	Pages                   func(repository.Repository, settings.PagesDeclaration) (bool, error)
	ReadArchive             func(repository.Repository) (settings.ArchiveState, error)
	Archive                 func(repository.Repository, settings.ArchiveState, bool) error
	ReadForkApproval        func(repository.Repository) (settings.ForkApprovalObservation, error)
	ForkApproval            func(repository.Repository, string) (bool, error)
	ReadPrivateFork         func(repository.Repository) (settings.PrivateForkObservation, error)
	PrivateFork             func(repository.Repository, settings.PrivateForkDeclaration) (bool, error)
	ReadRetention           func(repository.Repository) (settings.RetentionObservation, error)
	Retention               func(repository.Repository, int) (bool, error)
	ReadCodeQLDefaultSetup  func(repository.Repository) (settings.CodeQLDefaultSetupObservation, error)
	ReadAIScanPullRequests  func(repository.Repository) (settings.AIScanPullRequestsObservation, error)
	ReadPrivateReporting    func(repository.Repository) (settings.PrivateReportingObservation, error)
	CodeQLDefaultSetup      func(repository.Repository, settings.SecurityChange) (bool, error)
	AIScanPullRequests      func(repository.Repository, settings.SecurityChange) (bool, error)
	PrivateReporting        func(repository.Repository, settings.SecurityChange) (bool, error)
	ReadSecretScanning      func(repository.Repository) (settings.SecretScanningObservation, error)
	SecretScanning          func(repository.Repository, settings.SecurityChange) (bool, error)
	WorkflowPermissions     func(repository.Repository, settings.ActionsPolicyDeclaration) (bool, error)
	Actions                 func(repository.Repository, settings.ActionsPolicyDeclaration) (bool, error)
	Repository              func(repository.Repository, settings.RepositoryDeclaration) (bool, error)
	Topics                  func(repository.Repository, settings.TopicsDeclaration) (bool, error)
	ReadLabels              func(repository.Repository) (settings.LabelObservation, error)
	CreateLabel             func(repository.Repository, settings.LabelChange) (int64, error)
	CreatedID               func(string, string, int64) error
	UpdateLabel             func(repository.Repository, settings.LabelChange) error
	DeleteLabel             func(repository.Repository, settings.LabelChange) error
	ReadRulesets            func(repository.Repository) (settings.RulesetObservation, error)
	Ruleset                 func(repository.Repository, settings.RulesetChange) (int64, error)
	ReadEnvironments        func(repository.Repository, []string) (settings.EnvironmentObservation, error)
	Environment             func(repository.Repository, settings.EnvironmentDeclaration) (bool, error)
}
