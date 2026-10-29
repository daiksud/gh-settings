package main

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func writeRepositoryWithWriters(repo repository.Repository, desired settings.RepositoryDeclaration, restWrite, accessWrite func(repository.Repository, settings.RepositoryDeclaration) error) error {
	restDesired := desired
	restDesired.Archived = nil
	restDesired.HasPullRequests = nil
	restDesired.PullRequestCreationPolicy = nil
	restDesired.IssueCreationPolicy = nil
	restDesired.HasSponsorships = nil
	if !restDesired.Empty() {
		if err := restWrite(repo, restDesired); err != nil {
			return err
		}
	}
	if desired.RepositoryAccessEmpty() {
		return nil
	}
	err := accessWrite(repo, desired)
	if err != nil && !restDesired.Empty() {
		err = &repositoryWriteEffectsFailure{cause: err}
		if desired.IssueCreationPolicy != nil || desired.HasSponsorships != nil {
			return fmt.Errorf("apply repository access (earlier repository writes remain applied): %w", err)
		}
		return fmt.Errorf("apply PR access (earlier repository writes remain applied): %w", err)
	}
	return err
}

// Earlier REST work completed in this composite operation. Preserve the inner
// access failure while preventing its prerequisite marker from erasing progress.
type repositoryWriteEffectsFailure struct{ cause error }

func (failure *repositoryWriteEffectsFailure) Error() string           { return failure.cause.Error() }
func (failure *repositoryWriteEffectsFailure) Unwrap() error           { return failure.cause }
func (*repositoryWriteEffectsFailure) RepositoryWritesMayHaveApplied() {}
