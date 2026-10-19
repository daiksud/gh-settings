package app

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ApplyResult distinguishes successful completion from a local no-op.
// With an error, Applied:false does not establish the remote outcome.
type ApplyResult struct {
	Applied bool
}

// Apply keeps direct repository/topics writes; labels read their current collection before CRUD.
func Apply(load DeclarationLoader, resolve TargetResolver, write Writer) (ApplyResult, error) {
	desired, err := Validate(load)
	if err != nil {
		return ApplyResult{}, err
	}
	if desired.Empty() {
		return ApplyResult{}, nil
	}
	repo, err := resolve()
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
		for _, change := range comparison.Labels.Values() {
			var err error
			switch change.Operation() {
			case "create":
				err = write.CreateLabel(repo, change)
			case "update":
				err = write.UpdateLabel(repo, change)
			case "delete":
				err = write.DeleteLabel(repo, change)
			}
			if err != nil {
				return ApplyResult{}, fmt.Errorf("%s label %q (earlier successful writes remain applied): %w", change.Operation(), change.Name(), err)
			}
		}
	}
	return ApplyResult{Applied: true}, nil
}
