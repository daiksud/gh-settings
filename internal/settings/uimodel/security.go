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

type AIScan struct{ AIScanForPullRequests *bool }

type CodeSecurity struct {
	AIScan         *AIScan
	CodeQLAnalysis *CodeQLAnalysis
}
type CodeQLRunner struct {
	Type  string
	Label *string
}
type CodeQLAnalysis struct {
	DefaultSetup *bool
	Languages    *[]string
	QuerySuite   *string
	Runner       *CodeQLRunner
	ThreatModel  *string
}

type AdvancedSecurity struct {
	CodeSecurity                  *CodeSecurity
	DependabotAlerts              *bool
	DependabotSecurityUpdates     *bool
	SecretProtection              *bool
	PushProtection                *bool
	PrivateVulnerabilityReporting *bool
}

type SecurityGeneralReleases struct {
	EnableReleaseImmutability *bool
}
