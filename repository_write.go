package main

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

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
			return true, fmt.Errorf("apply repository access (earlier repository writes remain applied): %w", err)
		}
		return true, fmt.Errorf("apply PR access (earlier repository writes remain applied): %w", err)
	}
	if err != nil {
		return accessWrote, err
	}
	return restWrote || accessWrote, nil
}

// Earlier REST work completed in this composite operation. Preserve the inner
// access failure while preventing its prerequisite marker from erasing progress.
type repositoryWriteEffectsFailure struct{ cause error }

func (failure *repositoryWriteEffectsFailure) Error() string           { return failure.cause.Error() }
func (failure *repositoryWriteEffectsFailure) Unwrap() error           { return failure.cause }
func (*repositoryWriteEffectsFailure) RepositoryWritesMayHaveApplied() {}
