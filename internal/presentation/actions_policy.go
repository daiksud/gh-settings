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
	before = actionsDisplayValue(change.Field(), before)
	after = actionsDisplayValue(change.Field(), after)
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	value := string(b)
	if before == nil {
		value = "(unavailable while policy is not selected)"
	}
	return fmt.Sprintf("%s: %s → %s", SettingPath(change.Field()), value, a)
}

func actionsDisplayValue(field string, value any) any {
	v, ok := value.(string)
	if !ok {
		return value
	}
	switch field {
	case "actions.fork_pr_approval_policy":
		if choice, valid := settings.ForkApprovalChoiceForValue(v); valid {
			return string(choice)
		}
	case "actions.default_workflow_permissions":
		if v == "read" {
			return string(uimodel.ReadRepositoryContentsAndPackagesPermissions)
		}
		if v == "write" {
			return string(uimodel.ReadAndWritePermissions)
		}
	}
	return value
}
