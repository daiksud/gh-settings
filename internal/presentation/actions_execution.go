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

func ExactActionsExecutionLine(context *settings.ActionsExecutionContext) string {
	if context == nil || context.Validate() != nil {
		return settings.ActionsPermissionsPath + ": selection unavailable"
	}
	return fmt.Sprintf("%s: %q → %q", settings.ActionsPermissionsPath, context.Before, context.After)
}

func IsActionsExecutionField(field string) bool {
	return field == "actions.enabled" || field == "actions.allowed_actions"
}
