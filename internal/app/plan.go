package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// PlanResult identifies the selected repository and its declared-field comparison.
// A local no-op has no target and no managed fields.
type PlanResult struct {
	Target     repository.Repository
	Comparison settings.Comparison
}

// PlanError identifies a failed planning stage without changing its diagnostic.
type PlanError struct {
	Stage  string
	Target repository.Repository
	Err    error
}

func (e *PlanError) Error() string { return e.Err.Error() }
func (e *PlanError) Unwrap() error { return e.Err }

// TargetResolver supplies the repository belonging to the selected checkout.
type TargetResolver func() (repository.Repository, error)

// Plan validates locally, then reads and compares only when fields are declared.
func Plan(load DeclarationLoader, resolve TargetResolver, read Reader) (PlanResult, error) {
	desired, err := Validate(load)
	if err != nil {
		return PlanResult{}, &PlanError{Stage: "invalid-config", Err: err}
	}
	if desired.Empty() {
		return PlanResult{}, nil
	}
	repo, err := resolve()
	if err != nil {
		return PlanResult{}, &PlanError{Stage: "target-failure", Err: err}
	}
	var comparison settings.Comparison
	if !desired.Repository.BaseEmpty() {
		current, err := read.Repository(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison = settings.CompareRepository(current, desired.Repository)
	}
	if !desired.Repository.MergeMethodsEmpty() {
		current, err := read.MergeMethods(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		methods := settings.CompareMergeMethods(current, desired.Repository)
		comparison.Managed += methods.Managed
		comparison.Changes = append(comparison.Changes, methods.Changes...)
	}
	if desired.Topics.Names != nil {
		current, err := read.Topics(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		topics := settings.CompareTopics(current, desired.Topics)
		comparison.Managed += topics.Managed
		comparison.Changes = append(comparison.Changes, topics.Changes...)
	}
	if desired.Labels != nil {
		current, err := read.Labels(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		labels, err := settings.CompareLabels(current, desired.Labels)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += labels.Managed
		comparison.Labels = labels.Labels
	}
	return PlanResult{Target: repo, Comparison: comparison}, nil
}
