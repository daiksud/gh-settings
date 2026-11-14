package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyRunnerPrune(result *ApplyResult, repo repository.Repository, desired []settings.RunnerLabels, prepared settings.RunnerPruneObservation, unarchived bool, write Writer) (bool, error) {
	candidates, err := prepared.Candidates(desired)
	if err != nil {
		return false, err
	}
	ids := []int64{}
	for _, v := range candidates {
		ids = append(ids, v.ID)
	}
	if unarchived {
		prepared = prepared.AfterUnarchive()
	}
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: RunnersRead})
	current, err := write.ReadRunnerPrune(repo)
	if err == nil {
		err = settings.VerifyRunnerPrune(prepared, current, ids, nil)
	}
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare runner prune%s: %w", environmentProgress(*result), err)
	}
	result.Operations[index].Status = OperationCompleted
	wrote := false
	expected := current
	for _, d := range desired {
		index = len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: RunnerLabelsUpdate, RunnerID: d.ID})
		changed, err := write.RunnerLabels(repo, d)
		if err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			return wrote, fmt.Errorf("apply runner labels%s: %w", environmentProgress(*result), err)
		}
		if !changed {
			result.Operations[index].Status = OperationUnchanged
			continue
		}
		result.Operations[index].Status = OperationCompleted
		wrote = true
		after, err := write.ReadRunnerPrune(repo)
		if err == nil {
			err = settings.VerifyRunnerPrune(expected, after, ids, &d)
		}
		if err != nil {
			index = len(result.Operations)
			result.Operations = append(result.Operations, ApplyOperation{Kind: RunnersRead})
			result.failOperation(index, OperationFailedRead)
			return wrote, fmt.Errorf("runner labels were updated but full inventory verification failed%s: %w", environmentProgress(*result), err)
		}
		expected = after
	}
	for _, id := range ids {
		index = len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: RunnerDelete, RunnerID: id})
		attempted, err := write.DeleteRunnerPrune(repo, expected, ids, id)
		if err != nil {
			status := OperationFailedRead
			if attempted {
				status = OperationFailedUnknown
			}
			result.failOperation(index, status)
			return wrote, fmt.Errorf("runner removal%s: %w", environmentProgress(*result), err)
		}
		if !attempted {
			result.failOperation(index, OperationFailedRead)
			return wrote, fmt.Errorf("runner removal lacked write evidence")
		}
		result.Operations[index].Status = OperationCompleted
		wrote = true
		expected, _ = expected.Without(id)
		// IDs already removed are no longer candidates for subsequent checks.
		ids = removeRunnerID(ids, id)
	}
	return wrote, nil
}
func removeRunnerID(ids []int64, id int64) []int64 {
	out := []int64{}
	for _, v := range ids {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}
