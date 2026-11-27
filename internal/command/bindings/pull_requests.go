package bindings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func updatePullRequestExemptions(repo repository.Repository, desired settings.PullRequestExemptionsDeclaration) (bool, error) {
	client, err := github.NewPullRequestLimitsClient(repo.Host)
	if err != nil {
		return false, &clientPreWriteFailure{err}
	}
	return client.UpdateExemptions(repo, desired)
}

// readArchivedCopilot supplies strict selected-toggle evidence for archived apply.
func readArchivedCopilot(repo repository.Repository, selected settings.ArchivedCopilotToggles) (settings.ArchivedCopilotToggles, error) {
	client, err := github.NewArchivedCopilotClient(repo.Host)
	if err != nil {
		return settings.ArchivedCopilotToggles{}, err
	}
	return client.Read(repo, selected)
}
