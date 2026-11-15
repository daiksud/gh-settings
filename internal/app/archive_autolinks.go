package app

import (
	"fmt"
	"reflect"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// The observations are independent reads, not an atomic snapshot.
func verifyArchivedAutolinks(repo repository.Repository, desired *[]settings.Autolink, archive settings.ArchiveState, read func(repository.Repository) (settings.AutolinkObservation, error)) error {
	if desired == nil || len(*desired) == 0 {
		return nil
	}
	for _, value := range *desired {
		if value.ID == nil || *value.ID <= 0 {
			return fmt.Errorf("verify archived autolinks: existing IDs required")
		}
	}
	if read == nil {
		return fmt.Errorf("verify archived autolinks: reader unavailable")
	}
	observe := func() (settings.AutolinkObservation, error) {
		o, err := read(repo)
		if err != nil {
			return o, err
		}
		if o.Target.ID != archive.ID || o.Target.FullName != archive.FullName || !o.Target.Archived {
			return o, fmt.Errorf("autolink repository identity or archive state changed")
		}
		changes, err := settings.CompareAutolinks(o, *desired, false)
		if err != nil || len(changes) != 0 {
			return o, fmt.Errorf("autolink selection is incomplete or differs")
		}
		return o, nil
	}
	first, err := observe()
	if err != nil {
		return fmt.Errorf("verify archived autolinks: %w", err)
	}
	fresh, err := observe()
	if err != nil {
		return fmt.Errorf("verify archived autolinks: %w", err)
	}
	if first.Target != fresh.Target {
		return fmt.Errorf("verify archived autolinks: repository identity changed")
	}
	for _, value := range *desired {
		before, ok := first.Find(*value.ID)
		after, found := fresh.Find(*value.ID)
		if !ok || !found || !reflect.DeepEqual(before, after) {
			return fmt.Errorf("verify archived autolinks: selected identity changed")
		}
	}
	return nil
}
