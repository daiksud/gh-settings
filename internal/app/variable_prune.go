package app

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type environmentVariablePrune struct {
	id             int64
	newEnvironment bool
	changes        []settings.VariableChange
}

func sortedEnvironmentVariables(d []settings.EnvironmentVariableDeclaration) []settings.EnvironmentVariableDeclaration {
	values := slices.Clone(d)
	slices.SortFunc(values, func(a, b settings.EnvironmentVariableDeclaration) int {
		return strings.Compare(settings.EnvironmentIdentity(a.Environment), settings.EnvironmentIdentity(b.Environment))
	})
	return values
}

func pruneEnvironmentWillBeCreated(repo repository.Repository, desired *settings.EnvironmentDeclarations, name string, read func(repository.Repository, []string) (settings.EnvironmentObservation, error)) (bool, error) {
	if desired == nil {
		return false, nil
	}
	for _, policy := range desired.Values() {
		if settings.EnvironmentIdentity(policy.Name) != settings.EnvironmentIdentity(name) {
			continue
		}
		selected, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{policy})
		if err != nil {
			return false, err
		}
		observed, err := read(repo, []string{policy.Name})
		if err != nil {
			return false, err
		}
		changes, err := settings.ReconcileEnvironments(observed, &selected)
		if err != nil {
			return false, err
		}
		for _, change := range changes {
			if change.Operation() == "create" {
				return true, nil
			}
		}
		return false, nil
	}
	return false, nil
}

func applyPrunedVariables(result *ApplyResult, repo repository.Repository, desired settings.VariableDeclarations, environment string, prepared environmentVariablePrune, readKind, upsertKind, deleteKind ApplyOperationKind, read func(repository.Repository) (settings.VariableInventory, error), upsert func(repository.Repository, settings.RepositoryVariable) (bool, error), remove func(repository.Repository, int64, settings.VariableChange) error) (bool, error) {
	readIndex := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: readKind, EnvironmentName: environment})
	inventory, err := read(repo)
	if err != nil {
		result.failOperation(readIndex, OperationFailedRead)
		return false, err
	}
	var id int64
	if environment != "" {
		id, err = inventory.EnvironmentID(environment)
		if err != nil || !prepared.newEnvironment && id != prepared.id {
			result.failOperation(readIndex, OperationFailedRead)
			return false, fmt.Errorf("Environment identity changed after prune preflight")
		}
	}
	deletions, err := settings.PrunedVariableChanges(inventory, &desired)
	if err != nil || !reflect.DeepEqual(prepared.changes, deletions) {
		result.failOperation(readIndex, OperationFailedRead)
		return false, fmt.Errorf("variable prune candidates unavailable or changed after preflight")
	}
	result.Operations[readIndex].Status = OperationCompleted
	values := settings.SortedVariables(desired)
	first := len(result.Operations)
	for _, value := range values {
		result.Operations = append(result.Operations, ApplyOperation{Kind: upsertKind, EnvironmentName: environment, VariableName: value.Name})
	}
	for _, change := range deletions {
		result.Operations = append(result.Operations, ApplyOperation{Kind: deleteKind, EnvironmentName: environment, VariableName: change.Name()})
	}
	wroteAny := false
	fail := func(index int, err error) (bool, error) {
		status := OperationFailedUnknown
		var pre PreWriteReadFailure
		if errors.As(err, &pre) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return wroteAny, fmt.Errorf("apply variable prune (earlier successful writes remain applied): %w", err)
	}
	for i, value := range values {
		wrote, err := upsert(repo, value)
		if err != nil {
			return fail(first+i, err)
		}
		result.Operations[first+i].Status = OperationUnchanged
		if wrote {
			result.Operations[first+i].Status = OperationCompleted
			wroteAny = true
		}
	}
	for i, change := range deletions {
		index := first + len(values) + i
		if err := remove(repo, id, change); err != nil {
			return fail(index, err)
		}
		result.Operations[index].Status = OperationCompleted
		wroteAny = true
	}
	return wroteAny, nil
}

func planVariablePrune(repo repository.Repository, desired settings.VariableDeclarations, read func(repository.Repository, string) (settings.VariableObservation, error), inventory settings.VariableInventory) ([]settings.VariableChange, error) {
	changes, err := planVariables(repo, desired, read)
	if err != nil {
		return nil, err
	}
	deletions, err := settings.PrunedVariableChanges(inventory, &desired)
	if err != nil {
		return nil, err
	}
	return append(changes, deletions...), nil
}
