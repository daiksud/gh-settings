package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// PlanResult identifies the selected repository and its declared-field comparison.
type PlanResult struct {
	Target     repository.Repository
	Comparison settings.RepositoryComparison
}

// Plan validates locally, resolves the target, then reads and compares its settings.
func Plan(readRepository func(repository.Repository) (settings.RepositoryState, error)) (PlanResult, error) {
	desired, err := Validate()
	if err != nil {
		return PlanResult{}, err
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
