package settings

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// ActionsExecutionChange owns one radio decision while preserving its native
// field ownership and the visibility of the original policy projection.
type ActionsExecutionChange struct {
	beforeEnabled bool
	after         uimodel.ActionsPermissionChoice
	normalization ObservationNormalization
}

func NewActionsExecutionChange(observed ActionsPolicyState, desired uimodel.ActionsPermissionChoice) (ActionsExecutionChange, error) {
	if err := observed.Validate(); err != nil {
		return ActionsExecutionChange{}, err
	}
	if observed.AllowedActionsNormalization == (ObservationNormalization{}) {
		return ActionsExecutionChange{}, fmt.Errorf("Actions radio decision requires original policy provenance")
	}
	if _, _, valid := ActionsPermissionsValueForChoice(desired); !valid {
		return ActionsExecutionChange{}, fmt.Errorf("invalid desired Actions permission choice")
	}
	if !observed.Enabled && desired == uimodel.DisableActions {
		return ActionsExecutionChange{}, fmt.Errorf("known disabled Actions already matches intent")
	}
	return ActionsExecutionChange{beforeEnabled: observed.Enabled, after: desired, normalization: observed.AllowedActionsNormalization}, nil
}

func (c ActionsExecutionChange) Before() uimodel.ActionsPermissionChoice {
	if c.after == "" {
		return ""
	}
	if !c.beforeEnabled {
		return uimodel.DisableActions
	}
	return DefaultActionsPermissionsChoice
}
func (c ActionsExecutionChange) After() uimodel.ActionsPermissionChoice  { return c.after }
func (c ActionsExecutionChange) Normalization() ObservationNormalization { return c.normalization }
func (c ActionsExecutionChange) ProjectionVisible() bool                 { return c.beforeEnabled }
func (c ActionsExecutionChange) NativeFields() []string {
	if c.after == "" {
		return nil
	}
	fields := []string{"actions.enabled"}
	if c.after != uimodel.DisableActions {
		fields = append(fields, "actions.allowed_actions")
	}
	return fields
}
