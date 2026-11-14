package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Both selected controls must match on each independent read. The lifecycle
// checks surrounding these reads are not an atomic repository snapshot.
func verifyArchivedCopilot(repo repository.Repository, instructions, mcp *bool, read func(repository.Repository, bool, bool) (settings.ArchivedCopilotToggles, error)) error {
	if instructions == nil && mcp == nil {
		return nil
	}
	if read == nil {
		return fmt.Errorf("verify archived Copilot toggles: reader unavailable")
	}
	for range 2 {
		observed, err := read(repo, instructions != nil, mcp != nil)
		if err != nil {
			return fmt.Errorf("verify archived Copilot toggles: observation unavailable")
		}
		if instructions != nil && (observed.Instructions == nil || *observed.Instructions != *instructions) ||
			mcp != nil && (observed.MCP == nil || *observed.MCP != *mcp) {
			return fmt.Errorf("archived Copilot toggles differ or lack selected evidence")
		}
	}
	return nil
}
