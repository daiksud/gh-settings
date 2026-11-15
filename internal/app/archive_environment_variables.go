package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Each selected parent and its values are read without writes. A shared parent
// with managed policy must be the same identity as the policy observation.
func verifyArchivedEnvironmentVariables(repo repository.Repository, desired []settings.EnvironmentVariableDeclaration, policyIDs map[string]int64, read func(repository.Repository, string, []string) (int64, []settings.VariableObservation, error)) error {
	for _, scope := range sortedEnvironmentVariables(desired) {
		values := settings.SortedVariables(scope.Variables)
		if len(values) == 0 {
			continue
		}
		if read == nil {
			return fmt.Errorf("verify archived environment variables: reader unavailable")
		}
		names := make([]string, len(values))
		for i, value := range values {
			names[i] = value.Name
		}
		id, observed, err := read(repo, scope.Environment, names)
		if err != nil || id <= 0 || len(observed) != len(values) {
			return fmt.Errorf("verify archived environment variables: incomplete observation")
		}
		if policyID, ok := policyIDs[settings.EnvironmentIdentity(scope.Environment)]; ok && policyID != id {
			return fmt.Errorf("verify archived environment variables: parent identity changed")
		}
		for i, value := range values {
			change, err := settings.CompareVariable(observed[i], value)
			if err != nil {
				return fmt.Errorf("verify archived environment variables: invalid observation")
			}
			if change != nil {
				return fmt.Errorf("archived environment variables differ")
			}
		}
	}
	return nil
}
