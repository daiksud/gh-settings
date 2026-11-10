package actions

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

func validateCopilotReviewEffortReport(raw []PlanChange) ([]settings.CopilotReviewEffortChange, error) {
	if len(raw) > 1 {
		return nil, fmt.Errorf("duplicate Copilot review effort report control")
	}
	var changes []settings.CopilotReviewEffortChange
	for _, entry := range raw {
		before, beforeOK := entry.Before.(string)
		after, afterOK := entry.After.(string)
		if entry.Field != settings.CopilotReviewEffortPath || !beforeOK || !afterOK || entry.Normalization != nil || entry.SecurityStatusNormalization != nil {
			return nil, fmt.Errorf("invalid Copilot review effort report")
		}
		change, err := settings.NewCopilotReviewEffortChange(before, after)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
