package github

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ReadArchived acquires the same state as the normal writer, twice, and binds
// both observations to the enclosing lifecycle identity. These reads are not
// an atomic snapshot or a repository identity lock.
func (c *OIDCSubjectClient) ReadArchived(repo repository.Repository, archive settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
	var observations [2]settings.OIDCSubjectObservation
	if archive.Validate(repo) != nil || !archive.Archived {
		return observations, fmt.Errorf("invalid archived OIDC repository identity")
	}
	var ownerID int64
	for i := range observations {
		state, err := c.read(repo)
		if err != nil {
			return observations, err
		}
		if state.id != archive.ID || !state.archived || i > 0 && state.ownerID != ownerID {
			return observations, fmt.Errorf("archived OIDC repository identity changed")
		}
		ownerID = state.ownerID
		state.value.UseImmutableSubjectClaim = state.immutable
		observations[i], err = settings.NewOIDCSubjectObservation(state.value)
		if err != nil {
			return observations, err
		}
	}
	return observations, nil
}
