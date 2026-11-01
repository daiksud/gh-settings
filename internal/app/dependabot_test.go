package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func appDependabot(a, u bool) settings.DependabotObservation {
	ao := settings.NewDependabotAlerts404Observation()
	if a {
		ao = settings.NewEnabledDependabotAlertsObservation()
	}
	return settings.DependabotObservation{Alerts: ao, Updates: settings.NewDependabotSecurityUpdatesObservation(u, true)}
}

func TestDependabotExportComposesAndRequiresCompleteReads(t *testing.T) {
	for _, failure := range []string{"", "unacquired", "inconsistent", "later"} {
		t.Run(failure, func(t *testing.T) {
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reader.SecretScanning = func(repository.Repository) (settings.SecretScanningObservation, error) {
				return settings.NewSecretScanningObservation(false, false), nil
			}
			reader.Dependabot = func(repository.Repository) (settings.DependabotObservation, error) {
				if failure == "unacquired" {
					return settings.DependabotObservation{}, nil
				}
				return appDependabot(false, failure == "inconsistent"), nil
			}
			reader.PrivateReporting = func(repository.Repository) (settings.PrivateReportingObservation, error) {
				if failure == "later" {
					return settings.PrivateReportingObservation{}, errors.New("later read failed")
				}
				return settings.NewPrivateReportingObservation(true), nil
			}
			saved := false
			err := Export(ExportCheckout{IncludeDependabot: true, IncludeSecretScanning: true, IncludePrivateReporting: true, Repository: func() (repository.Repository, error) { return repository.Repository{}, nil }, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saved = true
					if d.Security.DependabotAlerts == nil || *d.Security.DependabotAlerts || d.Security.DependabotSecurityUpdates == nil || *d.Security.DependabotSecurityUpdates || d.Security.SecretScanning == nil || *d.Security.SecretScanning || d.Security.PrivateVulnerabilityReporting == nil || !*d.Security.PrivateVulnerabilityReporting {
						t.Fatal("security family overwritten or false omitted", d.Security)
					}
					return nil
				}, nil
			}}, reader)
			if (err == nil) != (failure == "") || saved != (failure == "") {
				t.Fatal(saved, err)
			}
		})
	}
}

func TestDependabotPlanApplyOrderingAndPartialAcceptance(t *testing.T) {
	for _, after := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			d := settings.Declaration{Security: settings.SecurityDeclaration{DependabotAlerts: appSecurityBool(after), DependabotSecurityUpdates: appSecurityBool(after)}}
			load := func() (settings.Declaration, error) { return d, nil }
			resolve := func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}
			read := func(repository.Repository) (settings.DependabotObservation, error) {
				return appDependabot(!after, !after), nil
			}
			plan, err := Plan(load, resolve, Reader{Dependabot: read})
			if err != nil || plan.Comparison.Managed != 2 || len(plan.Comparison.Security) != 2 {
				t.Fatal(plan, err)
			}
			var writes []string
			result, err := Apply(load, resolve, Writer{ReadDependabot: read, Dependabot: func(_ repository.Repository, c settings.SecurityChange) (settings.DependabotWriteResult, error) {
				writes = append(writes, c.Field())
				if fail && len(writes) == 2 {
					return settings.DependabotWriteResult{}, errors.New("later write failed")
				}
				return settings.DependabotWriteResult{Wrote: true, AcceptedDisable: !after}, nil
			}})
			want := []string{settings.DependabotAlertsField, settings.DependabotUpdatesField}
			if !after {
				want[0], want[1] = want[1], want[0]
			}
			if !reflect.DeepEqual(writes, want) || (err != nil) != fail {
				t.Fatal(result, writes, err)
			}
			status := OperationCompleted
			if !after {
				status = OperationAcceptedUnverified
			}
			if result.Operations[1].Status != status || !after && len(result.Notices) == 0 {
				t.Fatal(result)
			}
			if fail && result.Outcome != ApplyPartialFailure || !fail && !result.Completed() {
				t.Fatal(result)
			}
		}
	}
}

func TestDependabotValidationReadFailureOmissionAndEquality(t *testing.T) {
	resolve := func() (repository.Repository, error) { return repository.Repository{}, nil }
	for _, d := range []settings.SecurityDeclaration{{}, {DependabotAlerts: appSecurityBool(false)}, {DependabotAlerts: appSecurityBool(false), DependabotSecurityUpdates: appSecurityBool(true)}} {
		calls := 0
		load := func() (settings.Declaration, error) { return settings.Declaration{Security: d}, nil }
		read := func(repository.Repository) (settings.DependabotObservation, error) {
			calls++
			return appDependabot(false, false), nil
		}
		_, err := Plan(load, resolve, Reader{Dependabot: read})
		if d.Empty() && calls != 0 || d.DependabotSecurityUpdates != nil && (calls != 0 || err == nil) {
			t.Fatal(calls, err)
		}
		result, err := Apply(load, resolve, Writer{ReadDependabot: read, Dependabot: func(repository.Repository, settings.SecurityChange) (settings.DependabotWriteResult, error) {
			t.Fatal("matching, invalid or omitted intent wrote")
			return settings.DependabotWriteResult{}, nil
		}})
		if d.DependabotSecurityUpdates == nil && err != nil {
			t.Fatal(result, err)
		}
	}
	d := settings.Declaration{Security: settings.SecurityDeclaration{DependabotAlerts: appSecurityBool(true)}}
	result, err := Apply(func() (settings.Declaration, error) { return d, nil }, resolve, Writer{ReadDependabot: func(repository.Repository) (settings.DependabotObservation, error) {
		return settings.DependabotObservation{}, errors.New("read denied")
	}})
	if err == nil || result.Outcome != ApplyFailed || result.Operations[0].Status != OperationFailedRead {
		t.Fatal(result, err)
	}
}
