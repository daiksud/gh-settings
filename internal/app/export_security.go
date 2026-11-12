package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ErrExportCodeQLConflict is a bounded diagnostic category, not upstream text.
var ErrExportCodeQLConflict = errors.New("observed and retained CodeQL declarations conflict")

// Acquisition and conversion are atomic within each independent family. A failed
// family never becomes false or an empty observed value, including on re-export.
func exportSecurity(checkout ExportCheckout, read Reader, repo repository.Repository) (settings.SecurityDeclaration, error) {
	result := checkout.Existing.Security
	var infoError error
	observe := func(path string, observe func(*settings.SecurityDeclaration) error) {
		candidate := result
		if err := observe(&candidate); err != nil {
			if checkout.Info != nil {
				infoError = errors.Join(infoError, checkout.Info(path, err))
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
				checkout.SecurityStatusNormalizationWarning(record)
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
		if read.CodeQLConfiguration == nil {
			return fmt.Errorf("reader unavailable")
		}

		o, err := read.CodeQLConfiguration(repo)
		if err != nil {
			return err
		}
		d, err := o.Declaration()
		if err != nil {
			return err
		}
		candidate.CodeQLConfiguration = d

		return nil
	})
	observe("repository.settings.advanced_security.code_security.codeql_analysis.default_setup", func(candidate *settings.SecurityDeclaration) error {
		if read.CodeQLDefaultSetup == nil {
			return fmt.Errorf("reader unavailable")
		}

		observed, err := read.CodeQLDefaultSetup(repo)
		if err != nil {
			return err
		}
		enabled, err := observed.Value()
		if err != nil {
			return err
		}
		candidate.CodeQLDefaultSetup = &enabled

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
	// The schema cannot express disabled CodeQL setup together with configuration.
	// Preserve a coherent prior pair rather than erase an unavailable declaration.
	if result.CodeQLDefaultSetup != nil && !*result.CodeQLDefaultSetup && !result.CodeQLConfiguration.Empty() {
		result.CodeQLDefaultSetup = checkout.Existing.Security.CodeQLDefaultSetup
		result.CodeQLConfiguration = checkout.Existing.Security.CodeQLConfiguration
		if checkout.Info != nil {
			infoError = errors.Join(infoError, checkout.Info("repository.settings.advanced_security.code_security.codeql_analysis", ErrExportCodeQLConflict))
		}
	}
	return result, infoError
}
