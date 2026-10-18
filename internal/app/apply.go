package app

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// ApplyResult distinguishes successful completion from a local no-op.
// With an error, Applied:false does not establish the remote outcome.
type ApplyResult struct {
	Applied bool
}

// Apply keeps direct repository/topics writes; labels read their current collection before CRUD.
func Apply(write Writer) (ApplyResult, error) {
	desired, err := Validate()
	if err != nil {
		return ApplyResult{}, err
	}
	if desired.Empty() {
		return ApplyResult{}, nil
	}
	repo, err := workspace.CurrentRepository()
	if err != nil {
		return ApplyResult{}, err
	}
	if !desired.Repository.Empty() {
		if err := write.Repository(repo, desired.Repository); err != nil {
			return ApplyResult{}, err
		}
	}
	if desired.Topics.Names != nil {
		if err := write.Topics(repo, desired.Topics); err != nil {
			return ApplyResult{}, err
		}
	}
	if desired.Labels != nil {
		current, err := write.ReadLabels(repo)
		if err != nil {
			return ApplyResult{}, fmt.Errorf("read labels for apply (earlier resource writes remain applied): %w", err)
		}
		comparison, err := settings.CompareLabels(current, desired.Labels)
		if err != nil {
			return ApplyResult{}, fmt.Errorf("compare labels for apply (earlier resource writes remain applied): %w", err)
		}
		if len(comparison.Labels) > 0 {
			if err := write.Labels(repo, comparison.Labels); err != nil {
				return ApplyResult{}, err
			}
		}
	}
	return ApplyResult{Applied: true}, nil
}
