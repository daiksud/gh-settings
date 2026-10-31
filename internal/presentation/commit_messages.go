package presentation

import (
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
)

func CommitMessageNormalizationLine(normalization settings.CommitMessageNormalization) string {
	return fmt.Sprintf("Warning: %s: observed title %s and message %s are outside UI choices; using %s (projection only).", normalization.Path, strconv.Quote(normalization.RawTitle), strconv.Quote(normalization.RawMessage), strconv.Quote(string(normalization.Fallback)))
}

func CommitMessageLine(change settings.NormalizedCommitMessageChange) string {
	before, after := change.Values()
	normalization := change.Normalization()
	return CommitMessageNormalizationLine(normalization) + "\n" + fmt.Sprintf("%s: %s (normalized observation) → %s (write required)", normalization.Path, strconv.Quote(string(before)), strconv.Quote(string(after)))
}

// ExactCommitMessageLine formats a complete, exact pair as one UI selection.
func ExactCommitMessageLine(context settings.CommitMessageContext) string {
	before, after, err := context.Choices()
	if err != nil {
		return context.Menu.Path() + ": selection unavailable"
	}
	return fmt.Sprintf("%s: %q → %q", context.Menu.Path(), before, after)
}

func CommitMessageContextForField(path string, contexts []settings.CommitMessageContext) (settings.CommitMessageContext, bool) {
	menu, ok := settings.CommitMessageMenuForField(path)
	if !ok {
		return settings.CommitMessageContext{}, false
	}
	for _, context := range contexts {
		if context.Menu == menu {
			return context, true
		}
	}
	return settings.CommitMessageContext{Menu: menu}, true
}
