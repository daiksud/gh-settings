package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyEnvironmentVariables(result *ApplyResult, repo repository.Repository, declarations []settings.EnvironmentVariableDeclaration, write func(repository.Repository, string, settings.RepositoryVariable) (bool, error)) (bool, error) {
	declarations = slices.Clone(declarations)
	slices.SortFunc(declarations, func(a, b settings.EnvironmentVariableDeclaration) int {
		return strings.Compare(settings.EnvironmentIdentity(a.Environment), settings.EnvironmentIdentity(b.Environment))
	})
	// Queue the entire selected family before execution so later failures preserve
	// both confirmed progress and the names that were never attempted.
	first := len(result.Operations)
	for _, selected := range declarations {
		for _, value := range settings.SortedVariables(selected.Variables) {
			result.Operations = append(result.Operations, ApplyOperation{Kind: EnvironmentVariableUpsert, EnvironmentName: selected.Environment, VariableName: value.Name})
		}
	}
	wroteAny := false
	index := first
	for _, selected := range declarations {
		for _, value := range settings.SortedVariables(selected.Variables) {
			wrote, err := write(repo, selected.Environment, value)
			if err != nil {
				status := OperationFailedUnknown
				var pre PreWriteReadFailure
				if errors.As(err, &pre) {
					status = OperationFailedRead
				}
				result.failOperation(index, status)
				return wroteAny, fmt.Errorf("apply environment %q variable %q (earlier successful writes remain applied): %w", selected.Environment, value.Name, err)
			}
			result.Operations[index].Status = OperationUnchanged
			if wrote {
				result.Operations[index].Status = OperationCompleted
				wroteAny = true
			}
			index++
		}
	}
	return wroteAny, nil
}

func exportEnvironmentVariables(repo repository.Repository, selections []settings.EnvironmentVariableSelection, read func(repository.Repository, string, string) (settings.VariableObservation, error)) ([]settings.EnvironmentVariableDeclaration, error) {
	var result []settings.EnvironmentVariableDeclaration
	for _, selected := range selections {
		values, err := exportVariables(repo, []string{selected.Name}, func(repo repository.Repository, name string) (settings.VariableObservation, error) {
			return read(repo, selected.Environment, name)
		})
		if err != nil {
			return nil, fmt.Errorf("environment %q: %w", selected.Environment, err)
		}
		index := slices.IndexFunc(result, func(v settings.EnvironmentVariableDeclaration) bool {
			return settings.EnvironmentIdentity(v.Environment) == settings.EnvironmentIdentity(selected.Environment)
		})
		if index < 0 {
			result = append(result, settings.EnvironmentVariableDeclaration{Environment: selected.Environment, Variables: *values})
			continue
		}
		merged, err := settings.NewVariableDeclarations(append(result[index].Variables.Values(), values.Values()...))
		if err != nil {
			return nil, err
		}
		result[index].Variables = merged
	}
	slices.SortFunc(result, func(a, b settings.EnvironmentVariableDeclaration) int {
		return strings.Compare(settings.EnvironmentIdentity(a.Environment), settings.EnvironmentIdentity(b.Environment))
	})
	return result, nil
}
