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

// Reader defers each API client until the operation needs that resource.
type Reader struct {
	Pages               func(repository.Repository) (settings.PagesObservation, error)
	Archive             func(repository.Repository) (settings.ArchiveState, error)
	ForkApproval        func(repository.Repository) (settings.ForkApprovalObservation, error)
	PrivateFork         func(repository.Repository) (settings.PrivateForkObservation, error)
	Retention           func(repository.Repository) (settings.RetentionObservation, error)
	PrivateReporting    func(repository.Repository) (settings.PrivateReportingObservation, error)
	SecretScanning      func(repository.Repository) (settings.SecretScanningObservation, error)
	WorkflowPermissions func(repository.Repository) (settings.WorkflowPermissionsState, error)
	Actions             func(repository.Repository, bool) (settings.ActionsPolicyState, error)
	ObserveRepository   func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error)
	Topics              func(repository.Repository) ([]string, error)
	Labels              func(repository.Repository) (settings.LabelObservation, error)
	Rulesets            func(repository.Repository) (settings.RulesetObservation, error)
	Environments        func(repository.Repository, []string) (settings.EnvironmentObservation, error)
}

// Writer sends each resource independently; successful earlier writes are not rolled back.
type Writer struct {
	ReadPages            func(repository.Repository) (settings.PagesObservation, error)
	Pages                func(repository.Repository, settings.PagesDeclaration) (bool, error)
	ReadArchive          func(repository.Repository) (settings.ArchiveState, error)
	Archive              func(repository.Repository, settings.ArchiveState, bool) error
	ReadForkApproval     func(repository.Repository) (settings.ForkApprovalObservation, error)
	ForkApproval         func(repository.Repository, string) (bool, error)
	ReadPrivateFork      func(repository.Repository) (settings.PrivateForkObservation, error)
	PrivateFork          func(repository.Repository, settings.PrivateForkDeclaration) (bool, error)
	ReadRetention        func(repository.Repository) (settings.RetentionObservation, error)
	Retention            func(repository.Repository, int) (bool, error)
	ReadPrivateReporting func(repository.Repository) (settings.PrivateReportingObservation, error)
	PrivateReporting     func(repository.Repository, settings.SecurityChange) (bool, error)
	ReadSecretScanning   func(repository.Repository) (settings.SecretScanningObservation, error)
	SecretScanning       func(repository.Repository, settings.SecurityChange) (bool, error)
	WorkflowPermissions  func(repository.Repository, settings.ActionsPolicyDeclaration) error
	Actions              func(repository.Repository, settings.ActionsPolicyDeclaration) error
	Repository           func(repository.Repository, settings.RepositoryDeclaration) error
	Topics               func(repository.Repository, settings.TopicsDeclaration) error
	ReadLabels           func(repository.Repository) (settings.LabelObservation, error)
	CreateLabel          func(repository.Repository, settings.LabelChange) (int64, error)
	CreatedID            func(string, string, int64) error
	UpdateLabel          func(repository.Repository, settings.LabelChange) error
	DeleteLabel          func(repository.Repository, settings.LabelChange) error
	ReadRulesets         func(repository.Repository) (settings.RulesetObservation, error)
	Ruleset              func(repository.Repository, settings.RulesetChange) (int64, error)
	ReadEnvironments     func(repository.Repository, []string) (settings.EnvironmentObservation, error)
	Environment          func(repository.Repository, settings.EnvironmentDeclaration) (bool, error)
}
