package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func AutolinkLine(c settings.AutolinkChange) string {
	operation := "create"
	value := c.After
	identity := ""
	if c.Before != nil {
		operation = "delete"
		value = c.Before
		identity = fmt.Sprintf(" (id %d)", *value.ID)
	}
	return fmt.Sprintf("%s: %s%s reference_prefix: %q, target_url: %q, reference_format: %s", settings.AutolinksPath, operation, identity, value.ReferencePrefix, value.TargetURL, value.ReferenceFormat)
}
