package app

import (
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func applyCopilotReviewEffort(result *ApplyResult, repo repository.Repository, desired string, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CopilotReviewEffortUpdate})
	wrote, err := write.CopilotReviewEffort(repo, desired)
	if err != nil {
		result.failOperation(index, writeFailureStatus(err))
		return false, fmt.Errorf("apply Copilot review effort: %w", err)
	}
	result.Operations[index].Status = OperationUnchanged
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}
