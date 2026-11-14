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
		return false, fmt.Errorf("read environments for apply: %w", err)
	}
	changes, err := settings.ReconcileEnvironments(observed, desired)
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return false, fmt.Errorf("reconcile environments for apply: %w", err)
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
			var effects EnvironmentWriteEffectsFailure
			confirmed := errors.As(err, &effects)
			if !confirmed && errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(i, status)
			if confirmed {
				result.Outcome = ApplyPartialFailure
			}
			return wrote || confirmed, fmt.Errorf("upsert environment %q: %w", change.Name(), err)
		}
		result.Operations[i].Status = OperationCompleted
		if !changed {
			result.Operations[i].Status = OperationUnchanged
		}
		wrote = wrote || changed
	}
	return wrote, nil
}
