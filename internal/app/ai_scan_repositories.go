package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyAIScanRepositories(result *ApplyResult, repo repository.Repository, d settings.SecurityDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: AIScanRepositoriesRead})
	observed, err := write.ReadAIScanRepositories(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read AI Scan for repositories for apply: %w", err)
	}
	_, changes, err := settings.CompareAIScanRepositories(observed, d)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("compare AI Scan for repositories: %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	if len(changes) == 0 {
		return false, nil
	}
	result.Notices = append(result.Notices, "AI Scan for repositories permits separately requested scans that may consume AI credits; this apply confirms configuration only and does not start a scan.")
	index = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: AIScanRepositoriesUpdate, SecurityField: changes[0].Field()})
	changed, err := write.AIScanRepositories(repo, changes[0])
	if err != nil {
		status := OperationFailedUnknown
		var readFailure PreWriteReadFailure
		if errors.As(err, &readFailure) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply AI Scan for repositories: %w", err)
	}
	result.Operations[index].Status = OperationUnchanged
	if changed {
		result.Operations[index].Status = OperationCompleted
	}
	return changed, nil
}
