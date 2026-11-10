package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
)

func applyCopilotMCP(result *ApplyResult, repo repository.Repository, desired bool, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CopilotMCPUpdate})
	wrote, err := write.CopilotMCP(repo, desired)
	if err != nil {
		status := OperationFailedUnknown
		var pre PreWriteReadFailure
		if errors.As(err, &pre) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply Copilot MCP tools (earlier successful writes remain applied): %w", err)
	}
	result.Operations[index].Status = OperationUnchanged
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}
