package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func CopilotToggleLine(change settings.CopilotToggleChange) string {
	before, after := change.Values()
	return fmt.Sprintf("%s: %t → %t", change.Field(), before, after)
}
