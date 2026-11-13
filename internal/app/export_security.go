package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ErrExportCodeQLDisabled is a bounded diagnostic category, not upstream text.
var ErrExportCodeQLDisabled = errors.New("CodeQL default setup is confirmed disabled; configuration removed")

// Failed acquisition never invents false or an empty observed value. CodeQL
// state is independently usable when details from the same response are unknown.
func exportSecurity(checkout ExportCheckout, read Reader, repo repository.Repository) (settings.SecurityDeclaration, error) {
	result := checkout.Existing.Security
	var diagnosticError error
	observe := func(path string, observe func(*settings.SecurityDeclaration) error) {
		candidate := result
		if err := observe(&candidate); err != nil {
			if checkout.Info != nil {
				diagnosticError = errors.Join(diagnosticError, checkout.Info(path, err))
			}
			return
		}
		result = candidate
	}
	observe("repository.settings.advanced_security.secret_protection", func(candidate *settings.SecurityDeclaration) error {
		if read.SecretScanning == nil {
			return fmt.Errorf("reader unavailable")
		}

		observed, err := read.SecretScanning(repo)
		if err != nil {
			return err
		}
		if _, _, err := observed.Values(); err != nil {
			return err
		}
		if checkout.SecurityStatusNormalizationWarning != nil {
			for _, record := range observed.Normalizations() {
				diagnosticError = errors.Join(diagnosticError, checkout.SecurityStatusNormalizationWarning(record))
			}
		}
		d, err := observed.Declaration()
		if err != nil {
			return err
		}
		candidate.SecretScanning = d.SecretScanning
		candidate.PushProtection = d.PushProtection
		return nil
	})
	observe("repository.settings.advanced_security.dependabot", func(candidate *settings.SecurityDeclaration) error {
		if read.Dependabot == nil {
			return fmt.Errorf("reader unavailable")
		}

		o, err := read.Dependabot(repo)
		if err != nil {
			return err
		}
		d, err := o.Declaration()
		if err != nil {
			return err
		}
		candidate.DependabotAlerts = d.DependabotAlerts
		candidate.DependabotSecurityUpdates = d.DependabotSecurityUpdates

		return nil
	})
	observe("repository.settings.advanced_security.code_security.codeql_analysis", func(candidate *settings.SecurityDeclaration) error {
		if read.CodeQLForExport == nil {
			return fmt.Errorf("reader unavailable")
		}
		setup, configuration, readErr := read.CodeQLForExport(repo)
		enabled, stateErr := setup.Value()
		if stateErr != nil {
			if readErr != nil {
				return readErr
			}
			return stateErr
		}
		candidate.CodeQLDefaultSetup = &enabled
		if !enabled {
			candidate.CodeQLConfiguration = settings.CodeQLConfigurationDeclaration{}
			if !result.CodeQLConfiguration.Empty() && checkout.Info != nil {
				diagnosticError = errors.Join(diagnosticError, checkout.Info("repository.settings.advanced_security.code_security.codeql_analysis", ErrExportCodeQLDisabled))
			}
			return nil
		}
		if readErr == nil {
			var d settings.CodeQLConfigurationDeclaration
			d, readErr = configuration.Declaration()
			if readErr == nil {
				candidate.CodeQLConfiguration = d
			}
		}
		if readErr != nil && checkout.Info != nil {
			diagnosticError = errors.Join(diagnosticError, checkout.Info("repository.settings.advanced_security.code_security.codeql_analysis", readErr))
		}
		return nil
	})
	observe("repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_pull_requests", func(candidate *settings.SecurityDeclaration) error {
		if read.AIScanPullRequests == nil {
			return fmt.Errorf("reader unavailable")
		}

		observed, err := read.AIScanPullRequests(repo)
		if err != nil {
			return err
		}
		enabled, err := observed.Value()
		if err != nil {
			return err
		}
		candidate.AIScanPullRequests = &enabled

		return nil
	})
	observe("repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_repositories", func(candidate *settings.SecurityDeclaration) error {
		if read.AIScanRepositories == nil {
			return fmt.Errorf("reader unavailable")
		}

		observed, err := read.AIScanRepositories(repo)
		if err != nil {
			return err
		}
		enabled, err := observed.Value()
		if err != nil {
			return err
		}
		candidate.AIScanRepositories = &enabled

		return nil
	})
	observe("repository.settings.advanced_security.private_vulnerability_reporting", func(candidate *settings.SecurityDeclaration) error {
		if read.PrivateReporting == nil {
			return fmt.Errorf("reader unavailable")
		}

		observed, err := read.PrivateReporting(repo)
		if err != nil {
			return err
		}
		enabled, err := observed.Value()
		if err != nil {
			return err
		}
		candidate.PrivateVulnerabilityReporting = &enabled

		return nil
	})
	observe("repository.settings.general.releases.immutable_release", func(candidate *settings.SecurityDeclaration) error {
		if read.ImmutableReleases == nil {
			return fmt.Errorf("reader unavailable")
		}

		observed, err := read.ImmutableReleases(repo)
		if err != nil {
			return err
		}
		enabled, _, _, err := observed.Values()
		if err != nil {
			return err
		}
		candidate.ImmutableReleases = &enabled

		return nil
	})
	return result, diagnosticError
}
