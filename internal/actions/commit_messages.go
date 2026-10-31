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

// validateCommitMessageContexts binds complete menu evidence to the original
// changed native fields and rejects fabricated or unrelated companion values.
func validateCommitMessageContexts(contexts []settings.CommitMessageContext, changes []PlanChange) error {
	seen := map[settings.CommitMessageMenu]bool{}
	for _, context := range contexts {
		if seen[context.Menu] {
			return fmt.Errorf("duplicate commit message context")
		}
		seen[context.Menu] = true
		if _, _, err := context.Choices(); err != nil {
			return err
		}
		var title, message string
		switch context.Menu {
		case settings.MergeCommitMenu:
			title, message = "repository.merge_commit_title", "repository.merge_commit_message"
		case settings.SquashCommitMenu:
			title, message = "repository.squash_merge_commit_title", "repository.squash_merge_commit_message"
		default:
			return fmt.Errorf("invalid commit message context")
		}
		for _, component := range []struct{ field, before, after string }{{title, context.Before.Title, context.After.Title}, {message, context.Before.Message, context.After.Message}} {
			matched := false
			for _, raw := range changes {
				if raw.Field != component.field {
					continue
				}
				if matched || raw.Normalization != nil || raw.SecurityStatusNormalization != nil || raw.Before != component.before || raw.After != component.after || component.before == component.after {
					return fmt.Errorf("commit message context does not match changed component")
				}
				matched = true
			}
			if matched != (component.before != component.after) {
				return fmt.Errorf("commit message context is missing a changed component")
			}
		}
	}
	for _, raw := range changes {
		if menu, ok := settings.CommitMessageMenuForField(raw.Field); ok && !seen[menu] {
			return fmt.Errorf("missing commit message context")
		}
	}
	return nil
}
