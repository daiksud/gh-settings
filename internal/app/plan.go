package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// TargetResolver supplies the repository belonging to the selected checkout.
type TargetResolver func() (repository.Repository, error)

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
	if desired.Repository.Archived != nil {
		state, err := read.Archive(repo)
		if err == nil {
			err = state.Validate(repo)
		}
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed++
		if state.Archived != *desired.Repository.Archived {
			field, _ := settings.ParseFieldID("repository.archived")
			change, _ := settings.NewBoolChange(field, state.Archived, *desired.Repository.Archived)
			comparison.Changes = append(comparison.Changes, change)
		}
	}
	other := desired.Repository
	other.Archived = nil
	if fields := settings.DeclaredRepositoryFields(other); len(fields) > 0 {
		observed, err := read.ObserveRepository(repo, fields)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		otherComparison, err := settings.CompareRepositoryObservation(observed, other)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += otherComparison.Managed
		comparison.Changes = append(otherComparison.Changes, comparison.Changes...)
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
	if desired.Rulesets != nil {
		observed, err := read.Rulesets(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		changes, err := settings.ReconcileRulesets(observed, desired.Rulesets)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.Rulesets = changes
	}
	if !desired.Actions.ExecutionEmpty() {
		observed, err := read.Actions(repo, !desired.Actions.SelectedActions.Empty())
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareActionsPolicy(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = changes
	}
	if !desired.Actions.WorkflowPermissionsEmpty() {
		observed, err := read.WorkflowPermissions(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareWorkflowPermissions(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if !desired.Actions.RetentionEmpty() {
		observed, err := read.Retention(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareRetention(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if desired.Actions.ForkApprovalPolicy != nil {
		o, err := read.ForkApproval(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareForkApproval(o, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if !desired.Actions.PrivateFork.Empty() {
		o, err := read.PrivateFork(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		managed, changes, err := settings.ComparePrivateFork(o, desired.Actions.PrivateFork)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if !desired.Security.SecretScanningEmpty() {
		observed, err := read.SecretScanning(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		if _, err := settings.SecretScanningTransitions(observed, desired.Security); err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareSecretScanning(observed, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = changes
	}

	if desired.Security.PrivateVulnerabilityReporting != nil {
		observed, err := read.PrivateReporting(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		managed, changes, err := settings.ComparePrivateReporting(observed, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = append(comparison.Security, changes...)
	}

	return PlanResult{Target: repo, Comparison: comparison}, nil
}
