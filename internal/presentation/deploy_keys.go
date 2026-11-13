package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func DeployKeyLine(change settings.DeployKeyChange) string {
	before, after := change.Values()
	value := after
	if value == nil {
		value = before
	}
	if value == nil {
		return settings.DeployKeysPath + ": invalid comparison"
	}
	identity := ""
	if value.ID != nil {
		identity = fmt.Sprintf(" (id %d)", *value.ID)
	}
	return fmt.Sprintf("%s[%q]%s: %s %s (allow_write_access=%t)", settings.DeployKeysPath, value.Title, identity, change.Operation(), value.Fingerprint, value.AllowWriteAccess)
}
