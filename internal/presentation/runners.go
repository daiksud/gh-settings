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
