package workspace

import (
	"strings"
	"testing"
)

func TestUIDependabotPresenceRoundTripAndStrictShape(t *testing.T) {
	for _, body := range []string{"{}", "{alerts: false}", "{security_updates: true}", "{alerts: true, security_updates: false}"} {
		input := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {dependabot: " + body + "}}}\n")
		model, err := DecodeConfig(input)
		if err != nil {
			t.Fatal(body, err)
		}
		a := model.Settings.SecurityAndQuality.AdvancedSecurity
		if (a.DependabotAlerts != nil) != strings.Contains(body, "alerts:") || (a.DependabotSecurityUpdates != nil) != strings.Contains(body, "security_updates:") {
			t.Fatal(body, a)
		}
		data, err := EncodeConfig(model)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeConfig(data)
		if err != nil {
			t.Fatal(string(data), err)
		}
		b := decoded.Settings.SecurityAndQuality.AdvancedSecurity
		if a.DependabotAlerts != nil && (b.DependabotAlerts == nil || *a.DependabotAlerts != *b.DependabotAlerts) {
			t.Fatal("false lost", string(data))
		}
	}
	for _, body := range []string{"null", "true", "[]", "{alerts: null}", "{security_updates: 'false'}", "{alerts: false, alerts: true}", "{paused: false}", "{vulnerability_alerts: true}", "{enabled: true}"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {dependabot: " + body + "}}}\n")); err == nil {
			t.Fatal("invalid Dependabot shape accepted", body)
		}
	}
}
