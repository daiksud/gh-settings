package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyVariables(result *ApplyResult, repo repository.Repository, desired settings.VariableDeclarations, kind ApplyOperationKind, write func(repository.Repository, settings.RepositoryVariable) (bool, error)) (bool, error) {
	values := settings.SortedVariables(desired)
	first := len(result.Operations)
	for _, value := range values {
		result.Operations = append(result.Operations, ApplyOperation{Kind: kind, VariableName: value.Name})
	}
	wroteAny := false
	for i, value := range values {
		index := first + i
		wrote, err := write(repo, value)
		if err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			return wroteAny, fmt.Errorf("apply repository variable %q (earlier successful writes remain applied): %w", value.Name, err)
		}
		result.Operations[index].Status = OperationUnchanged
		if wrote {
			result.Operations[index].Status = OperationCompleted
			wroteAny = true
		}
	}
	return wroteAny, nil
}

func planVariables(repo repository.Repository, desired settings.VariableDeclarations, read func(repository.Repository, string) (settings.VariableObservation, error)) ([]settings.VariableChange, error) {
	var changes []settings.VariableChange
	for _, value := range settings.SortedVariables(desired) {
		observed, err := read(repo, value.Name)
		if err != nil {
			return nil, err
		}
		change, err := settings.CompareVariable(observed, value)
		if err != nil {
			return nil, err
		}
		if change != nil {
			changes = append(changes, *change)
		}
	}
	return changes, nil
}
