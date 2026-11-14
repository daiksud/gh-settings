package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyRunners(result *ApplyResult, repo repository.Repository, desired []settings.RunnerLabels, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: RunnersRead})
	observed, err := write.ReadRunners(repo, desired)
	if err == nil {
		err = settings.PrepareRunnerLabels(observed, desired)
	}
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare runner labels%s: %w", environmentProgress(*result), err)
	}
	result.Operations[index].Status = OperationCompleted
	wroteAny := false
	for _, d := range desired {
		index = len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: RunnerLabelsUpdate, RunnerID: d.ID})
		wrote, err := write.RunnerLabels(repo, d)
		if err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			return wroteAny, fmt.Errorf("apply runner labels%s: %w", environmentProgress(*result), err)
		}
		result.Operations[index].Status = OperationUnchanged
		if wrote {
			result.Operations[index].Status = OperationCompleted
			wroteAny = true
		}
	}
	return wroteAny, nil
}
