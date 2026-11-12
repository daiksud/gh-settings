package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSecurityExportFirstExportAndReexportKeepUnknownDistinctFromFalse(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "first", true: "existing"}[existing], func(t *testing.T) {
			old := settings.SecurityDeclaration{}
			if existing {
				old = settings.SecurityDeclaration{SecretScanning: appSecurityBool(true), PushProtection: appSecurityBool(true), PrivateVulnerabilityReporting: appSecurityBool(true)}
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
				CodeQLConfiguration: func(repository.Repository) (settings.CodeQLConfigurationObservation, error) {
					calls = append(calls, "config")
					return settings.CodeQLConfigurationObservation{}, nil
				},
				CodeQLDefaultSetup: func(repository.Repository) (settings.CodeQLDefaultSetupObservation, error) {
					calls = append(calls, "setup")
					return settings.NewCodeQLDefaultSetupObservation(false), nil
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
			if !reflect.DeepEqual(calls, []string{"secret", "dependabot", "config", "setup", "ai", "private", "immutable"}) || len(infos) != 3 {
				t.Fatal(calls, infos)
			}
			if existing {
				if d.SecretScanning == nil || !*d.SecretScanning || d.PushProtection == nil || !*d.PushProtection || d.PrivateVulnerabilityReporting == nil || !*d.PrivateVulnerabilityReporting {
					t.Fatal(d)
				}
			} else if d.SecretScanning != nil || d.PushProtection != nil || d.PrivateVulnerabilityReporting != nil {
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
func TestSecurityExportRetainsCoherentCodeQLPairOnConflictingRead(t *testing.T) {
	languages := []string{"go"}
	suite := "default"
	old := settings.SecurityDeclaration{CodeQLDefaultSetup: appSecurityBool(true), CodeQLConfiguration: settings.CodeQLConfigurationDeclaration{Languages: &languages, QuerySuite: &suite}}
	var infos []string
	d, err := exportSecurity(ExportCheckout{Existing: settings.Declaration{Security: old}, Info: func(path string, _ error) error { infos = append(infos, path); return nil }}, Reader{CodeQLDefaultSetup: func(repository.Repository) (settings.CodeQLDefaultSetupObservation, error) {
		return settings.NewCodeQLDefaultSetupObservation(false), nil
	}}, repository.Repository{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d, old) || d.Validate() != nil {
		t.Fatal(d)
	}
	found := false
	for _, path := range infos {
		found = found || strings.HasSuffix(path, "codeql_analysis")
	}
	if !found {
		t.Fatal(infos)
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
