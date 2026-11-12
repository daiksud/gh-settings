package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestAutomaticSecurityExportWarningsRequireCompleteObservations(t *testing.T) {
	for _, failure := range []string{"", "read", "unacquired", "inconsistent"} {
		t.Run(failure, func(t *testing.T) {
			push := failure == "inconsistent"
			o, err := settings.NewSecretScanningObservationWithNormalizations(false, push, []settings.SecurityStatusNormalization{{Path: settings.SecretProtectionControl.Path(), RawValue: "", Fallback: false}})
			if err != nil {
				t.Fatal(err)
			}
			r := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			r.SecretScanning = func(repository.Repository) (settings.SecretScanningObservation, error) {
				if failure == "read" {
					return o, errors.New("denied")
				}
				if failure == "unacquired" {
					return settings.SecretScanningObservation{}, nil
				}
				return o, nil
			}
			saved, warnings := false, 0
			err = Export(ExportCheckout{Repository: checkoutRepository, SecurityStatusNormalizationWarning: func(settings.SecurityStatusNormalization) { warnings++ }, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saved = true
					if (failure == "") != (d.Security.SecretScanning != nil) {
						t.Fatal(d.Security)
					}
					return nil
				}, nil
			}}, r)
			want := 0
			if failure == "" || failure == "inconsistent" {
				want = 1
			}
			if err != nil || !saved || warnings != want {
				t.Fatal(err, saved, warnings, want)
			}
		})
	}
}
