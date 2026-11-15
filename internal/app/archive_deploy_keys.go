package app

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Independent complete inventory reads do not form an atomic snapshot.
func verifyArchivedDeployKeys(repo repository.Repository, desired *settings.DeployKeyDeclarations, archive settings.ArchiveState, read func(repository.Repository) (settings.DeployKeyObservation, error)) error {
	if desired == nil || desired.Len() == 0 {
		return nil
	}
	for _, d := range desired.Values() {
		if d.ID == nil || *d.ID <= 0 {
			return fmt.Errorf("verify archived deploy keys: existing IDs required")
		}
	}
	if read == nil {
		return fmt.Errorf("verify archived deploy keys: reader unavailable")
	}
	observe := func() (settings.DeployKeyObservation, settings.DeployKeyTarget, error) {
		o, err := read(repo)
		// Never forward a transport error that might contain public key material.
		if err != nil {
			return o, settings.DeployKeyTarget{}, fmt.Errorf("observation unavailable")
		}
		target, _, err := o.Values()
		if err != nil || target.ID != archive.ID || !strings.EqualFold(target.FullName, archive.FullName) || !target.Archived {
			return o, target, fmt.Errorf("repository identity or archive state changed")
		}
		_, changes, err := settings.PrepareDeployKeys(o, desired, false)
		if err != nil || len(changes) != 0 {
			return o, target, fmt.Errorf("selected deploy keys are incomplete or differ")
		}
		return o, target, nil
	}
	first, ft, err := observe()
	if err != nil {
		return fmt.Errorf("verify archived deploy keys: %w", err)
	}
	fresh, st, err := observe()
	if err != nil {
		return fmt.Errorf("verify archived deploy keys: %w", err)
	}
	if ft != st {
		return fmt.Errorf("verify archived deploy keys: repository identity changed")
	}
	for _, d := range desired.Values() {
		before, ok := first.State(*d.ID)
		after, found := fresh.State(*d.ID)
		if !ok || !found || !settings.DeployKeyStableEqual(before, after) {
			return fmt.Errorf("verify archived deploy keys: selected identity changed")
		}
	}
	return nil
}
