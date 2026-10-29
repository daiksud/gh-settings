package settings

import "testing"

func TestSecurityStatusMappingIsClosed(t *testing.T) {
	for _, test := range []struct {
		control     SecurityStatusControl
		path, field string
	}{
		{SecretProtectionControl, "repository.settings.security_and_quality.advanced_security.secret_protection", "security.secret_scanning"},
		{PushProtectionControl, "repository.settings.security_and_quality.advanced_security.push_protection", "security.secret_scanning_push_protection"},
	} {
		fallback, valid := test.control.DefaultValue()
		if !valid || fallback || test.control.Path() != test.path || test.control.Field() != test.field {
			t.Fatal(test, fallback, valid)
		}
	}
	for _, invalid := range []SecurityStatusControl{"", "secret_scanning", "private_vulnerability_reporting", "immutable_releases"} {
		_, valid := invalid.DefaultValue()
		if valid || invalid.Path() != "" || invalid.Field() != "" {
			t.Fatal(invalid)
		}
	}
	for _, test := range []struct {
		status       string
		value, valid bool
	}{{"enabled", true, true}, {"disabled", false, true}, {"", false, false}, {"FUTURE", false, false}, {"Enabled", false, false}, {" disabled ", false, false}} {
		value, valid := SecurityValueForStatus(test.status)
		if value != test.value || valid != test.valid {
			t.Fatal(test, value, valid)
		}
		if test.valid && SecurityStatusForValue(value) != test.status {
			t.Fatal(test)
		}
	}
}
