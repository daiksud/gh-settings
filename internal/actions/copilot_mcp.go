package actions

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func validateCopilotMCPReport(raw []PlanChange) ([]settings.CopilotMCPChange, error) {
	if len(raw) > 1 {
		return nil, fmt.Errorf("duplicate Copilot MCP tools report control")
	}
	var changes []settings.CopilotMCPChange
	for _, entry := range raw {
		before, beforeOK := entry.Before.(bool)
		after, afterOK := entry.After.(bool)
		if entry.Field != settings.CopilotMCPPath || !beforeOK || !afterOK || entry.Normalization != nil || entry.SecurityStatusNormalization != nil {
			return nil, fmt.Errorf("invalid Copilot MCP tools report")
		}
		change, err := settings.NewCopilotMCPChange(before, after)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
