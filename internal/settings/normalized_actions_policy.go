package settings

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// NewActionsPolicyChangeWithNormalization records a required write even when a
// UI fallback equals desired intent. Only consistent acquired provenance can
// authorize this otherwise invalid equal-value comparison.
func NewActionsPolicyChangeWithNormalization(field string, before, after any, normalization ObservationNormalization) (ActionsPolicyChange, error) {
	switch field {
	case "actions.default_workflow_permissions":
		if before != "read" || normalization.Path != WorkflowPermissionsPath || normalization.Fallback != string(uimodel.DefaultActionsWorkflowPermissions) || ValidDefaultWorkflowPermissions(normalization.RawValue) {
			return ActionsPolicyChange{}, fmt.Errorf("invalid workflow permission normalization provenance")
		}
	case "actions.fork_pr_approval_policy":
		value, valid := before.(string)
		if !valid {
			return ActionsPolicyChange{}, fmt.Errorf("fork approval normalization requires an effective string value")
		}
		if _, err := NewForkApprovalObservationWithNormalization(value, normalization); err != nil {
			return ActionsPolicyChange{}, err
		}
	default:
		return ActionsPolicyChange{}, fmt.Errorf("field %q cannot describe Actions normalization", field)
	}
	value, err := actionValue(field, after)
	if err != nil {
		return ActionsPolicyChange{}, err
	}
	return ActionsPolicyChange{field: field, before: before, after: value, normalization: normalization}, nil
}
