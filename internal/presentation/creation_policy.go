package presentation

import (
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
)

func CreationPolicyLine(change settings.Change) string {
	normalization := change.Normalization()
	var desired string
	change.Visit(func(bool, bool) {}, func(_, after string) {
		choice, _ := settings.CreationPolicyChoiceForValue(after)
		desired = string(choice)
	}, func([]string, []string) {})
	return NormalizationLine(normalization) + "\n" + fmt.Sprintf("%s: %s (normalized observation) → %s (write required)", normalization.Path, strconv.Quote(normalization.Fallback), strconv.Quote(desired))
}
