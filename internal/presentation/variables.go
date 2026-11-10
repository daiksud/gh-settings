package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func VariableLine(change settings.VariableChange) string {
	return variableLine("repository.settings.actions.variables", change)
}

func AgentVariableLine(change settings.VariableChange) string {
	return variableLine("repository.settings.agents.variables", change)
}

func variableLine(path string, change settings.VariableChange) string {
	before, after := change.Values()
	prefix := fmt.Sprintf("%s: %s %q", path, change.Operation(), change.Name())
	if before == nil {
		return fmt.Sprintf("%s value=%q", prefix, after)
	}
	return fmt.Sprintf("%s value: %q → %q", prefix, *before, after)
}
