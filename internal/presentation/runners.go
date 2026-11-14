package presentation

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func RunnerLabelsLine(c settings.RunnerLabelsChange) string {
	before, after := c.Values()
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	return fmt.Sprintf("%s: %s → %s", c.Field(), b, a)
}

// RunnerDeletionLine shows only the owned removal-before projection.
func RunnerDeletionLine(d settings.RunnerDeletion) string {
	b, _ := json.Marshal(d)
	return fmt.Sprintf("%s: %s → null", settings.RunnersPath, b)
}
