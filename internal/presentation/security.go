package presentation

import (
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
)

func SecurityLine(change settings.SecurityChange) string {
	before, after := change.Values()
	if normalization := change.Normalization(); normalization != (settings.SecurityStatusNormalization{}) {
		return SecurityStatusNormalizationLine(normalization) + "\n" + fmt.Sprintf("%s: %t (normalized observation) → %t (write required)", normalization.Path, before, after)
	}
	return fmt.Sprintf("settings.%s: %t → %t", change.Field(), before, after)
}

func SecurityStatusNormalizationLine(record settings.SecurityStatusNormalization) string {
	return fmt.Sprintf("Warning: %s: observed %s is outside UI choices; using %t (projection only).", record.Path, strconv.Quote(record.RawValue), record.Fallback)
}
