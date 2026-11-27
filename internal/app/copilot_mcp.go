package app

import (
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func applyCopilotToggle(result *ApplyResult, repo repository.Repository, desired bool, kind ApplyOperationKind, field string, write func(repository.Repository, bool) (bool, error)) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: kind})
	wrote, err := write(repo, desired)
	if err != nil {
		result.failOperation(index, writeFailureStatus(err))
		return false, fmt.Errorf("apply %s: %w", field, err)
	}
	result.Operations[index].Status = OperationUnchanged
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}
