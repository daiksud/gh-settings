package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type ApplyOutcome uint8

const (
	ApplyUnmanaged ApplyOutcome = iota
	ApplyCompleted
	ApplyMatchingLabels
	ApplyFailed
	ApplyPartialFailure
)

type ApplyOperationKind uint8

const (
	RepositoryUpdate ApplyOperationKind = iota
	TopicsUpdate
	LabelsRead
	LabelCreate
	LabelUpdate
	LabelDelete
	RulesetsRead
	RulesetCreate
	RulesetUpdate
	RulesetDelete
	ActionsPolicyUpdate
	WorkflowPermissionsUpdate
	SecretScanningRead
	SecretScanningUpdate
	CodeQLConfigurationRead
	CodeQLConfigurationUpdate
	CodeQLDefaultSetupRead
	CodeQLDefaultSetupUpdate
	AIScanRepositoriesRead
	AIScanRepositoriesUpdate
	AIScanPullRequestsRead
	AIScanPullRequestsUpdate
	PrivateReportingRead
	PrivateReportingUpdate
	ImmutableReleasesRead
	ImmutableReleasesUpdate
	RetentionRead
	RetentionUpdate
	ForkApprovalRead
	ForkApprovalUpdate
	PrivateForkRead
	PrivateForkUpdate
	ArchiveRead
	ArchiveUnarchive
	ArchiveUpdate
	PagesRead
	PagesUpdate
	EnvironmentsRead
	EnvironmentUpsert
	DependabotRead
	DependabotUpdate
	PagesCreate
	ActionsAccessRead
	ActionsAccessUpdate
	CacheStorageRead
	CacheStorageUpdate
	CacheRetentionRead
	CacheRetentionUpdate
	VariableUpsert
	CopilotReviewEffortUpdate
	CopilotInstructionsUpdate
	RunnersRead
	RunnerLabelsUpdate
	OIDCSubjectUpdate
	CopilotMCPUpdate
	PullRequestLimitsUpdate
	AgentVariableUpsert
	ActionsPoliciesRead
	ActionsEventPolicyCreate
	ActionsEventPolicyUpdate
	ActionsEventPolicyDelete
	EnvironmentVariableUpsert
	PullRequestExemptionsUpdate
	VariablesRead
	AgentVariablesRead
	EnvironmentVariablesRead
	VariableDelete
	AgentVariableDelete
	EnvironmentVariableDelete
	EnvironmentDelete
	CollaboratorsRead
	CollaboratorReconcile
	CollaboratorIdentityBinding
	CollaboratorCancelInvitation
	CollaboratorRemove
	AutolinksRead
	AutolinkCreate
	AutolinkDelete
	ClassicProtectionRead
	DeployKeysRead
	DeployKeyCreate
	DeployKeyDelete
	ClassicProtectionCreate
	ClassicProtectionUpdate
	ClassicProtectionDelete
)

type ApplyOperationStatus uint8

const (
	OperationNotExecuted ApplyOperationStatus = iota
	OperationCompleted
	OperationFailedRead
	OperationFailedUnknown
	OperationUnchanged
	OperationFailedLocal
)

// ApplyOperation records only what this execution established for one operation.
// A failed write has an unknown remote effect, rather than an unapplied status.
type ApplyOperation struct {
	RunnerID             int64
	DeployKeyFingerprint string
	ClassicPattern       string
	CollaboratorUsername string
	ActionsPolicyName    string
	VariableName         string
	Kind                 ApplyOperationKind
	SecurityField        string
	LabelName            string
	RulesetName          string
	EnvironmentName      string
	Status               ApplyOperationStatus
}

// ApplyResult retains known execution progress without claiming readback or rollback.
// Individual label operations become known only after a complete fresh read.
type ApplyResult struct {
	CollaboratorDeletions []settings.CollaboratorDeleteResult
	Collaborators         []CollaboratorResult
	Outcome               ApplyOutcome
	Target                repository.Repository
	Operations            []ApplyOperation
	CreatedIDs            []CreatedObject
	Notices               []string
}

type CreatedObject struct {
	Collection string
	Name       string
	ID         int64
}

// Completed preserves the CLI's managed-success decision, including matching labels.
func (result ApplyResult) Completed() bool {
	return result.Outcome == ApplyCompleted || result.Outcome == ApplyMatchingLabels
}

func (result *ApplyResult) failOperation(index int, status ApplyOperationStatus) {
	result.Operations[index].Status = status
	result.Outcome = ApplyFailed
	for _, operation := range result.Operations {
		if operation.Status == OperationCompleted && operation.Kind != RunnersRead && operation.Kind != DeployKeysRead && operation.Kind != AutolinksRead && operation.Kind != ClassicProtectionRead && operation.Kind != CollaboratorsRead && operation.Kind != CollaboratorIdentityBinding && operation.Kind != VariablesRead && operation.Kind != AgentVariablesRead && operation.Kind != EnvironmentVariablesRead && operation.Kind != ActionsPoliciesRead && operation.Kind != DependabotRead && operation.Kind != LabelsRead && operation.Kind != RulesetsRead && operation.Kind != EnvironmentsRead && operation.Kind != SecretScanningRead && operation.Kind != CodeQLConfigurationRead && operation.Kind != CodeQLDefaultSetupRead && operation.Kind != AIScanPullRequestsRead && operation.Kind != AIScanRepositoriesRead && operation.Kind != PrivateReportingRead && operation.Kind != ImmutableReleasesRead && operation.Kind != RetentionRead && operation.Kind != ActionsAccessRead && operation.Kind != CacheStorageRead && operation.Kind != CacheRetentionRead && operation.Kind != ForkApprovalRead && operation.Kind != PrivateForkRead && operation.Kind != ArchiveRead && operation.Kind != PagesRead {
			result.Outcome = ApplyPartialFailure
			break
		}
	}
}
