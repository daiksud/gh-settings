package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func WithActionsRetentionPolicyUI(ui uimodel.ActionsGeneral, days *int) (uimodel.ActionsGeneral, error) {
	if days == nil {
		return ui, nil
	}
	if err := settings.ValidateRetentionDays(*days); err != nil {
		return uimodel.ActionsGeneral{}, err
	}
	ui.CheckWorkflowRunStatusArtifactAndLogRetention = copyUIValue(days)
	return ui, nil
}
