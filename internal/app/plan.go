package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// PlanResult identifies the selected repository and its declared-field comparison.
// A local no-op has no target and no managed fields.
type PlanResult struct {
	Target     repository.Repository
	Comparison settings.RepositoryComparison
}

// Plan validates locally, then reads and compares only when fields are declared.
func Plan(readRepository func(repository.Repository) (settings.RepositoryState, error)) (PlanResult, error) {
	desired, err := Validate()
	if err != nil {
		return PlanResult{}, err
	}
	if desired.Empty() {
		return PlanResult{}, nil
	}
	repo, err := workspace.CurrentRepository()
	if err != nil {
		return PlanResult{}, err
	}
	current, err := readRepository(repo)
	if err != nil {
		return PlanResult{}, err
	}
	return PlanResult{Target: repo, Comparison: settings.CompareRepository(current, desired)}, nil
}
