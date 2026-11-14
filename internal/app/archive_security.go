package app

import (
	"fmt"
	"reflect"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// verifyArchivedSecurity checks the same selected controls as apply, without
// invoking a writer. The caller must also guard declaration schema changes.
func verifyArchivedSecurity(repo repository.Repository, desired settings.SecurityDeclaration, read Writer) error {
	remaining := desired
	if !desired.DependabotEmpty() {
		if err := matchArchivedSecurity(repo, "Dependabot", read.ReadDependabot, func(o settings.DependabotObservation) (int, []settings.SecurityChange, error) {
			return settings.DependabotTransitions(o, desired)
		}); err != nil {
			return err
		}
		remaining.DependabotAlerts, remaining.DependabotSecurityUpdates = nil, nil
	}
	if !desired.SecretScanningEmpty() {
		if err := matchArchivedSecurity(repo, "Secret scanning", read.ReadSecretScanning, func(o settings.SecretScanningObservation) (int, []settings.SecurityChange, error) {
			managed, _, err := settings.CompareSecretScanning(o, desired)
			if err != nil {
				return 0, nil, err
			}
			changes, err := settings.SecretScanningTransitions(o, desired)
			return managed, changes, err
		}); err != nil {
			return err
		}
		remaining.SecretScanning, remaining.PushProtection = nil, nil
	}
	if !desired.CodeQLConfiguration.Empty() {
		if err := matchArchivedSecurity(repo, "CodeQL configuration", read.ReadCodeQLConfiguration, func(o settings.CodeQLConfigurationObservation) (int, []settings.CodeQLConfigurationChange, error) {
			return settings.CompareCodeQLConfiguration(o, desired.CodeQLConfiguration)
		}); err != nil {
			return err
		}
		remaining.CodeQLConfiguration = settings.CodeQLConfigurationDeclaration{}
	}
	if desired.CodeQLDefaultSetup != nil {
		if err := matchArchivedSecurity(repo, "CodeQL default setup", read.ReadCodeQLDefaultSetup, func(o settings.CodeQLDefaultSetupObservation) (int, []settings.SecurityChange, error) {
			return settings.CompareCodeQLDefaultSetup(o, desired)
		}); err != nil {
			return err
		}
		remaining.CodeQLDefaultSetup = nil
	}
	if desired.AIScanRepositories != nil {
		if err := matchArchivedSecurity(repo, "AI Scan repositories", read.ReadAIScanRepositories, func(o settings.AIScanRepositoriesObservation) (int, []settings.SecurityChange, error) {
			return settings.CompareAIScanRepositories(o, desired)
		}); err != nil {
			return err
		}
		remaining.AIScanRepositories = nil
	}
	if desired.AIScanPullRequests != nil {
		if err := matchArchivedSecurity(repo, "AI Scan pull requests", read.ReadAIScanPullRequests, func(o settings.AIScanPullRequestsObservation) (int, []settings.SecurityChange, error) {
			return settings.CompareAIScanPullRequests(o, desired)
		}); err != nil {
			return err
		}
		remaining.AIScanPullRequests = nil
	}
	if desired.PrivateVulnerabilityReporting != nil {
		if err := matchArchivedSecurity(repo, "private vulnerability reporting", read.ReadPrivateReporting, func(o settings.PrivateReportingObservation) (int, []settings.SecurityChange, error) {
			return settings.ComparePrivateReporting(o, desired)
		}); err != nil {
			return err
		}
		remaining.PrivateVulnerabilityReporting = nil
	}
	if desired.ImmutableReleases != nil {
		if err := matchArchivedSecurity(repo, "immutable releases", read.ReadImmutableReleases, func(o settings.ImmutableReleasesObservation) (int, []settings.SecurityChange, error) {
			return settings.CompareImmutableReleases(o, desired)
		}); err != nil {
			return err
		}
		remaining.ImmutableReleases = nil
	}
	if !reflect.DeepEqual(remaining, settings.SecurityDeclaration{}) {
		return fmt.Errorf("unsupported archived security declaration")
	}
	return nil
}

func matchArchivedSecurity[Observation any, Change any](repo repository.Repository, name string, read func(repository.Repository) (Observation, error), compare func(Observation) (int, []Change, error)) error {
	if read == nil {
		return fmt.Errorf("verify archived %s: reader unavailable", name)
	}
	observed, err := read(repo)
	if err != nil {
		return fmt.Errorf("verify archived %s: %w", name, err)
	}
	managed, changes, err := compare(observed)
	if err != nil {
		return fmt.Errorf("verify archived %s: %w", name, err)
	}
	if managed == 0 || len(changes) != 0 {
		return fmt.Errorf("archived %s differs or lacks selected evidence", name)
	}
	return nil
}
