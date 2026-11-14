package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyForkApproval(result *ApplyResult, repo repository.Repository, d settings.ActionsPolicyDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: ForkApprovalRead})
	observed, err := write.ReadForkApproval(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read fork approval for apply: %w", err)
	}
	_, changes, err := settings.CompareForkApproval(observed, d)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("preflight fork approval: %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	if len(changes) == 0 {
		return false, nil
	}
	index = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: ForkApprovalUpdate})
	wrote, err := write.ForkApproval(repo, *d.ForkApprovalPolicy)
	if err != nil {
		status := OperationFailedUnknown
		var readFailure PreWriteReadFailure
		if errors.As(err, &readFailure) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply fork approval: %w", err)
	}
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}

func applyPrivateFork(result *ApplyResult, repo repository.Repository, d settings.ActionsPolicyDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: PrivateForkRead})
	observed, err := write.ReadPrivateFork(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read private fork for apply: %w", err)
	}
	_, changes, err := settings.ComparePrivateFork(observed, d.PrivateFork)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("preflight private fork: %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	if len(changes) == 0 {
		return false, nil
	}
	index = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: PrivateForkUpdate})
	wrote, err := write.PrivateFork(repo, d.PrivateFork)
	if err != nil {
		status := OperationFailedUnknown
		var readFailure PreWriteReadFailure
		if errors.As(err, &readFailure) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply private fork: %w", err)
	}
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}
