package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyPullRequestExemptions(result *ApplyResult, repo repository.Repository, desired settings.PullRequestExemptionsDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: PullRequestExemptionsUpdate})
	wrote, err := write.PullRequestExemptions(repo, desired)
	if err != nil {
		status := OperationFailedUnknown
		var pre PreWriteReadFailure
		if !wrote && errors.As(err, &pre) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		// An authoritative replacement can confirm removals before a later add or
		// confirmation fails. Preserve that progress even within this one operation.
		if wrote {
			result.Outcome = ApplyPartialFailure
		}
		return wrote, fmt.Errorf("apply pull request limit exemptions (earlier successful writes remain applied): %w", err)
	}
	result.Operations[index].Status = OperationUnchanged
	if wrote {
		result.Operations[index].Status = OperationCompleted
	}
	return wrote, nil
}
