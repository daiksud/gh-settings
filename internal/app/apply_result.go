package app

import "github.com/cli/go-gh/v2/pkg/repository"

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
	PrivateReportingRead
	PrivateReportingUpdate
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
)

type ApplyOperationStatus uint8

const (
	OperationNotExecuted ApplyOperationStatus = iota
	OperationCompleted
	OperationFailedRead
	OperationFailedUnknown
)

// ApplyOperation records only what this execution established for one operation.
// A failed write has an unknown remote effect, rather than an unapplied status.
type ApplyOperation struct {
	Kind          ApplyOperationKind
	SecurityField string
	LabelName     string
	RulesetName   string
	Status        ApplyOperationStatus
}

// ApplyResult retains known execution progress without claiming readback or rollback.
// Individual label operations become known only after a complete fresh read.
type ApplyResult struct {
	Outcome    ApplyOutcome
	Target     repository.Repository
	Operations []ApplyOperation
}

// Completed preserves the CLI's managed-success decision, including matching labels.
func (result ApplyResult) Completed() bool {
	return result.Outcome == ApplyCompleted || result.Outcome == ApplyMatchingLabels
}

func (result *ApplyResult) failOperation(index int, status ApplyOperationStatus) {
	result.Operations[index].Status = status
	result.Outcome = ApplyFailed
	for _, operation := range result.Operations {
		if operation.Status == OperationCompleted && operation.Kind != LabelsRead && operation.Kind != RulesetsRead && operation.Kind != SecretScanningRead && operation.Kind != PrivateReportingRead && operation.Kind != RetentionRead && operation.Kind != ForkApprovalRead && operation.Kind != PrivateForkRead && operation.Kind != ArchiveRead && operation.Kind != PagesRead {
			result.Outcome = ApplyPartialFailure
			break
		}
	}
}
