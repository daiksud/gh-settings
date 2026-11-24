package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func SecurityDeclaration(ui uimodel.SecuritySettings) (settings.SecurityDeclaration, error) {
	var result settings.SecurityDeclaration
	if ui.AdvancedSecurity != nil {
		if cs := ui.AdvancedSecurity.CodeSecurity; cs != nil && cs.AIScan != nil {
			result.AIScanPullRequests = copyUIValue(cs.AIScan.AIScanForPullRequests)
			result.AIScanRepositories = copyUIValue(cs.AIScan.AIScanForRepositories)
		}
		if cs := ui.AdvancedSecurity.CodeSecurity; cs != nil && cs.CodeQLAnalysis != nil {
			result.CodeQLDefaultSetup = copyUIValue(cs.CodeQLAnalysis.DefaultSetup)
			q := cs.CodeQLAnalysis
			result.CodeQLConfiguration = settings.CodeQLConfigurationDeclaration{Languages: q.Languages, QuerySuite: q.QuerySuites, ThreatModel: q.ThreatModel}
			if q.Runner != nil {
				v := settings.CodeQLRunner{Type: q.Runner.Type}
				if q.Runner.Label != nil {
					v.Label = *q.Runner.Label
				}
				if q.Runner.Type == "standard_github_runner" && q.Runner.Label != nil {
					return settings.SecurityDeclaration{}, fmt.Errorf("standard CodeQL runner forbids a label")
				}
				result.CodeQLConfiguration.Runner = &v
			}

		}
		result.DependabotAlerts = copyUIValue(ui.AdvancedSecurity.DependabotAlerts)
		result.DependabotSecurityUpdates = copyUIValue(ui.AdvancedSecurity.DependabotSecurityUpdates)
		result.SecretScanning = copyUIValue(ui.AdvancedSecurity.SecretProtection)
		result.PushProtection = copyUIValue(ui.AdvancedSecurity.PushProtection)
		result.PrivateVulnerabilityReporting = copyUIValue(ui.AdvancedSecurity.PrivateVulnerabilityReporting)
	}
	if ui.GeneralReleases != nil {
		result.ImmutableReleases = copyUIValue(ui.GeneralReleases.EnableReleaseImmutability)
	}
	if err := result.Validate(); err != nil {
		return settings.SecurityDeclaration{}, err
	}
	return result, nil
}
