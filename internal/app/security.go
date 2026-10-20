package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applySecretScanning(result *ApplyResult, repo repository.Repository, d settings.SecurityDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: SecretScanningRead})
	observed, err := write.ReadSecretScanning(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read Secret scanning for apply (earlier writes remain applied): %w", err)
	}
	changes, err := settings.SecretScanningTransitions(observed, d)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("preflight Secret scanning (earlier writes remain applied): %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	wrote := false
	for _, change := range changes {
		index := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: SecretScanningUpdate, SecurityField: change.Field()})
		changed, err := write.SecretScanning(repo, change)
		if err != nil {
			status := OperationFailedUnknown
			var readFailure PreWriteReadFailure
			if errors.As(err, &readFailure) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			return wrote, fmt.Errorf("apply %s (earlier successful writes remain applied): %w", change.Field(), err)
		}
		if changed {
			result.Operations[index].Status = OperationCompleted
			wrote = true
		}
	}
	return wrote, nil
}
