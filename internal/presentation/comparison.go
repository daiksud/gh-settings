package presentation

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RenderChanges passes each formatted entry to the caller in comparison order.
func RenderChanges[T any](changes []T, format func(T) string, writeLine func(string)) {
	for _, change := range changes {
		writeLine(format(change))
	}
}

// RenderRepositoryChanges groups native field changes by their displayed UI
// control. The caller decides how to escape and delimit each rendered entry.
func RenderRepositoryChanges(comparison *settings.Comparison, writeLine func(string)) {
	shownMenus := map[settings.CommitMessageMenu]bool{}
	for _, change := range comparison.Changes {
		path := change.Field().Path()
		if context, ok := CommitMessageContextForField(path, comparison.CommitMessageContexts); ok {
			if !shownMenus[context.Menu] {
				writeLine(ExactCommitMessageLine(context))
				shownMenus[context.Menu] = true
			}
			continue
		}
		if change.Normalization() != (settings.ObservationNormalization{}) {
			writeLine(CreationPolicyLine(change))
			continue
		}
		writeValues := func(before, after string) {
			writeLine(fmt.Sprintf("%s: %s → %s", SettingPath(path), before, after))
		}
		change.Visit(
			func(before, after bool) { writeValues(strconv.FormatBool(before), strconv.FormatBool(after)) },
			func(before, after string) {
				before, after = StringValues(path, before, after)
				writeValues(strconv.Quote(before), strconv.Quote(after))
			},
			func(before, after []string) {
				beforeJSON, _ := json.Marshal(before)
				afterJSON, _ := json.Marshal(after)
				writeValues(string(beforeJSON), string(afterJSON))
			},
		)
		if note := ArchiveOrder(change); note != "" {
			writeLine(note)
		}
	}
	for _, change := range comparison.CommitMessages {
		writeLine(CommitMessageLine(change))
	}
}

// RenderActionsPolicyChanges shows each execution permission menu once, even
// when both its enabled state and its allowed actions changed.
func RenderActionsPolicyChanges(comparison *settings.Comparison, writeLine func(string)) {
	for _, change := range comparison.ActionsExecution {
		writeLine(ActionsExecutionLine(change))
	}
	shownExecution := false
	for _, change := range comparison.Actions {
		if IsActionsExecutionField(change.Field()) {
			if !shownExecution {
				writeLine(ExactActionsExecutionLine(comparison.ActionsExecutionContext))
				shownExecution = true
			}
			continue
		}
		writeLine(ActionsPolicyLine(change))
	}
}
