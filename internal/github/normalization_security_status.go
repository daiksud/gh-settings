package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

type preparedSecurityStatus struct {
	Value         *bool
	Normalization settings.SecurityStatusNormalization
}

func prepareSecurityStatusObservation(raw *string, control settings.SecurityStatusControl) (preparedSecurityStatus, error) {
	fallback, valid := control.DefaultValue()
	if !valid {
		return preparedSecurityStatus{}, fmt.Errorf("unsupported security status control %q", control)
	}
	if raw == nil {
		return preparedSecurityStatus{}, nil
	}
	if value, valid := settings.SecurityValueForStatus(*raw); valid {
		return preparedSecurityStatus{Value: &value}, nil
	}
	return preparedSecurityStatus{Value: &fallback, Normalization: settings.SecurityStatusNormalization{Path: control.Path(), RawValue: *raw, Fallback: fallback}}, nil
}

func (client *SecretScanningClient) warnStatusNormalization(observation settings.SecretScanningObservation, field string) {
	if _, _, err := observation.RawStatuses(); err != nil || client.NormalizationWarning == nil {
		return
	}
	for _, control := range []settings.SecurityStatusControl{settings.SecretProtectionControl, settings.PushProtectionControl} {
		if control.Field() != field {
			continue
		}
		for _, record := range observation.Normalizations() {
			if record.Path == control.Path() {
				client.NormalizationWarning(record)
			}
		}
	}
}
