package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestImmutableReleasesIndependentPlanAndMatchingApply(t *testing.T) {
	d := settings.Declaration{Security: settings.SecurityDeclaration{ImmutableReleases: appSecurityBool(true)}}
	plan, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Reader{ImmutableReleases: func(repository.Repository) (settings.ImmutableReleasesObservation, error) {
		return settings.NewDisabledImmutableReleasesObservation(), nil
	}})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.Security) != 1 || plan.Comparison.Security[0].Field() != "security.immutable_releases" {
		t.Fatal(plan, err)
	}
	enabled := false
	writes := 0
	writer := Writer{ReadImmutableReleases: func(repository.Repository) (settings.ImmutableReleasesObservation, error) {
		return settings.NewImmutableReleasesObservation(enabled, false), nil
	}, ImmutableReleases: func(_ repository.Repository, c settings.SecurityChange) (bool, error) {
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
func TestImmutableReleasesFailureRetainsOnlyConfirmedWrites(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		for _, failure := range []string{"read", "pre-write", "write"} {
			d := settings.Declaration{Security: settings.SecurityDeclaration{SecretScanning: appSecurityBool(true), ImmutableReleases: appSecurityBool(true)}}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadSecretScanning: func(repository.Repository) (settings.SecretScanningObservation, error) {
				return settings.NewSecretScanningObservation(!earlier, false), nil
			}, SecretScanning: func(repository.Repository, settings.SecurityChange) (bool, error) { return true, nil }, ReadImmutableReleases: func(repository.Repository) (settings.ImmutableReleasesObservation, error) {
				if failure == "read" {
					return settings.ImmutableReleasesObservation{}, errors.New("policy unavailable")
				}
				return settings.NewDisabledImmutableReleasesObservation(), nil
			}, ImmutableReleases: func(repository.Repository, settings.SecurityChange) (bool, error) {
				if failure == "pre-write" {
					return false, &workflowReadFailure{errors.New("fresh policy read failed")}
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

func TestImmutableReleasesMatchingReadDoesNotInventEarlierWrite(t *testing.T) {
	for _, initiallyMatching := range []bool{false, true} {
		d := settings.Declaration{
			Security: settings.SecurityDeclaration{ImmutableReleases: appSecurityBool(true)},
			Pages:    settings.PagesDeclaration{CNAME: &settings.PagesDomain{}},
		}
		updates := 0
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
			ReadImmutableReleases: func(repository.Repository) (settings.ImmutableReleasesObservation, error) {
				return settings.NewImmutableReleasesObservation(initiallyMatching, false), nil
			},
			ImmutableReleases: func(repository.Repository, settings.SecurityChange) (bool, error) {
				updates++
				return false, nil // Fresh adapter state already matches; no toggle sent.
			},
			ReadPages: func(repository.Repository) (settings.PagesObservation, error) {
				return settings.PagesObservation{}, errors.New("later Pages read failed")
			},
		})
		if err == nil || result.Outcome != ApplyFailed || updates != map[bool]int{true: 0, false: 1}[initiallyMatching] {
			t.Fatal(initiallyMatching, result, updates, err)
		}
		if !initiallyMatching && result.Operations[1].Status != OperationUnchanged {
			t.Fatal("fresh no-op was not recorded as unchanged", result.Operations)
		}
	}
}
