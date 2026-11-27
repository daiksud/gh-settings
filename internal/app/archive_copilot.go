package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// All selected controls must match on each independent read. The lifecycle
// checks surrounding these reads are not an atomic repository snapshot.
func verifyArchivedCopilot(repo repository.Repository, selected settings.ArchivedCopilotToggles, read func(repository.Repository, settings.ArchivedCopilotToggles) (settings.ArchivedCopilotToggles, error)) error {
	if selected == (settings.ArchivedCopilotToggles{}) {
		return nil
	}
	if read == nil {
		return fmt.Errorf("verify archived Copilot toggles: reader unavailable")
	}
	for range 2 {
		observed, err := read(repo, selected)
		if err != nil {
			return fmt.Errorf("verify archived Copilot toggles: observation unavailable")
		}
		if selected.Instructions != nil && (observed.Instructions == nil || *observed.Instructions != *selected.Instructions) ||
			selected.MCP != nil && (observed.MCP == nil || *observed.MCP != *selected.MCP) ||
			selected.PublicSearch != nil && (observed.PublicSearch == nil || *observed.PublicSearch != *selected.PublicSearch) {
			return fmt.Errorf("archived Copilot toggles differ or lack selected evidence")
		}
	}
	return nil
}
