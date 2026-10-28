package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyImmutableReleases(result *ApplyResult, repo repository.Repository, d settings.SecurityDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: ImmutableReleasesRead})
	observed, err := write.ReadImmutableReleases(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read immutable releases for apply (earlier writes remain applied): %w", err)
	}
	_, changes, err := settings.CompareImmutableReleases(observed, d)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("compare immutable releases (earlier writes remain applied): %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	if len(changes) == 0 {
		return false, nil
	}
	index = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: ImmutableReleasesUpdate, SecurityField: changes[0].Field()})
	changed, err := write.ImmutableReleases(repo, changes[0])
	if err != nil {
		status := OperationFailedUnknown
		var readFailure PreWriteReadFailure
		if errors.As(err, &readFailure) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply immutable releases (earlier successful writes remain applied): %w", err)
	}
	if changed {
		result.Operations[index].Status = OperationCompleted
	} else {
		result.Operations[index].Status = OperationUnchanged
	}
	return changed, nil
}
