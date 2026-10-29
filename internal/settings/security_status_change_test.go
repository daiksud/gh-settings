package settings

import "testing"

func TestNormalizedSecurityChangeRetainsRequiredBooleanWrites(t *testing.T) {
	for _, control := range []SecurityStatusControl{SecretProtectionControl, PushProtectionControl} {
		for _, raw := range []string{"FUTURE", ""} {
			for _, after := range []bool{false, true} {
				record := SecurityStatusNormalization{Path: control.Path(), RawValue: raw, Fallback: false}
				change, err := NewSecurityChangeWithNormalization(control.Field(), false, after, record)
				if err != nil {
					t.Fatal(err)
				}
				before, desired := change.Values()
				if change.Field() != control.Field() || before || desired != after || change.Normalization() != record {
					t.Fatal(change)
				}
				if err := ValidateSecurityChanges([]SecurityChange{change}); err != nil {
					t.Fatal(err)
				}
				record.RawValue = "mutated input"
				owned := change.Normalization()
				owned.RawValue = "mutated output"
				if change.Normalization().RawValue != raw {
					t.Fatal("aliased change metadata")
				}
			}
		}
		if _, err := NewSecurityChange(control.Field(), false, false); err == nil {
			t.Fatal("ordinary no-op allowed")
		}
	}
}

func TestNormalizedSecurityChangeRejectsForgedEvidence(t *testing.T) {
	record := SecurityStatusNormalization{Path: SecretProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}
	for _, test := range []struct {
		field         string
		before, after any
		record        SecurityStatusNormalization
	}{
		{"security.secret_scanning", true, false, record},
		{"security.secret_scanning", false, nil, record},
		{"security.secret_scanning", "false", false, record},
		{"security.secret_scanning", false, "true", record},
		{"security.secret_scanning", false, 1, record},
		{"security.secret_scanning", false, false, SecurityStatusNormalization{}},
		{"security.secret_scanning", false, true, SecurityStatusNormalization{Path: record.Path, RawValue: "disabled", Fallback: false}},
		{"security.secret_scanning", false, true, SecurityStatusNormalization{Path: record.Path, RawValue: "enabled", Fallback: false}},
		{"security.secret_scanning", false, true, SecurityStatusNormalization{Path: record.Path, RawValue: "FUTURE", Fallback: true}},
		{"security.secret_scanning_push_protection", false, true, record},
		{"security.private_vulnerability_reporting", false, true, record},
		{"security.immutable_releases", false, true, record},
	} {
		change, err := NewSecurityChangeWithNormalization(test.field, test.before, test.after, test.record)
		if err == nil || change != (SecurityChange{}) {
			t.Fatal(test, change, err)
		}
	}
	scanning, err := NewSecurityChangeWithNormalization(SecretProtectionControl.Field(), false, false, record)
	if err != nil {
		t.Fatal(err)
	}
	push, _ := NewSecurityChange(PushProtectionControl.Field(), false, true)
	if ValidateSecurityChanges([]SecurityChange{scanning, push}) == nil || ValidateSecurityChanges([]SecurityChange{scanning, scanning}) == nil {
		t.Fatal("contradictory or duplicate normalized report accepted")
	}
}
