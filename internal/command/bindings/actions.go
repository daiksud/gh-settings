package bindings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func writeActionsPolicy(repo repository.Repository, desired settings.ActionsPolicyDeclaration) (bool, error) {
	client, err := github.NewActionsPolicyClient(repo.Host)
	if err != nil {
		return false, err
	}
	client.NormalizationWarning = warnNormalization
	return client.ApplyActionsPolicy(repo, desired)
}

func writeWorkflowPermissions(repo repository.Repository, desired settings.ActionsPolicyDeclaration) (bool, error) {
	client, err := github.NewActionsPolicyClient(repo.Host)
	if err != nil {
		return false, err
	}
	client.NormalizationWarning = warnNormalization
	return client.ApplyWorkflowPermissions(repo, desired)
}

func readForkApproval(repo repository.Repository) (settings.ForkApprovalObservation, error) {
	c, err := github.NewForkWorkflowsClient(repo.Host)
	if err != nil {
		return settings.ForkApprovalObservation{}, err
	}
	return c.ReadApproval(repo)
}

func readForkApprovalForApply(repo repository.Repository) (settings.ForkApprovalObservation, error) {
	observed, err := readForkApproval(repo)
	if err != nil {
		return settings.ForkApprovalObservation{}, err
	}
	if _, err := observed.Value(); err != nil {
		return settings.ForkApprovalObservation{}, err
	}
	if record := observed.Normalization(); record.Path != "" {
		warnNormalization(record)
	}
	return observed, nil
}

func writeForkApproval(repo repository.Repository, v string) (bool, error) {
	c, err := github.NewForkWorkflowsClient(repo.Host)
	if err != nil {
		return false, err
	}
	c.NormalizationWarning = warnNormalization
	return c.UpdateApproval(repo, v)
}

func verifyArchivedRunners(repo repository.Repository, desired []settings.RunnerLabels, archive settings.ArchiveState) error {
	client, err := github.NewRunnersClient(repo.Host)
	if err != nil {
		return err
	}
	return client.VerifyArchivedMatching(repo, desired, archive)
}

func deleteRunnerPrune(repo repository.Repository, expected settings.RunnerPruneObservation, candidates []int64, id int64) (bool, error) {
	c, e := github.NewRunnersClient(repo.Host)
	if e != nil {
		return false, e
	}
	return c.DeletePrune(repo, expected, candidates, id)
}
