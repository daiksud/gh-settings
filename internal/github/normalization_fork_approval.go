package github

import "github.com/daiksud/gh-settings/internal/settings"

func prepareForkApprovalObservation(raw *string) (settings.ForkApprovalObservation, error) {
	if raw == nil {
		return settings.ForkApprovalObservation{}, nil
	}
	if settings.ValidForkApprovalPolicy(*raw) {
		return settings.NewForkApprovalObservation(*raw)
	}
	fallback, _ := settings.ForkApprovalValueForChoice(settings.DefaultForkApprovalChoice)
	return settings.NewForkApprovalObservationWithNormalization(fallback, settings.ObservationNormalization{
		Path: settings.ForkApprovalPath, RawValue: *raw, Fallback: string(settings.DefaultForkApprovalChoice),
	})
}
