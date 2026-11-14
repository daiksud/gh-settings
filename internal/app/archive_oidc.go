package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func verifyArchivedOIDC(repo repository.Repository, desired *settings.OIDCSubjectTemplate, archive settings.ArchiveState, read func(repository.Repository, settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error)) error {
	if desired == nil {
		return nil
	}
	if read == nil {
		return fmt.Errorf("verify archived OIDC subject: reader unavailable")
	}
	observations, err := read(repo, archive)
	if err != nil {
		return fmt.Errorf("verify archived OIDC subject: %w", err)
	}
	for _, observation := range observations {
		managed, changes, err := settings.CompareOIDCSubject(observation, desired)
		if err != nil {
			return fmt.Errorf("verify archived OIDC subject: %w", err)
		}
		if managed != 1 || len(changes) != 0 {
			return fmt.Errorf("archived OIDC subject differs or lacks selected evidence")
		}
	}
	return nil
}
