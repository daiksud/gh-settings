package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyOIDCSubject(result *ApplyResult, repo repository.Repository, desired settings.OIDCSubjectTemplate, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: OIDCSubjectUpdate})
	wrote, err := write.OIDCSubject(repo, desired)
	if err != nil {
		result.failOperation(index, writeFailureStatus(err))
		return false, fmt.Errorf("apply OIDC subject template: %w", err)
	}
	result.Operations[index].Status = OperationUnchanged
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}
