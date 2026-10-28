package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func SecurityDeclaration(ui uimodel.SecuritySettings) (settings.SecurityDeclaration, error) {
	var result settings.SecurityDeclaration
	if ui.AdvancedSecurity != nil {
		result.SecretScanning = copyUISecurityValue(ui.AdvancedSecurity.SecretProtection)
		result.PushProtection = copyUISecurityValue(ui.AdvancedSecurity.PushProtection)
		result.PrivateVulnerabilityReporting = copyUISecurityValue(ui.AdvancedSecurity.PrivateVulnerabilityReporting)
	}
	if ui.GeneralReleases != nil {
		result.ImmutableReleases = copyUISecurityValue(ui.GeneralReleases.EnableReleaseImmutability)
	}
	if err := result.Validate(); err != nil {
		return settings.SecurityDeclaration{}, err
	}
	return result, nil
}

func WithSecurityScanningObservation(ui uimodel.SecuritySettings, observation settings.SecretScanningObservation) (uimodel.SecuritySettings, error) {
	scanning, push, err := observation.Values()
	if err != nil {
		return uimodel.SecuritySettings{}, err
	}
	advanced := copyUISecurityAdvanced(ui.AdvancedSecurity)
	advanced.SecretProtection = copyUISecurityValue(&scanning)
	advanced.PushProtection = copyUISecurityValue(&push)
	ui.AdvancedSecurity = &advanced
	return ui, nil
}

func copyUISecurityAdvanced(value *uimodel.AdvancedSecurity) uimodel.AdvancedSecurity {
	if value == nil {
		return uimodel.AdvancedSecurity{}
	}
	return *value
}
func copyUISecurityValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func WithSecurityReportingObservation(ui uimodel.SecuritySettings, observation settings.PrivateReportingObservation) (uimodel.SecuritySettings, error) {
	value, err := observation.Value()
	if err != nil {
		return uimodel.SecuritySettings{}, err
	}
	advanced := copyUISecurityAdvanced(ui.AdvancedSecurity)
	advanced.PrivateVulnerabilityReporting = copyUISecurityValue(&value)
	ui.AdvancedSecurity = &advanced
	return ui, nil
}

// PreparedSecurityImmutable keeps owner enforcement outside desired UI controls.
// A nil EnforcedByOwner means unknown, including the existing disabled 404 case.
type PreparedSecurityImmutable struct {
	Settings        uimodel.SecuritySettings
	EnforcedByOwner *bool
}

func WithSecurityImmutableObservation(ui uimodel.SecuritySettings, observation settings.ImmutableReleasesObservation) (PreparedSecurityImmutable, error) {
	enabled, enforced, known, err := observation.Values()
	if err != nil {
		return PreparedSecurityImmutable{}, err
	}
	releases := uimodel.SecurityGeneralReleases{}
	if ui.GeneralReleases != nil {
		releases = *ui.GeneralReleases
	}
	releases.EnableReleaseImmutability = copyUISecurityValue(&enabled)
	ui.GeneralReleases = &releases
	result := PreparedSecurityImmutable{Settings: ui}
	if known {
		result.EnforcedByOwner = copyUISecurityValue(&enforced)
	}
	return result, nil
}
