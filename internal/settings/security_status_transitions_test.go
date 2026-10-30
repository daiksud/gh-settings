package settings

import "testing"

func TestSecurityTransitionsRequireRawPrerequisiteCertainty(t *testing.T) {
	for _, raw := range []string{"FUTURE", ""} {
		unknownPush, err := NewSecretScanningObservationWithNormalizations(true, false, []SecurityStatusNormalization{{Path: PushProtectionControl.Path(), RawValue: raw, Fallback: false}})
		if err != nil {
			t.Fatal(err)
		}
		if changes, err := SecretScanningTransitions(unknownPush, SecurityDeclaration{SecretScanning: securityBool(false)}); err == nil || len(changes) != 0 {
			t.Fatal("unknown push proved disabled", changes, err)
		}
		changes, err := SecretScanningTransitions(unknownPush, SecurityDeclaration{SecretScanning: securityBool(false), PushProtection: securityBool(false)})
		if err != nil || len(changes) != 2 || changes[0].Field() != PushProtectionControl.Field() || changes[1].Field() != SecretProtectionControl.Field() || changes[0].Normalization().RawValue != raw {
			t.Fatal("explicit disable ordering lost", changes, err)
		}
		unknownScanning, err := NewSecretScanningObservationWithNormalizations(false, false, []SecurityStatusNormalization{{Path: SecretProtectionControl.Path(), RawValue: raw, Fallback: false}})
		if err != nil {
			t.Fatal(err)
		}
		if changes, err := SecretScanningTransitions(unknownScanning, SecurityDeclaration{PushProtection: securityBool(true)}); err == nil || len(changes) != 0 {
			t.Fatal("unknown scanning proved enabled", changes, err)
		}
		changes, err = SecretScanningTransitions(unknownScanning, SecurityDeclaration{SecretScanning: securityBool(true), PushProtection: securityBool(true)})
		if err != nil || len(changes) != 2 || changes[0].Field() != SecretProtectionControl.Field() || changes[1].Field() != PushProtectionControl.Field() {
			t.Fatal("explicit enable ordering lost", changes, err)
		}
	}
	for _, observed := range []SecretScanningObservation{NewSecretScanningObservation(true, false), NewSecretScanningObservation(false, false)} {
		changes, err := SecretScanningTransitions(observed, SecurityDeclaration{PushProtection: securityBool(false)})
		if err != nil || len(changes) != 0 {
			t.Fatal("canonical false no-op changed", changes, err)
		}
	}
}

func TestFreshSecurityChangePreservesRawCompanionAndClearsWrittenRecord(t *testing.T) {
	scanning := SecurityStatusNormalization{Path: SecretProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}
	push := SecurityStatusNormalization{Path: PushProtectionControl.Path(), RawValue: "", Fallback: false}
	observed, err := NewSecretScanningObservationWithNormalizations(false, false, []SecurityStatusNormalization{scanning, push})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		control                SecurityStatusControl
		desired                bool
		record                 SecurityStatusNormalization
		wantScanning, wantPush string
	}{
		{SecretProtectionControl, true, scanning, "enabled", ""},
		{PushProtectionControl, false, push, "FUTURE", "disabled"},
	} {
		change, err := NewSecurityChangeWithNormalization(test.control.Field(), false, test.desired, test.record)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := AfterSecretScanningChange(observed, change)
		if err != nil {
			t.Fatal(err)
		}
		s, p, err := expected.RawStatuses()
		if err != nil || s != test.wantScanning || p != test.wantPush || len(expected.Normalizations()) != 1 || expected.Normalizations()[0].Path == test.control.Path() {
			t.Fatal("raw forecast lost certainty", expected, s, p, err)
		}
		originalS, originalP, err := observed.RawStatuses()
		if err != nil || originalS != "FUTURE" || originalP != "" || len(observed.Normalizations()) != 2 {
			t.Fatal("source mutated", observed, err)
		}
	}
}

func TestFreshSecurityChangeRejectsUnprovenPrerequisites(t *testing.T) {
	pushRecord := SecurityStatusNormalization{Path: PushProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}
	unknownPush, err := NewSecretScanningObservationWithNormalizations(true, false, []SecurityStatusNormalization{pushRecord})
	if err != nil {
		t.Fatal(err)
	}
	scanningRecord := SecurityStatusNormalization{Path: SecretProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}
	unknownScanning, err := NewSecretScanningObservationWithNormalizations(false, false, []SecurityStatusNormalization{scanningRecord})
	if err != nil {
		t.Fatal(err)
	}
	disableScanning, _ := NewSecurityChange(SecretProtectionControl.Field(), true, false)
	enablePush, _ := NewSecurityChange(PushProtectionControl.Field(), false, true)
	for _, test := range []struct {
		observed SecretScanningObservation
		change   SecurityChange
	}{
		{unknownPush, disableScanning},
		{unknownScanning, enablePush},
		{SecretScanningObservation{}, enablePush},
		{NewSecretScanningObservation(true, false), SecurityChange{field: PushProtectionControl.Field(), before: false, after: false}},
		{NewSecretScanningObservation(true, false), SecurityChange{field: PushProtectionControl.Field(), before: false, after: true, normalization: scanningRecord}},
	} {
		if expected, err := AfterSecretScanningChange(test.observed, test.change); err == nil || expected != (SecretScanningObservation{}) {
			t.Fatal(test, expected, err)
		}
	}
}
