package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyAIScanPullRequests(result *ApplyResult, repo repository.Repository, d settings.SecurityDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: AIScanPullRequestsRead})
	observed, err := write.ReadAIScanPullRequests(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read AI Scan for pull requests for apply: %w", err)
	}
	_, changes, err := settings.CompareAIScanPullRequests(observed, d)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("compare AI Scan for pull requests: %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	if len(changes) == 0 {
		return false, nil
	}
	result.Notices = append(result.Notices, "AI Scan for pull requests permits automated AI processing and may consume AI credits; successful readback confirms enablement only.")
	index = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: AIScanPullRequestsUpdate, SecurityField: changes[0].Field()})
	changed, err := write.AIScanPullRequests(repo, changes[0])
	if err != nil {
		status := OperationFailedUnknown
		var readFailure PreWriteReadFailure
		if errors.As(err, &readFailure) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply AI Scan for pull requests: %w", err)
	}
	if changed {
		result.Operations[index].Status = OperationCompleted
	}
	return changed, nil
}
