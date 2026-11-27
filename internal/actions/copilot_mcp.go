package actions

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func validateCopilotToggleReport(raw []PlanChange, field string) ([]settings.CopilotToggleChange, error) {
	if len(raw) > 1 {
		return nil, fmt.Errorf("duplicate %s report control", field)
	}
	var changes []settings.CopilotToggleChange
	for _, entry := range raw {
		before, beforeOK := entry.Before.(bool)
		after, afterOK := entry.After.(bool)
		if entry.Field != field || !beforeOK || !afterOK || entry.Normalization != nil || entry.SecurityStatusNormalization != nil {
			return nil, fmt.Errorf("invalid %s report", field)
		}
		change, err := settings.NewCopilotToggleChange(field, before, after)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
