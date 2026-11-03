package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyDependabot(result *ApplyResult, repo repository.Repository, d settings.SecurityDeclaration, write Writer) (bool, error) {
	i := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: DependabotRead})
	o, err := write.ReadDependabot(repo)
	var changes []settings.SecurityChange
	if err == nil {
		_, changes, err = settings.DependabotTransitions(o, d)
	}
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return false, fmt.Errorf("preflight Dependabot (earlier writes remain applied): %w", err)
	}
	result.Operations[i].Status = OperationCompleted
	wrote := false
	for _, change := range changes {
		i := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: DependabotUpdate, SecurityField: change.Field()})
		outcome, err := write.Dependabot(repo, change)
		if err != nil {
			status := OperationFailedUnknown
			var before PreWriteReadFailure
			if errors.As(err, &before) {
				status = OperationFailedRead
			}
			result.failOperation(i, status)
			return wrote, fmt.Errorf("apply Dependabot (earlier writes may have applied): %w", err)
		}
		result.Operations[i].Status = OperationUnchanged
		if outcome.Wrote {
			wrote = true
			result.Operations[i].Status = OperationCompleted
		}
	}
	return wrote, nil
}
