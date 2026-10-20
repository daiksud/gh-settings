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
	WorkflowPermissions func(repository.Repository) (settings.WorkflowPermissionsState, error)
	Actions             func(repository.Repository, bool) (settings.ActionsPolicyState, error)
	ObserveRepository   func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error)
	Topics              func(repository.Repository) ([]string, error)
	Labels              func(repository.Repository) (settings.LabelObservation, error)
	Rulesets            func(repository.Repository) (settings.RulesetObservation, error)
}

// Writer sends each resource independently; successful earlier writes are not rolled back.
type Writer struct {
	WorkflowPermissions func(repository.Repository, settings.ActionsPolicyDeclaration) error
	Actions             func(repository.Repository, settings.ActionsPolicyDeclaration) error
	Repository          func(repository.Repository, settings.RepositoryDeclaration) error
	Topics              func(repository.Repository, settings.TopicsDeclaration) error
	ReadLabels          func(repository.Repository) (settings.LabelObservation, error)
	CreateLabel         func(repository.Repository, settings.LabelChange) error
	UpdateLabel         func(repository.Repository, settings.LabelChange) error
	DeleteLabel         func(repository.Repository, settings.LabelChange) error
	ReadRulesets        func(repository.Repository) (settings.RulesetObservation, error)
	Ruleset             func(repository.Repository, settings.RulesetChange) error
}
