package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyCodeQLDefaultSetup(result *ApplyResult, repo repository.Repository, d settings.SecurityDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CodeQLDefaultSetupRead})
	observed, err := write.ReadCodeQLDefaultSetup(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read CodeQL default setup for apply: %w", err)
	}
	_, changes, err := settings.CompareCodeQLDefaultSetup(observed, d)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("compare CodeQL default setup: %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	if len(changes) == 0 {
		return false, nil
	}
	result.Notices = append(result.Notices, "Changing CodeQL default setup may start validation/analysis and consume Actions resources; successful readback confirms configuration only.")
	index = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CodeQLDefaultSetupUpdate, SecurityField: changes[0].Field()})
	changed, err := write.CodeQLDefaultSetup(repo, changes[0])
	if err != nil {
		status := OperationFailedUnknown
		var readFailure PreWriteReadFailure
		if errors.As(err, &readFailure) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply CodeQL default setup: %w", err)
	}
	if changed {
		result.Operations[index].Status = OperationCompleted
	}
	return changed, nil
}
