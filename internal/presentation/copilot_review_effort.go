package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func CopilotReviewEffortLine(change settings.CopilotReviewEffortChange) string {
	before, after := change.Values()
	return fmt.Sprintf("%s: %s → %s", change.Field(), before, after)
}
