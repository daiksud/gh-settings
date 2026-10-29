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
	ui.CheckWorkflowRunStatusArtifactAndLogRetention = copyUIActionsValue(days)
	return ui, nil
}

// PreparedActionsRetention preserves the unowned cap independently of configured
// days. Applying/exporting the value still requires existing cap validation.
type PreparedActionsRetention struct {
	General            uimodel.ActionsGeneral
	MaximumAllowedDays int
}

func WithActionsRetentionObservation(ui uimodel.ActionsGeneral, observed settings.RetentionObservation) (PreparedActionsRetention, error) {
	days, maximum, err := observed.Values()
	if err != nil {
		return PreparedActionsRetention{}, err
	}
	ui.CheckWorkflowRunStatusArtifactAndLogRetention = &days
	return PreparedActionsRetention{General: ui, MaximumAllowedDays: maximum}, nil
}
