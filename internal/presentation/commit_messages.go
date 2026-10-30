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
