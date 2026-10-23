package presentation

import (
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func PropertyLine(c settings.PropertyChange) string {
	b, a := c.Values()
	before, _ := json.Marshal(b)
	after, _ := json.Marshal(a)
	return fmt.Sprintf("settings.custom_properties.%s: %s → %s", c.Name(), before, after)
}
