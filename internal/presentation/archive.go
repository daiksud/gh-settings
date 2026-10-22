package presentation

import "github.com/daiksud/gh-settings/internal/settings"

// ArchiveOrder explains when a lifecycle transition occurs relative to other writes.
// It is review context, not a guarantee that remote operations are atomic.
func ArchiveOrder(change settings.Change) string {
	if change.Field().Path() != "repository.archived" {
		return ""
	}
	var note string
	change.Visit(func(_, after bool) {
		if after {
			note = "Lifecycle order: archive after other settings."
		} else {
			note = "Lifecycle order: unarchive before other settings."
		}
	}, func(_, _ string) {}, func(_, _ []string) {})
	return note
}
