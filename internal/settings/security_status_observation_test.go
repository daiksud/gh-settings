package settings

import (
	"reflect"
	"testing"
)

func TestSecurityStatusObservationRetainsOwnedRawEvidence(t *testing.T) {
	scanning := SecurityStatusNormalization{Path: "repository.settings.advanced_security.secret_protection.enabled", RawValue: "FUTURE", Fallback: false}
	push := SecurityStatusNormalization{Path: "repository.settings.advanced_security.secret_protection.push_protection", RawValue: "", Fallback: false}
	records := []SecurityStatusNormalization{push, scanning}
	observed, err := NewSecretScanningObservationWithNormalizations(false, false, records)
	if err != nil {
		t.Fatal(err)
	}
	if got := observed.Normalizations(); !reflect.DeepEqual(got, []SecurityStatusNormalization{scanning, push}) {
		t.Fatal(got)
	}
	s, p, err := observed.RawStatuses()
	if err != nil || s != "FUTURE" || p != "" {
		t.Fatal(s, p, err)
	}
	records[0].RawValue = "mutated input"
	owned := observed.Normalizations()
	owned[0].RawValue = "mutated output"
	if !reflect.DeepEqual(observed.Normalizations(), []SecurityStatusNormalization{scanning, push}) {
		t.Fatal("aliased metadata")
	}
	declaration, err := observed.Declaration()
	if err != nil || declaration.SecretScanning == nil || *declaration.SecretScanning || declaration.PushProtection == nil || *declaration.PushProtection {
		t.Fatal(declaration, err)
	}
	for _, values := range [][2]bool{{false, false}, {false, true}, {true, false}, {true, true}} {
		canonical := NewSecretScanningObservation(values[0], values[1])
		s, p, err := canonical.RawStatuses()
		if err != nil || s != SecurityStatusForValue(values[0]) || p != SecurityStatusForValue(values[1]) || len(canonical.Normalizations()) != 0 {
			t.Fatal(canonical, s, p, err)
		}
		copy, err := NewSecretScanningObservationWithNormalizations(values[0], values[1], nil)
		if err != nil || copy != canonical {
			t.Fatal(copy, canonical, err)
		}
	}
	inconsistent, err := NewSecretScanningObservationWithNormalizations(false, true, []SecurityStatusNormalization{scanning})
	if err != nil {
		t.Fatal("acquisition enforced declaration dependency", err)
	}
	if _, err := inconsistent.Declaration(); err == nil {
		t.Fatal("inconsistent projected declaration repaired")
	}
	if _, _, err := (SecretScanningObservation{}).RawStatuses(); err == nil {
		t.Fatal("unacquired raw statuses available")
	}
	if len((SecretScanningObservation{}).Normalizations()) != 0 {
		t.Fatal("unacquired records available")
	}
}

func TestSecurityStatusObservationRejectsForgedProvenance(t *testing.T) {
	valid := SecurityStatusNormalization{Path: SecretProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}
	for _, test := range []struct {
		scanning, push bool
		records        []SecurityStatusNormalization
	}{
		{false, false, []SecurityStatusNormalization{{}}},
		{true, false, []SecurityStatusNormalization{valid}},
		{false, false, []SecurityStatusNormalization{valid, valid}},
		{false, false, []SecurityStatusNormalization{{Path: valid.Path, RawValue: "enabled", Fallback: false}}},
		{false, false, []SecurityStatusNormalization{{Path: valid.Path, RawValue: "disabled", Fallback: false}}},
		{false, false, []SecurityStatusNormalization{{Path: valid.Path, RawValue: "FUTURE", Fallback: true}}},
		{false, false, []SecurityStatusNormalization{{Path: "repository.settings.advanced_security.private_vulnerability_reporting", RawValue: "FUTURE", Fallback: false}}},
		{false, true, []SecurityStatusNormalization{{Path: PushProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}}},
	} {
		observed, err := NewSecretScanningObservationWithNormalizations(test.scanning, test.push, test.records)
		if err == nil || observed != (SecretScanningObservation{}) {
			t.Fatal(test, observed, err)
		}
		if _, _, err := observed.Values(); err == nil {
			t.Fatal("invalid provenance acquired")
		}
	}
}
