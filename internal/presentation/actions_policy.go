package presentation

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func ActionsPolicyLine(change settings.ActionsPolicyChange) string {
	before, after := change.Values()
	if normalization := change.Normalization(); normalization.Path != "" {
		choice := string(uimodel.ReadRepositoryContentsAndPackagesPermissions)
		switch change.Field() {
		case "actions.fork_pr_approval_policy":
			value, _ := after.(string)
			forkChoice, _ := settings.ForkApprovalChoiceForValue(value)
			choice = string(forkChoice)
		case "actions.default_workflow_permissions":
			if after == "write" {
				choice = string(uimodel.ReadAndWritePermissions)
			}
		}
		return NormalizationLine(normalization) + "\n" + fmt.Sprintf("%s: %s (normalized observation) → %s (write required)", normalization.Path, strconv.Quote(normalization.Fallback), strconv.Quote(choice))
	}
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	value := string(b)
	if before == nil {
		value = "(unavailable while policy is not selected)"
	}
	return fmt.Sprintf("settings.%s: %s → %s", change.Field(), value, a)
}
