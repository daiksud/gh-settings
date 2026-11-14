package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedSecurityUsesSelectedCompleteObservations(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "repo"}
	no, yes := false, true
	query := "default"
	config := settings.CodeQLConfigurationState{Languages: []string{"go"}, QuerySuite: "default", Runner: settings.CodeQLRunner{Type: "standard_github_runner"}, ThreatModel: "remote_sources"}
	observation, err := settings.NewCodeQLConfigurationObservation(config)
	if err != nil {
		t.Fatal(err)
	}
	base := Writer{
		ReadDependabot: func(repository.Repository) (settings.DependabotObservation, error) {
			return settings.DependabotObservation{Alerts: settings.NewDependabotAlerts404Observation(), Updates: settings.NewDependabotSecurityUpdates404Observation()}, nil
		},
		ReadSecretScanning: func(repository.Repository) (settings.SecretScanningObservation, error) {
			return settings.NewSecretScanningObservation(false, false), nil
		},
		ReadCodeQLConfiguration: func(repository.Repository) (settings.CodeQLConfigurationObservation, error) { return observation, nil },
		ReadCodeQLDefaultSetup: func(repository.Repository) (settings.CodeQLDefaultSetupObservation, error) {
			return settings.NewCodeQLDefaultSetupObservation(false), nil
		},
		ReadAIScanRepositories: func(repository.Repository) (settings.AIScanRepositoriesObservation, error) {
			return settings.NewAIScanRepositoriesObservation(false), nil
		},
		ReadAIScanPullRequests: func(repository.Repository) (settings.AIScanPullRequestsObservation, error) {
			return settings.NewAIScanPullRequestsObservation(false), nil
		},
		ReadPrivateReporting: func(repository.Repository) (settings.PrivateReportingObservation, error) {
			return settings.NewPrivateReportingObservation(false), nil
		},
		ReadImmutableReleases: func(repository.Repository) (settings.ImmutableReleasesObservation, error) {
			return settings.NewDisabledImmutableReleasesObservation(), nil
		},
	}
	tests := []struct {
		name       string
		desired    settings.SecurityDeclaration
		missing    func(*Writer)
		unacquired func(*Writer)
	}{
		{"dependabot", settings.SecurityDeclaration{DependabotAlerts: &no, DependabotSecurityUpdates: &no}, func(w *Writer) { w.ReadDependabot = nil }, func(w *Writer) {
			w.ReadDependabot = func(repository.Repository) (settings.DependabotObservation, error) {
				return settings.DependabotObservation{}, nil
			}
		}},
		{"secret scanning", settings.SecurityDeclaration{SecretScanning: &no, PushProtection: &no}, func(w *Writer) { w.ReadSecretScanning = nil }, func(w *Writer) {
			w.ReadSecretScanning = func(repository.Repository) (settings.SecretScanningObservation, error) {
				return settings.SecretScanningObservation{}, nil
			}
		}},
		{"codeql configuration", settings.SecurityDeclaration{CodeQLConfiguration: settings.CodeQLConfigurationDeclaration{QuerySuite: &query}}, func(w *Writer) { w.ReadCodeQLConfiguration = nil }, func(w *Writer) {
			w.ReadCodeQLConfiguration = func(repository.Repository) (settings.CodeQLConfigurationObservation, error) {
				return settings.CodeQLConfigurationObservation{}, nil
			}
		}},
		{"codeql default", settings.SecurityDeclaration{CodeQLDefaultSetup: &no}, func(w *Writer) { w.ReadCodeQLDefaultSetup = nil }, func(w *Writer) {
			w.ReadCodeQLDefaultSetup = func(repository.Repository) (settings.CodeQLDefaultSetupObservation, error) {
				return settings.CodeQLDefaultSetupObservation{}, nil
			}
		}},
		{"ai repositories", settings.SecurityDeclaration{AIScanRepositories: &no}, func(w *Writer) { w.ReadAIScanRepositories = nil }, func(w *Writer) {
			w.ReadAIScanRepositories = func(repository.Repository) (settings.AIScanRepositoriesObservation, error) {
				return settings.AIScanRepositoriesObservation{}, nil
			}
		}},
		{"ai pull requests", settings.SecurityDeclaration{AIScanPullRequests: &no}, func(w *Writer) { w.ReadAIScanPullRequests = nil }, func(w *Writer) {
			w.ReadAIScanPullRequests = func(repository.Repository) (settings.AIScanPullRequestsObservation, error) {
				return settings.AIScanPullRequestsObservation{}, nil
			}
		}},
		{"private reporting", settings.SecurityDeclaration{PrivateVulnerabilityReporting: &no}, func(w *Writer) { w.ReadPrivateReporting = nil }, func(w *Writer) {
			w.ReadPrivateReporting = func(repository.Repository) (settings.PrivateReportingObservation, error) {
				return settings.PrivateReportingObservation{}, nil
			}
		}},
		{"immutable releases", settings.SecurityDeclaration{ImmutableReleases: &no}, func(w *Writer) { w.ReadImmutableReleases = nil }, func(w *Writer) {
			w.ReadImmutableReleases = func(repository.Repository) (settings.ImmutableReleasesObservation, error) {
				return settings.ImmutableReleasesObservation{}, nil
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := verifyArchivedSecurity(repo, tc.desired, base); err != nil {
				t.Fatal(err)
			}
			w := base
			tc.missing(&w)
			if err := verifyArchivedSecurity(repo, tc.desired, w); err == nil {
				t.Fatal("missing reader accepted")
			}
			w = base
			tc.unacquired(&w)
			if err := verifyArchivedSecurity(repo, tc.desired, w); err == nil {
				t.Fatal("unacquired observation accepted")
			}
			readers := map[string]string{
				"dependabot": "ReadDependabot", "secret scanning": "ReadSecretScanning",
				"codeql configuration": "ReadCodeQLConfiguration", "codeql default": "ReadCodeQLDefaultSetup",
				"ai repositories": "ReadAIScanRepositories", "ai pull requests": "ReadAIScanPullRequests",
				"private reporting": "ReadPrivateReporting", "immutable releases": "ReadImmutableReleases",
			}
			w = base
			field := reflect.ValueOf(&w).Elem().FieldByName(readers[tc.name])
			denied := errors.New("denied")
			field.Set(reflect.MakeFunc(field.Type(), func([]reflect.Value) []reflect.Value {
				return []reflect.Value{reflect.Zero(field.Type().Out(0)), reflect.ValueOf(denied)}
			}))
			if err := verifyArchivedSecurity(repo, tc.desired, w); !errors.Is(err, denied) {
				t.Fatalf("reader error not retained: %v", err)
			}
		})
	}
	for _, d := range []settings.SecurityDeclaration{
		{DependabotAlerts: &yes}, {SecretScanning: &yes}, {CodeQLDefaultSetup: &yes}, {AIScanRepositories: &yes}, {AIScanPullRequests: &yes}, {PrivateVulnerabilityReporting: &yes}, {ImmutableReleases: &yes},
	} {
		if err := verifyArchivedSecurity(repo, d, base); err == nil {
			t.Fatalf("mismatch accepted: %+v", d)
		}
	}
	extended := "extended"
	if err := verifyArchivedSecurity(repo, settings.SecurityDeclaration{CodeQLConfiguration: settings.CodeQLConfigurationDeclaration{QuerySuite: &extended}}, base); err == nil {
		t.Fatal("different CodeQL configuration accepted")
	}
	unknown, err := settings.NewSecretScanningObservationWithNormalizations(false, false, []settings.SecurityStatusNormalization{{Path: settings.SecretProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}})
	if err != nil {
		t.Fatal(err)
	}
	unknownWriter := base
	unknownWriter.ReadSecretScanning = func(repository.Repository) (settings.SecretScanningObservation, error) { return unknown, nil }
	if err := verifyArchivedSecurity(repo, settings.SecurityDeclaration{SecretScanning: &no}, unknownWriter); err == nil {
		t.Fatal("normalized unknown status accepted as exact")
	}
	denied := errors.New("denied")
	w := base
	w.ReadSecretScanning = func(repository.Repository) (settings.SecretScanningObservation, error) {
		return settings.SecretScanningObservation{}, denied
	}
	if err := verifyArchivedSecurity(repo, settings.SecurityDeclaration{SecretScanning: &no}, w); !errors.Is(err, denied) {
		t.Fatal(err)
	}
}

func TestArchivedSecurityRejectsComparatorWithoutSelectedEvidence(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "repo"}
	read := func(repository.Repository) (settings.SecretScanningObservation, error) {
		return settings.NewSecretScanningObservation(false, false), nil
	}
	miswired := func(settings.SecretScanningObservation) (int, []settings.SecurityChange, error) {
		return 0, nil, nil
	}
	if err := matchArchivedSecurity(repo, "secret scanning", read, miswired); err == nil {
		t.Fatal("a comparator that checked no selected control was accepted")
	}
}
