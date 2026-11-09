package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func VariableLine(change settings.VariableChange) string {
	before, after := change.Values()
	prefix := fmt.Sprintf("repository.settings.actions.variables: %s %q", change.Operation(), change.Name())
	if before == nil {
		return fmt.Sprintf("%s value=%q", prefix, after)
	}
	return fmt.Sprintf("%s value: %q → %q", prefix, *before, after)
}
