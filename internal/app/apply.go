package app

import "github.com/daiksud/gh-settings/internal/workspace"

// ApplyResult distinguishes successful completion from a local no-op.
// With an error, Applied:false does not establish the remote outcome.
type ApplyResult struct {
	Applied bool
}

// Apply validates locally and sends the declaration directly without a preliminary read.
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
	return ApplyResult{Applied: true}, nil
}
