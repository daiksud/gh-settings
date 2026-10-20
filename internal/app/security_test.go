package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func appSecurityBool(v bool) *bool { return &v }
func TestSecretScanningApplyOrderAndPartialProgress(t *testing.T) {
	for _, enable := range []bool{false, true} {
		for _, failure := range []int{-1, 0, 1} {
			d := settings.Declaration{Security: settings.SecurityDeclaration{SecretScanning: appSecurityBool(enable), PushProtection: appSecurityBool(enable)}}
			calls := []string{}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadSecretScanning: func(repository.Repository) (settings.SecretScanningObservation, error) {
				return settings.NewSecretScanningObservation(!enable, !enable), nil
			}, SecretScanning: func(_ repository.Repository, c settings.SecurityChange) (bool, error) {
				calls = append(calls, c.Field())
				if len(calls)-1 == failure {
					return false, errors.New("write or verification uncertain")
				}
				return true, nil
			}})
			first := "security.secret_scanning"
			if !enable {
				first = "security.secret_scanning_push_protection"
			}
			if len(calls) == 0 || calls[0] != first {
				t.Fatal("wrong dependency order", enable, calls)
			}
			want := ApplyCompleted
			if failure == 0 {
				want = ApplyFailed
			}
			if failure == 1 {
				want = ApplyPartialFailure
			}
			if result.Outcome != want || (err == nil) != (failure < 0) {
				t.Fatal(enable, failure, result, err)
			}
			if failure == 1 && (result.Operations[1].Status != OperationCompleted || result.Operations[2].Status != OperationFailedUnknown) {
				t.Fatal("earlier confirmed write lost", result)
			}
		}
	}
}
func TestSecretScanningReadFailureAndOmission(t *testing.T) {
	for _, failure := range []string{"read", "preflight", "pre-write", "matching", "concurrent matching"} {
		d := settings.Declaration{Security: settings.SecurityDeclaration{PushProtection: appSecurityBool(true)}}
		calls := 0
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadSecretScanning: func(repository.Repository) (settings.SecretScanningObservation, error) {
			if failure == "read" {
				return settings.SecretScanningObservation{}, errors.New("hidden")
			}
			return settings.NewSecretScanningObservation(failure != "preflight", failure == "matching"), nil
		}, SecretScanning: func(repository.Repository, settings.SecurityChange) (bool, error) {
			calls++
			if failure == "pre-write" {
				return false, &workflowReadFailure{errors.New("fresh prerequisite failed")}
			}
			return false, nil
		}})
		good := failure == "matching" || failure == "concurrent matching"
		if (err == nil) != good {
			t.Fatal(failure, result, err)
		}
		if !good && result.Outcome != ApplyFailed {
			t.Fatal("read-only progress counted as write", failure, result)
		}
		if failure == "matching" && calls != 0 {
			t.Fatal("matching state wrote")
		}
		if failure == "pre-write" && result.Operations[1].Status != OperationFailedRead {
			t.Fatal(result)
		}
	}
	invalid := settings.Declaration{Security: settings.SecurityDeclaration{SecretScanning: appSecurityBool(false), PushProtection: appSecurityBool(true)}}
	if _, err := Apply(func() (settings.Declaration, error) { return invalid, nil }, func() (repository.Repository, error) {
		t.Fatal("invalid local config resolved target")
		return repository.Repository{}, nil
	}, Writer{}); err == nil {
		t.Fatal("invalid config accepted")
	}
	if _, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{}, nil }, func() (repository.Repository, error) {
		t.Fatal("empty declaration resolved target")
		return repository.Repository{}, nil
	}, Reader{}); err != nil {
		t.Fatal(err)
	}
	d := settings.Declaration{Security: settings.SecurityDeclaration{SecretScanning: appSecurityBool(true)}}
	result, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Reader{SecretScanning: func(repository.Repository) (settings.SecretScanningObservation, error) {
		return settings.NewSecretScanningObservation(false, false), nil
	}})
	if err != nil || result.Comparison.Managed != 1 || len(result.Comparison.Security) != 1 {
		t.Fatal(result, err)
	}
}
