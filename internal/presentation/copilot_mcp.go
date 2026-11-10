package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func CopilotMCPLine(change settings.CopilotMCPChange) string {
	before, after := change.Values()
	return fmt.Sprintf("%s: %t → %t", change.Field(), before, after)
}
