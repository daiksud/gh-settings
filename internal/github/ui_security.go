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
			q := cs.CodeQLAnalysis
			result.CodeQLDefaultSetup = copyUIValue(q.DefaultSetup)
			result.CodeQLConfiguration.Languages = q.Languages
			if scan := q.ScanSettings; scan != nil {
				result.CodeQLConfiguration.QuerySuite = scan.QuerySuite
				result.CodeQLConfiguration.ThreatModel = scan.ThreatModel
				if scan.RunnerLabel != nil && (scan.RunnerType == nil || *scan.RunnerType != "labeled_runner") {
					return settings.SecurityDeclaration{}, fmt.Errorf("%s.scan_settings.runner_label requires runner_type labeled_runner", settings.CodeQLConfigurationPath)
				}
				if scan.RunnerType != nil {
					runner := &settings.CodeQLRunner{Type: *scan.RunnerType}
					if scan.RunnerLabel != nil {
						runner.Label = *scan.RunnerLabel
					}
					result.CodeQLConfiguration.Runner = runner
				}
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
