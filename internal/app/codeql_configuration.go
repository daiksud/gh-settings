package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyCodeQLConfiguration(result *ApplyResult, repo repository.Repository, d settings.CodeQLConfigurationDeclaration, w Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CodeQLConfigurationRead})
	o, err := w.ReadCodeQLConfiguration(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read CodeQL configuration: %w", err)
	}
	_, changes, err := settings.CompareCodeQLConfiguration(o, d)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, err
	}
	result.Operations[index].Status = OperationCompleted
	if len(changes) == 0 {
		return false, nil
	}
	result.Notices = append(result.Notices, "Changing CodeQL configuration may start validation/analysis and consume Actions resources; configuration readback does not confirm scan completion.")
	index = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CodeQLConfigurationUpdate})
	changed, err := w.CodeQLConfiguration(repo, d)
	if err != nil {
		status := OperationFailedUnknown
		var pre PreWriteReadFailure
		if errors.As(err, &pre) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply CodeQL configuration: %w", err)
	}
	if changed {
		result.Operations[index].Status = OperationCompleted
	}
	return changed, nil
}
