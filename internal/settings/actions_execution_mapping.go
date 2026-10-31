package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const ActionsPermissionsPath = "repository.settings.actions.general.actions_permissions.policy"
const DefaultActionsPermissionsChoice = uimodel.DefaultActionsPermissions

var actionsPermissionsChoices = [...]struct {
	choice  uimodel.ActionsPermissionChoice
	enabled bool
	allowed string
}{
	{uimodel.AllowAllActionsAndReusableWorkflows, true, "all"},
	{uimodel.DisableActions, false, ""},
	{uimodel.AllowOwnerActionsAndReusableWorkflows, true, "local_only"},
	{uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows, true, "selected"},
}

func ActionsPermissionsValueForChoice(choice uimodel.ActionsPermissionChoice) (enabled bool, allowed *string, valid bool) {
	for _, option := range actionsPermissionsChoices {
		if option.choice == choice {
			if !option.enabled {
				return false, nil, true
			}
			value := option.allowed
			return true, &value, true
		}
	}
	return false, nil, false
}

func ActionsPermissionsChoiceForValue(enabled bool, allowed string) (uimodel.ActionsPermissionChoice, bool) {
	for _, option := range actionsPermissionsChoices {
		if option.enabled == enabled && (!enabled || option.allowed == allowed) {
			return option.choice, true
		}
	}
	return "", false
}
