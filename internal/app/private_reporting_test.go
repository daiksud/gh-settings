package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPrivateReportingIndependentPlanAndMatchingApply(t *testing.T) {
	d := settings.Declaration{Security: settings.SecurityDeclaration{PrivateVulnerabilityReporting: appSecurityBool(true)}}
	plan, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Reader{PrivateReporting: func(repository.Repository) (settings.PrivateReportingObservation, error) {
		return settings.NewPrivateReportingObservation(false), nil
	}})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.Security) != 1 || plan.Comparison.Security[0].Field() != "security.private_vulnerability_reporting" {
		t.Fatal(plan, err)
	}
	enabled := false
	writes := 0
	writer := Writer{ReadPrivateReporting: func(repository.Repository) (settings.PrivateReportingObservation, error) {
		return settings.NewPrivateReportingObservation(enabled), nil
	}, PrivateReporting: func(_ repository.Repository, c settings.SecurityChange) (bool, error) {
		writes++
		_, enabled = c.Values()
		return true, nil
	}}
	for i := 0; i < 2; i++ {
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, writer)
		if err != nil || !result.Completed() {
			t.Fatal(i, result, err)
		}
	}
	if writes != 1 || !enabled {
		t.Fatal(writes, enabled)
	}
}
func TestPrivateReportingFailureRetainsOnlyConfirmedWrites(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		for _, failure := range []string{"read", "pre-write", "write"} {
			d := settings.Declaration{Security: settings.SecurityDeclaration{SecretScanning: appSecurityBool(true), PrivateVulnerabilityReporting: appSecurityBool(true)}}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadSecretScanning: func(repository.Repository) (settings.SecretScanningObservation, error) {
				return settings.NewSecretScanningObservation(!earlier, false), nil
			}, SecretScanning: func(repository.Repository, settings.SecurityChange) (bool, error) { return true, nil }, ReadPrivateReporting: func(repository.Repository) (settings.PrivateReportingObservation, error) {
				if failure == "read" {
					return settings.PrivateReportingObservation{}, errors.New("visibility/status unavailable")
				}
				return settings.NewPrivateReportingObservation(false), nil
			}, PrivateReporting: func(repository.Repository, settings.SecurityChange) (bool, error) {
				if failure == "pre-write" {
					return false, &workflowReadFailure{errors.New("fresh target is private")}
				}
				return false, errors.New("write/readback uncertain")
			}})
			want := ApplyFailed
			if earlier {
				want = ApplyPartialFailure
			}
			status := OperationFailedUnknown
			if failure != "write" {
				status = OperationFailedRead
			}
			if err == nil || result.Outcome != want || result.Operations[len(result.Operations)-1].Status != status {
				t.Fatal(earlier, failure, result, err)
			}
		}
	}
}
