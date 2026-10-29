package settings

import "testing"

func TestSecurityComparisonKeepsNormalizedBooleanWrites(t *testing.T) {
	for _, control := range []SecurityStatusControl{SecretProtectionControl, PushProtectionControl} {
		for _, raw := range []string{"FUTURE", ""} {
			for _, desired := range []bool{false, true} {
				record := SecurityStatusNormalization{Path: control.Path(), RawValue: raw, Fallback: false}
				observed, err := NewSecretScanningObservationWithNormalizations(false, false, []SecurityStatusNormalization{record})
				if err != nil {
					t.Fatal(err)
				}
				declaration := SecurityDeclaration{}
				if control == SecretProtectionControl {
					declaration.SecretScanning = &desired
				} else {
					declaration.PushProtection = &desired
				}
				managed, changes, err := CompareSecretScanning(observed, declaration)
				if err != nil || managed != 1 || len(changes) != 1 || changes[0].Field() != control.Field() || changes[0].Normalization() != record {
					t.Fatal(control, desired, managed, changes, err)
				}
				before, after := changes[0].Values()
				if before || after != desired {
					t.Fatal(before, after)
				}
				managed, changes, err = CompareSecretScanning(observed, SecurityDeclaration{})
				if err != nil || managed != 0 || len(changes) != 0 {
					t.Fatal("unmanaged evidence reported", managed, changes, err)
				}
			}
		}
	}
	observed, err := NewSecretScanningObservationWithNormalizations(false, false, []SecurityStatusNormalization{{Path: PushProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}, {Path: SecretProtectionControl.Path(), RawValue: "", Fallback: false}})
	if err != nil {
		t.Fatal(err)
	}
	managed, changes, err := CompareSecretScanning(observed, SecurityDeclaration{SecretScanning: securityBool(false), PushProtection: securityBool(false)})
	if err != nil || managed != 2 || len(changes) != 2 || changes[0].Field() != SecretProtectionControl.Field() || changes[1].Field() != PushProtectionControl.Field() {
		t.Fatal(managed, changes, err)
	}
	if _, _, err := CompareSecretScanning(observed, SecurityDeclaration{SecretScanning: securityBool(false), PushProtection: securityBool(true)}); err == nil {
		t.Fatal("desired dependency relaxed")
	}
}

func TestSecurityComparisonDiscardsInconsistentManagedProvenance(t *testing.T) {
	for _, record := range []SecurityStatusNormalization{
		{Path: "wrong", RawValue: "FUTURE", Fallback: false},
		{Path: PushProtectionControl.Path(), RawValue: "disabled", Fallback: false},
		{Path: PushProtectionControl.Path(), RawValue: "FUTURE", Fallback: true},
	} {
		observed := NewSecretScanningObservation(false, false)
		observed.normalizations[1] = record
		managed, changes, err := CompareSecretScanning(observed, SecurityDeclaration{SecretScanning: securityBool(true), PushProtection: securityBool(false)})
		if err == nil || managed != 0 || len(changes) != 0 {
			t.Fatal("partial comparison escaped", managed, changes, err)
		}
	}
	observed := NewSecretScanningObservation(false, true)
	observed.normalizations[1] = SecurityStatusNormalization{Path: PushProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}
	if managed, changes, err := CompareSecretScanning(observed, SecurityDeclaration{PushProtection: securityBool(false)}); err == nil || managed != 0 || len(changes) != 0 {
		t.Fatal(managed, changes, err)
	}
}
