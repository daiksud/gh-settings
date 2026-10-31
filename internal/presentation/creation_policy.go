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

// StringValues converts supported creation policies for presentation only.
func StringValues(field, before, after string) (string, string) {
	switch field {
	case settings.IssuePermissionsControl.Field().Path(), settings.PullRequestPermissionsControl.Field().Path():
		b, bok := settings.CreationPolicyChoiceForValue(before)
		a, aok := settings.CreationPolicyChoiceForValue(after)
		if bok && aok {
			return string(b), string(a)
		}
	}
	return before, after
}
