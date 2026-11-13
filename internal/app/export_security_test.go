package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSecurityExportFirstExportAndReexportKeepUnknownDistinctFromFalse(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "first", true: "existing"}[existing], func(t *testing.T) {
			old := settings.SecurityDeclaration{}
			if existing {
				old = settings.SecurityDeclaration{SecretScanning: appSecurityBool(true), PushProtection: appSecurityBool(true), PrivateVulnerabilityReporting: appSecurityBool(true), AIScanRepositories: appSecurityBool(true)}
			}
			calls := []string{}
			infos := []string{}
			r := Reader{
				SecretScanning: func(repository.Repository) (settings.SecretScanningObservation, error) {
					calls = append(calls, "secret")
					return settings.SecretScanningObservation{}, errors.New("sensitive response or token")
				},
				Dependabot: func(repository.Repository) (settings.DependabotObservation, error) {
					calls = append(calls, "dependabot")
					return appDependabot(false, false), nil
				},
				CodeQLForExport: func(repository.Repository) (settings.CodeQLDefaultSetupObservation, settings.CodeQLConfigurationObservation, error) {
					calls = append(calls, "codeql")
					return settings.NewCodeQLDefaultSetupObservation(false), settings.CodeQLConfigurationObservation{}, nil
				},
				AIScanPullRequests: func(repository.Repository) (settings.AIScanPullRequestsObservation, error) {
					calls = append(calls, "ai")
					return settings.NewAIScanPullRequestsObservation(false), nil
				},
				PrivateReporting: func(repository.Repository) (settings.PrivateReportingObservation, error) {
					calls = append(calls, "private")
					return settings.PrivateReportingObservation{}, errors.New("403")
				},
				ImmutableReleases: func(repository.Repository) (settings.ImmutableReleasesObservation, error) {
					calls = append(calls, "immutable")
					return settings.NewImmutableReleasesObservation(true, false), nil
				},
			}
			d, err := exportSecurity(ExportCheckout{Existing: settings.Declaration{Security: old}, Info: func(path string, _ error) error { infos = append(infos, path); return nil }}, r, repository.Repository{})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []string{"secret", "dependabot", "codeql", "ai", "private", "immutable"}) || len(infos) != 3 {
				t.Fatal(calls, infos)
			}
			if existing {
				if d.SecretScanning == nil || !*d.SecretScanning || d.PushProtection == nil || !*d.PushProtection || d.PrivateVulnerabilityReporting == nil || !*d.PrivateVulnerabilityReporting || d.AIScanRepositories == nil || !*d.AIScanRepositories {
					t.Fatal(d)
				}
			} else if d.SecretScanning != nil || d.PushProtection != nil || d.PrivateVulnerabilityReporting != nil || d.AIScanRepositories != nil {
				t.Fatal(d)
			}
			if d.DependabotAlerts == nil || *d.DependabotAlerts || d.DependabotSecurityUpdates == nil || *d.DependabotSecurityUpdates || d.CodeQLDefaultSetup == nil || *d.CodeQLDefaultSetup || d.AIScanPullRequests == nil || *d.AIScanPullRequests || d.ImmutableReleases == nil || !*d.ImmutableReleases {
				t.Fatal(d)
			}
			if err := d.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestSecurityExportCodeQLKnownAndUnknown(t *testing.T) {
	languages := []string{"go"}
	suite := "default"
	threat := "remote_and_local_sources"
	config := settings.CodeQLConfigurationDeclaration{Languages: &languages, QuerySuite: &suite, ThreatModel: &threat}
	for _, prior := range []string{"first", "state-only", "details"} {
		for _, state := range []string{"disabled", "enabled", "unknown", "enabled-unreadable-details"} {
			t.Run(prior+"/"+state, func(t *testing.T) {
				old := settings.SecurityDeclaration{}
				if prior != "first" {
					old.CodeQLDefaultSetup = appSecurityBool(true)
				}
				if prior == "details" {
					old.CodeQLConfiguration = config
				}
				continued := false
				disabledInfo := false
				d, err := exportSecurity(ExportCheckout{Existing: settings.Declaration{Security: old}, Info: func(_ string, err error) error {
					disabledInfo = disabledInfo || errors.Is(err, ErrExportCodeQLDisabled)
					return nil
				}}, Reader{
					CodeQLForExport: func(repository.Repository) (settings.CodeQLDefaultSetupObservation, settings.CodeQLConfigurationObservation, error) {
						if state == "unknown" {
							return settings.CodeQLDefaultSetupObservation{}, settings.CodeQLConfigurationObservation{}, errors.New("unavailable")
						}
						setup := settings.NewCodeQLDefaultSetupObservation(state != "disabled")
						if state == "enabled" {
							observed, err := settings.NewCodeQLConfigurationObservation(settings.CodeQLConfigurationState{Languages: []string{"python"}, QuerySuite: "extended", Runner: settings.CodeQLRunner{Type: "standard_github_runner"}, ThreatModel: "remote_sources"})
							return setup, observed, err
						}
						if state == "enabled-unreadable-details" {
							return setup, settings.CodeQLConfigurationObservation{}, errors.New("incomplete details")
						}
						return setup, settings.CodeQLConfigurationObservation{}, nil
					},
					ImmutableReleases: func(repository.Repository) (settings.ImmutableReleasesObservation, error) {
						continued = true
						return settings.NewImmutableReleasesObservation(true, false), nil
					},
				}, repository.Repository{})
				if err != nil || !continued || d.Validate() != nil {
					t.Fatal(d, err, continued)
				}
				switch state {
				case "disabled":
					if d.CodeQLDefaultSetup == nil || *d.CodeQLDefaultSetup || !d.CodeQLConfiguration.Empty() || disabledInfo != (prior == "details") {
						t.Fatal(d, disabledInfo)
					}
				case "enabled":
					if d.CodeQLDefaultSetup == nil || !*d.CodeQLDefaultSetup || d.CodeQLConfiguration.Languages == nil || !reflect.DeepEqual(*d.CodeQLConfiguration.Languages, []string{"python"}) || d.CodeQLConfiguration.ThreatModel == nil || *d.CodeQLConfiguration.ThreatModel != "remote_sources" {
						t.Fatal(d)
					}
				case "unknown":
					if !reflect.DeepEqual(d.CodeQLDefaultSetup, old.CodeQLDefaultSetup) || !reflect.DeepEqual(d.CodeQLConfiguration, old.CodeQLConfiguration) {
						t.Fatal(d, old)
					}
				case "enabled-unreadable-details":
					if d.CodeQLDefaultSetup == nil || !*d.CodeQLDefaultSetup || !reflect.DeepEqual(d.CodeQLConfiguration, old.CodeQLConfiguration) {
						t.Fatal(d, old)
					}
				}
			})
		}
	}
}

func TestSecurityExportInfoWriteFailurePreventsSave(t *testing.T) {
	cause := errors.New("diagnostic output unavailable")
	saved := false
	r := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
	err := Export(ExportCheckout{Repository: checkoutRepository, Info: func(string, error) error { return cause }, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(settings.Declaration) error { saved = true; return nil }, nil
	}}, r)
	if !errors.Is(err, cause) || saved {
		t.Fatal(err, saved)
	}
}
