package actions

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// CommitMessagePlanChange represents one normalized menu owning two native fields.
type CommitMessagePlanChange struct {
	Menu          settings.CommitMessageMenu          `json:"menu"`
	After         uimodel.CommitMessageChoice         `json:"after"`
	Normalization settings.CommitMessageNormalization `json:"normalization"`
}

func validateCommitMessageReport(entries []CommitMessagePlanChange, native []PlanChange) ([]settings.NormalizedCommitMessageChange, error) {
	var changes []settings.NormalizedCommitMessageChange
	seen := map[settings.CommitMessageMenu]bool{}
	for _, entry := range entries {
		if seen[entry.Menu] {
			return nil, fmt.Errorf("duplicate normalized commit message menu")
		}
		seen[entry.Menu] = true
		change, err := settings.NewNormalizedCommitMessageChange(entry.Menu, entry.Normalization, entry.After)
		if err != nil {
			return nil, err
		}
		for _, field := range change.NativeFields() {
			for _, raw := range native {
				if raw.Field == field.Path() {
					return nil, fmt.Errorf("normalized commit message menu overlaps native component")
				}
			}
		}
		changes = append(changes, change)
	}
	return changes, nil
}
