package app

import (
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func applyCopilotInstructions(result *ApplyResult, repo repository.Repository, desired bool, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CopilotInstructionsUpdate})
	wrote, err := write.CopilotInstructions(repo, desired)
	if err != nil {
		result.failOperation(index, writeFailureStatus(err))
		return false, fmt.Errorf("apply Copilot custom instructions: %w", err)
	}
	result.Operations[index].Status = OperationUnchanged
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}
