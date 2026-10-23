package presentation

import (
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func PagesLine(c settings.PagesChange) string {
	b, a := c.Values()
	before, _ := json.Marshal(b)
	after, _ := json.Marshal(a)
	if c.BeforeInactive() {
		before = []byte("inactive")
	}
	line := fmt.Sprintf("settings.%s: %s → %s", c.Field(), before, after)
	if c.Field() == "pages.build_type" && a == "workflow" {
		line += " (branch source becomes inactive)"
	}
	return line
}
