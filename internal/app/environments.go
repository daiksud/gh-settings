package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func environmentNames(d *settings.EnvironmentDeclarations) []string {
	names := []string{}
	if d != nil {
		for _, policy := range d.Values() {
			names = append(names, policy.Name)
		}
	}
	return names
}

func applyEnvironments(result *ApplyResult, repo repository.Repository, desired *settings.EnvironmentDeclarations, write Writer) (bool, error) {
	i := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: EnvironmentsRead})
	observed, err := write.ReadEnvironments(repo, environmentNames(desired))
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return false, fmt.Errorf("read environments for apply (earlier writes remain applied): %w", err)
	}
	changes, err := settings.ReconcileEnvironments(observed, desired)
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return false, err
	}
	result.Operations[i].Status = OperationCompleted
	first := len(result.Operations)
	for _, change := range changes {
		result.Operations = append(result.Operations, ApplyOperation{Kind: EnvironmentUpsert, EnvironmentName: change.Name()})
	}
	wrote := false
	for offset, change := range changes {
		i := first + offset
		changed, err := write.Environment(repo, *change.After())
		if err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(i, status)
			return wrote, fmt.Errorf("upsert environment %q (earlier successful writes remain applied): %w", change.Name(), err)
		}
		result.Operations[i].Status = OperationCompleted
		if !changed {
			result.Operations[i].Status = OperationUnchanged
		}
		wrote = wrote || changed
	}
	return wrote, nil
}
