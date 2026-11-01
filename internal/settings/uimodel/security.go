// Package uimodel contains UI-oriented repository settings.
package uimodel

// These are adopted out-of-UI observation fallbacks in #480,
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
	DependabotAlerts              *bool
	DependabotSecurityUpdates     *bool
	SecretProtection              *bool
	PushProtection                *bool
	PrivateVulnerabilityReporting *bool
}

type SecurityGeneralReleases struct {
	EnableReleaseImmutability *bool
}
