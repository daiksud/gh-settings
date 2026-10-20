package presentation

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func SecurityLine(change settings.SecurityChange) string {
	before, after := change.Values()
	return fmt.Sprintf("settings.%s: %t → %t", change.Field(), before, after)
}
