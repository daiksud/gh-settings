package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSecretScanningRawEqualityRequiresOriginalPair(t *testing.T) {
	normalized := func(scanning, push bool, control settings.SecurityStatusControl, raw string) settings.SecretScanningObservation {
		t.Helper()
		observed, err := settings.NewSecretScanningObservationWithNormalizations(scanning, push, []settings.SecurityStatusNormalization{{Path: control.Path(), RawValue: raw, Fallback: false}})
		if err != nil {
			t.Fatal(err)
		}
		return observed
	}
	canonical := settings.NewSecretScanningObservation(false, false)
	unknownScanning := normalized(false, false, settings.SecretProtectionControl, "FUTURE")
	unknownCompanion := normalized(true, false, settings.PushProtectionControl, "")
	for _, test := range []struct {
		expected, actual settings.SecretScanningObservation
		equal, wantErr   bool
	}{
		{canonical, canonical, true, false},
		{unknownScanning, unknownScanning, true, false},
		{canonical, unknownScanning, false, false},
		{unknownScanning, canonical, false, false},
		{unknownCompanion, unknownCompanion, true, false},
		{unknownCompanion, normalized(true, false, settings.PushProtectionControl, "CHANGED"), false, false},
		{unknownCompanion, settings.NewSecretScanningObservation(true, false), false, false},
		{unknownCompanion, normalized(false, false, settings.PushProtectionControl, ""), false, false},
		{settings.SecretScanningObservation{}, canonical, false, true},
		{canonical, settings.SecretScanningObservation{}, false, true},
	} {
		equal, err := sameRawSecretScanningStatuses(test.expected, test.actual)
		if equal != test.equal || (err != nil) != test.wantErr {
			t.Fatal(test, equal, err)
		}
	}
}
