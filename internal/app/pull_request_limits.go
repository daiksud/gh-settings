package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyPullRequestLimits(result *ApplyResult, repo repository.Repository, desired settings.PullRequestLimitsDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: PullRequestLimitsUpdate})
	wrote, err := write.PullRequestLimits(repo, desired)
	if err != nil {
		result.failOperation(index, writeFailureStatus(err))
		return false, fmt.Errorf("apply pull request limits: %w", err)
	}
	result.Operations[index].Status = OperationUnchanged
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}
