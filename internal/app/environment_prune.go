package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func checkEnvironmentPrune(result *ApplyResult, repo repository.Repository, desired settings.Declaration, initial settings.EnvironmentPruneObservation, write Writer) error {
	i := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: EnvironmentsRead})
	fresh, err := write.ReadEnvironmentPrune(repo, settings.EnvironmentRetainedNames(desired))
	if err == nil && !initial.Equal(fresh) {
		err = fmt.Errorf("Environment deletion candidates or observable state changed after global preflight")
	}
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return fmt.Errorf("recheck Environment prune: %w", err)
	}
	result.Operations[i].Status = OperationCompleted
	return nil
}

func applyEnvironmentPrune(result *ApplyResult, repo repository.Repository, prepared settings.EnvironmentPruneObservation, write Writer) (bool, error) {
	candidates, err := prepared.Candidates()
	if err != nil {
		return false, err
	}
	first := len(result.Operations)
	for _, candidate := range candidates {
		result.Operations = append(result.Operations, ApplyOperation{Kind: EnvironmentDelete, EnvironmentName: candidate.Name()})
	}
	wrote := false
	for offset, candidate := range candidates {
		i := first + offset
		if err := write.DeleteEnvironment(repo, candidate); err != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if errors.As(err, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(i, status)
			return wrote, fmt.Errorf("delete Environment %q: %w", candidate.Name(), err)
		}
		result.Operations[i].Status = OperationCompleted
		wrote = true
	}
	return wrote, nil
}

func planEnvironmentPrune(repo repository.Repository, desired settings.Declaration, read Reader, comparison *settings.Comparison) error {
	observed, err := read.EnvironmentPrune(repo, settings.EnvironmentRetainedNames(desired))
	if err != nil {
		return err
	}
	candidates, err := observed.Candidates()
	if err != nil {
		return err
	}
	if err := settings.CheckClassicDeploymentPruneConflict(desired.BranchProtectionRules, candidates); err != nil {
		return err
	}
	for _, candidate := range candidates {
		change, err := candidate.Change()
		if err != nil {
			return err
		}
		comparison.Environments = append(comparison.Environments, change)
	}
	slices.SortFunc(comparison.Environments, func(a, b settings.EnvironmentChange) int {
		return strings.Compare(settings.EnvironmentIdentity(a.Name()), settings.EnvironmentIdentity(b.Name()))
	})
	if desired.Environments.Len() == 0 {
		comparison.Managed++
	}
	return nil
}
