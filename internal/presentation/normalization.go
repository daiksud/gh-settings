package presentation

import (
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
)

func NormalizationLine(normalization settings.ObservationNormalization) string {
	return fmt.Sprintf("Warning: %s: observed %s is outside UI choices; using %s (projection only).", normalization.Path, strconv.Quote(normalization.RawValue), strconv.Quote(normalization.Fallback))
}
