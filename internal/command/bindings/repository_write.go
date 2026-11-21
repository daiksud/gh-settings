package bindings

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func writeRepository(repo repository.Repository, desired settings.RepositoryDeclaration) (bool, error) {
	return writeRepositoryWithWriters(repo, desired,
		func(repo repository.Repository, restDesired settings.RepositoryDeclaration) (bool, error) {
			client, err := github.NewWriteClient(repo.Host)
			if err != nil {
				return false, err
			}
			client.CommitMessageNormalizationWarning = warnCommitMessageNormalization
			return client.ApplyRepository(repo, restDesired)
		},
		func(repo repository.Repository, accessDesired settings.RepositoryDeclaration) (bool, error) {
			client, err := github.NewRepositoryAccessClient(repo.Host)
			if err != nil {
				return false, err
			}
			client.NormalizationWarning = warnNormalization
			return client.ApplyRepositoryAccess(repo, accessDesired)
		})
}

func writeArchive(repo repository.Repository, previous settings.ArchiveState, archived bool) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.UpdateArchive(repo, previous, archived)
}

func writeRepositoryWithWriters(repo repository.Repository, desired settings.RepositoryDeclaration, restWrite func(repository.Repository, settings.RepositoryDeclaration) (bool, error), accessWrite func(repository.Repository, settings.RepositoryDeclaration) (bool, error)) (bool, error) {
	restDesired := desired
	restDesired.Archived = nil
	restDesired.HasPullRequests = nil
	restDesired.PullRequestCreationPolicy = nil
	restDesired.IssueCreationPolicy = nil
	restDesired.HasSponsorships = nil
	var restWrote bool
	if !restDesired.Empty() {
		var err error
		restWrote, err = restWrite(repo, restDesired)
		if err != nil {
			return restWrote, err
		}
	}
	if desired.RepositoryAccessEmpty() {
		return restWrote, nil
	}
	accessWrote, err := accessWrite(repo, desired)
	if err != nil && restWrote {
		err = &repositoryWriteEffectsFailure{cause: err}
		if desired.IssueCreationPolicy != nil || desired.HasSponsorships != nil {
			return true, fmt.Errorf("apply repository access: %w", err)
		}
		return true, fmt.Errorf("apply PR access: %w", err)
	}
	if err != nil {
		return accessWrote, err
	}
	return restWrote || accessWrote, nil
}

// Earlier REST work completed in this composite operation. Preserve the inner
// access failure while preventing its prerequisite marker from erasing progress.
type repositoryWriteEffectsFailure struct{ cause error }

func (failure *repositoryWriteEffectsFailure) Error() string {
	return "earlier repository writes remain applied: " + failure.cause.Error()
}

func (failure *repositoryWriteEffectsFailure) Unwrap() error           { return failure.cause }
func (*repositoryWriteEffectsFailure) RepositoryWritesMayHaveApplied() {}
func (*repositoryWriteEffectsFailure) ConfirmedWriteProgressReported() {}

func updateLabel(repo repository.Repository, change settings.LabelChange) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.UpdateLabel(repo, change)
}

func deleteLabel(repo repository.Repository, change settings.LabelChange) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.DeleteLabel(repo, change)
}
