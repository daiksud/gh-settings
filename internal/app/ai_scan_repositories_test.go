package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestAIScanRepositoriesIndependentPlanAndMatchingApply(t *testing.T) {
	d := settings.Declaration{Security: settings.SecurityDeclaration{AIScanRepositories: appSecurityBool(true)}}
	plan, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Reader{AIScanRepositories: func(repository.Repository) (settings.AIScanRepositoriesObservation, error) {
		return settings.NewAIScanRepositoriesObservation(false), nil
	}})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.Security) != 1 || plan.Comparison.Security[0].Field() != "security.ai_scan_repositories" {
		t.Fatal(plan, err)
	}
	enabled := false
	writes := 0
	writer := Writer{ReadAIScanRepositories: func(repository.Repository) (settings.AIScanRepositoriesObservation, error) {
		return settings.NewAIScanRepositoriesObservation(enabled), nil
	}, AIScanRepositories: func(_ repository.Repository, c settings.SecurityChange) (bool, error) {
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
func TestAIScanRepositoriesFailureRetainsOnlyConfirmedWrites(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		for _, failure := range []string{"read", "pre-write", "write"} {
			d := settings.Declaration{Security: settings.SecurityDeclaration{SecretScanning: appSecurityBool(true), AIScanRepositories: appSecurityBool(true)}}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadSecretScanning: func(repository.Repository) (settings.SecretScanningObservation, error) {
				return settings.NewSecretScanningObservation(!earlier, false), nil
			}, SecretScanning: func(repository.Repository, settings.SecurityChange) (bool, error) { return true, nil }, ReadAIScanRepositories: func(repository.Repository) (settings.AIScanRepositoriesObservation, error) {
				if failure == "read" {
					return settings.AIScanRepositoriesObservation{}, errors.New("visibility/status unavailable")
				}
				return settings.NewAIScanRepositoriesObservation(false), nil
			}, AIScanRepositories: func(repository.Repository, settings.SecurityChange) (bool, error) {
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
