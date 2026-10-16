package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// ApplyResult distinguishes successful completion from a local no-op.
// With an error, Applied:false does not establish the remote outcome.
type ApplyResult struct {
	Applied bool
}

// Apply validates locally and sends the declaration directly without a preliminary read.
func Apply(writeRepository func(repository.Repository, settings.RepositoryDeclaration) error) (ApplyResult, error) {
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
	if err := writeRepository(repo, desired); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Applied: true}, nil
}
