package presentation

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func PullRequestLimitsLine(change settings.PullRequestLimitsChange) string {
	before, after := change.Values()
	return fmt.Sprintf("%s: %v → %v", change.Field(), before, after)
}
