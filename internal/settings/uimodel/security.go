// Package uimodel prepares UI-oriented values without activating public YAML.
package uimodel

// These are candidates for out-of-UI observation normalization in #480,
// never defaults for omitted desired fields or failed acquisition.
const (
	DefaultSecretProtection = false
	DefaultPushProtection   = false
)

type SecuritySettings struct {
	AdvancedSecurity *AdvancedSecurity
	GeneralReleases  *SecurityGeneralReleases
}

type AdvancedSecurity struct {
	SecretProtection              *bool
	PushProtection                *bool
	PrivateVulnerabilityReporting *bool
}

type SecurityGeneralReleases struct {
	EnableReleaseImmutability *bool
}
