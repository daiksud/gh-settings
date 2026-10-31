package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPrepareSecurityStatusObservation(t *testing.T) {
	for _, test := range []struct {
		control settings.SecurityStatusControl
		path    string
	}{
		{settings.SecretProtectionControl, "repository.settings.advanced_security.secret_protection.enabled"},
		{settings.PushProtectionControl, "repository.settings.advanced_security.secret_protection.push_protection"},
	} {
		for _, raw := range []string{"enabled", "disabled", "FUTURE", "", " \t"} {
			prepared, err := prepareSecurityStatusObservation(&raw, test.control)
			if err != nil {
				t.Fatal(err)
			}
			wantRecord := settings.SecurityStatusNormalization{}
			if raw != "enabled" && raw != "disabled" {
				wantRecord = settings.SecurityStatusNormalization{Path: test.path, RawValue: raw, Fallback: false}
			}
			if prepared.Value == nil || *prepared.Value != (raw == "enabled") || prepared.Normalization != wantRecord {
				t.Fatalf("raw=%q got=%+v want=%+v", raw, prepared, wantRecord)
			}
			original := raw
			raw = "changed caller storage"
			if prepared.Normalization.RawValue != "" && prepared.Normalization.RawValue != original {
				t.Fatal("caller raw storage retained", prepared)
			}
			*prepared.Value = !*prepared.Value
			repeat, err := prepareSecurityStatusObservation(&original, test.control)
			if err != nil || repeat.Value == nil || *repeat.Value != (original == "enabled") {
				t.Fatal("effective pointer aliased", repeat, err)
			}
		}
		unacquired, err := prepareSecurityStatusObservation(nil, test.control)
		if err != nil || unacquired.Value != nil || unacquired.Normalization != (settings.SecurityStatusNormalization{}) {
			t.Fatal("nil acquired", unacquired, err)
		}
	}
	for _, raw := range []*string{nil, actionsUIPtr("enabled"), actionsUIPtr("FUTURE")} {
		prepared, err := prepareSecurityStatusObservation(raw, settings.SecurityStatusControl("other"))
		if err == nil || prepared.Value != nil || prepared.Normalization != (settings.SecurityStatusNormalization{}) {
			t.Fatal("invalid control accepted", prepared, err)
		}
	}
}

func TestSecurityUpdaterWarningsUseOnlyFreshManagedEvidence(t *testing.T) {
	scanning := settings.SecurityStatusNormalization{Path: settings.SecretProtectionControl.Path(), RawValue: "FRESH_SCANNING", Fallback: false}
	push := settings.SecurityStatusNormalization{Path: settings.PushProtectionControl.Path(), RawValue: "", Fallback: false}
	observed, err := settings.NewSecretScanningObservationWithNormalizations(false, false, []settings.SecurityStatusNormalization{scanning, push})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		observation settings.SecretScanningObservation
		field       string
		expected    []settings.SecurityStatusNormalization
	}{
		{observed, settings.SecretProtectionControl.Field(), []settings.SecurityStatusNormalization{scanning}},
		{observed, settings.PushProtectionControl.Field(), []settings.SecurityStatusNormalization{push}},
		{observed, "security.private_vulnerability_reporting", nil},
		{settings.NewSecretScanningObservation(false, false), settings.SecretProtectionControl.Field(), nil},
		{settings.SecretScanningObservation{}, settings.SecretProtectionControl.Field(), nil},
	} {
		var warnings []settings.SecurityStatusNormalization
		client := &SecretScanningClient{NormalizationWarning: func(record settings.SecurityStatusNormalization) { warnings = append(warnings, record) }}
		client.warnStatusNormalization(test.observation, test.field)
		if len(warnings) != len(test.expected) {
			t.Fatal(test, warnings)
		}
		for i, record := range warnings {
			if record != test.expected[i] {
				t.Fatal(test, warnings)
			}
		}
		client.NormalizationWarning = nil
		client.warnStatusNormalization(test.observation, test.field)
	}
}
