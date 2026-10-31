package settings

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// ActionsExecutionContext binds native comparison entries to the radio choice.
// It retains the native companion for validation; rendering never reveals a
// disabled radio's latent allowed-actions value.
type ActionsExecutionContext struct {
	Before          uimodel.ActionsPermissionChoice `json:"before"`
	After           uimodel.ActionsPermissionChoice `json:"after"`
	ObservedAllowed string                          `json:"observed_allowed"`
}

func (c ActionsExecutionContext) Validate() error {
	if _, _, ok := ActionsPermissionsValueForChoice(c.Before); !ok {
		return fmt.Errorf("invalid observed Actions permission")
	}
	if !ValidAllowedActions(c.ObservedAllowed) {
		return fmt.Errorf("invalid observed Actions policy")
	}
	if choice, ok := ActionsPermissionsChoiceForValue(true, c.ObservedAllowed); c.Before != uimodel.DisableActions && (!ok || choice != c.Before) {
		return fmt.Errorf("inconsistent observed Actions policy")
	}
	if _, _, ok := ActionsPermissionsValueForChoice(c.After); !ok || c.Before == c.After {
		return fmt.Errorf("invalid or unchanged desired Actions permission")
	}
	return nil
}
