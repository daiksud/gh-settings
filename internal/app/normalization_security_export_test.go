package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSecurityExportWarningsRequireCompleteSelectedObservations(t *testing.T) {
	for _, test := range []struct {
		name          string
		include, push bool
		failure       string
		controls      []settings.SecurityStatusControl
		wantWarnings  int
	}{
		{"both normalized", true, false, "", []settings.SecurityStatusControl{settings.PushProtectionControl, settings.SecretProtectionControl}, 2},
		{"canonical", true, false, "", nil, 0},
		{"unrequested", false, false, "", []settings.SecurityStatusControl{settings.SecretProtectionControl}, 0},
		{"failed with partial state", true, false, "read", []settings.SecurityStatusControl{settings.SecretProtectionControl}, 0},
		{"unacquired", true, false, "unacquired", nil, 0},
		{"inconsistent projected declaration", true, true, "inconsistent", []settings.SecurityStatusControl{settings.SecretProtectionControl}, 1},
		{"later failure", true, false, "later", []settings.SecurityStatusControl{settings.SecretProtectionControl}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var records []settings.SecurityStatusNormalization
			for _, control := range test.controls {
				records = append(records, settings.SecurityStatusNormalization{Path: control.Path(), RawValue: "", Fallback: false})
			}
			observed, err := settings.NewSecretScanningObservationWithNormalizations(false, test.push, records)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("acquisition denied")
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reads := 0
			reader.SecretScanning = func(repository.Repository) (settings.SecretScanningObservation, error) {
				reads++
				if test.failure == "read" {
					return observed, cause
				}
				if test.failure == "unacquired" {
					return settings.SecretScanningObservation{}, nil
				}
				return observed, nil
			}
			reader.PrivateReporting = func(repository.Repository) (settings.PrivateReportingObservation, error) {
				return settings.PrivateReportingObservation{}, cause
			}
			saved := false
			var warnings []settings.SecurityStatusNormalization
			err = Export(ExportCheckout{Repository: checkoutRepository, IncludeSecretScanning: test.include, IncludePrivateReporting: test.failure == "later", SecurityStatusNormalizationWarning: func(record settings.SecurityStatusNormalization) { warnings = append(warnings, record) }, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saved = true
					if test.include && (d.Security.SecretScanning == nil || *d.Security.SecretScanning || d.Security.PushProtection == nil || *d.Security.PushProtection) {
						t.Fatal("effective booleans lost", d.Security)
					}
					return nil
				}, nil
			}}, reader)
			wantReads := 0
			if test.include {
				wantReads = 1
			}
			if (err != nil) != (test.failure != "") || saved != (test.failure == "") || reads != wantReads || len(warnings) != test.wantWarnings {
				t.Fatal(saved, reads, warnings, err)
			}
			if test.failure == "read" || test.failure == "later" {
				if !errors.Is(err, cause) {
					t.Fatal("original failure lost", err)
				}
			}
			if test.wantWarnings > 0 && !reflect.DeepEqual(warnings, observed.Normalizations()) {
				t.Fatal("warning ownership/order lost", warnings, observed.Normalizations())
			}
		})
	}
}
