package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func CopilotInstructionsLine(change settings.CopilotInstructionsChange) string {
	before, after := change.Values()
	return fmt.Sprintf("%s: %t → %t", change.Field(), before, after)
}
