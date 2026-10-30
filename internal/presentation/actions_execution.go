package presentation

import (
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
)

func ActionsExecutionLine(change settings.ActionsExecutionChange) string {
	before := strconv.Quote(string(change.Before()))
	warning := ""
	if change.ProjectionVisible() {
		warning = NormalizationLine(change.Normalization()) + "\n"
		before += " (normalized observation)"
	}
	return warning + fmt.Sprintf("%s: %s → %s (write required)", settings.ActionsPermissionsPath, before, strconv.Quote(string(change.After())))
}
