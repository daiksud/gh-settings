package presentation

import (
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"strconv"
)

func PagesLine(c settings.PagesChange) string {
	b, a := c.Values()
	if record := c.Normalization(); record != (settings.ObservationNormalization{}) {
		value, _ := a.(string)
		choice, _ := settings.PagesFolderChoiceForValue(value)
		return NormalizationLine(record) + "\n" + fmt.Sprintf("%s: %s (normalized observation) → %s (write required)", record.Path, strconv.Quote(record.Fallback), strconv.Quote(string(choice)))
	}
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
