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
				return wroteAny, fmt.Errorf("apply environment %q variable %q: %w", selected.Environment, value.Name, err)
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
