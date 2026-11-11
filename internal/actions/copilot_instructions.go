package actions

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func validateCopilotInstructionsReport(raw []PlanChange) ([]settings.CopilotInstructionsChange, error) {
	if len(raw) > 1 {
		return nil, fmt.Errorf("duplicate Copilot custom instructions report control")
	}
	var changes []settings.CopilotInstructionsChange
	for _, entry := range raw {
		before, beforeOK := entry.Before.(bool)
		after, afterOK := entry.After.(bool)
		if entry.Field != settings.CopilotInstructionsPath || !beforeOK || !afterOK || entry.Normalization != nil || entry.SecurityStatusNormalization != nil {
			return nil, fmt.Errorf("invalid Copilot custom instructions report")
		}
		change, err := settings.NewCopilotInstructionsChange(before, after)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
