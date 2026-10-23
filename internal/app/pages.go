package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyPages(result *ApplyResult, repo repository.Repository, d settings.PagesDeclaration, write Writer) (bool, error) {
	i := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: PagesRead})
	o, err := write.ReadPages(repo)
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return false, fmt.Errorf("read Pages for apply (earlier writes remain applied): %w", err)
	}
	_, _, err = settings.ComparePages(o, d)
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return false, fmt.Errorf("preflight Pages (earlier writes remain applied): %w", err)
	}
	result.Operations[i].Status = OperationCompleted
	i = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: PagesUpdate})
	wrote, err := write.Pages(repo, d)
	if err != nil {
		status := OperationFailedUnknown
		var pre PreWriteReadFailure
		if errors.As(err, &pre) {
			status = OperationFailedRead
		}
		result.failOperation(i, status)
		return false, fmt.Errorf("apply Pages (earlier successful writes remain applied): %w", err)
	}
	if wrote {
		result.Operations[i].Status = OperationCompleted
	}
	return wrote, nil
}
