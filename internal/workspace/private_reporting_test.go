package workspace

import (
	"strings"
	"testing"
)

func TestPrivateReportingYAML(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {security: {private_vulnerability_reporting: " + value + "}}"))
		if err != nil {
			t.Fatal(err)
		}
		if d.Empty() || !d.Security.SecretScanningEmpty() || d.Security.PrivateVulnerabilityReporting == nil {
			t.Fatal(d)
		}
		data, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || again.Security.PrivateVulnerabilityReporting == nil || *again.Security.PrivateVulnerabilityReporting != (value == "true") || strings.Contains(string(data), "secret_scanning:") {
			t.Fatal(string(data), again, err)
		}
	}
	for _, value := range []string{"null", "'true'", "[]", "{enabled: true}", "yes", "1"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {security: {private_vulnerability_reporting: " + value + "}}")); err == nil {
			t.Fatal("invalid value accepted", value)
		}
	}
	for _, input := range []string{"schemaVersion: 1\nsettings: {security: {private_vulnerability_reporting: true, private_vulnerability_reporting: false}}", "schemaVersion: 1\nsettings: {security: {private_reporting: true}}"} {
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatal("invalid key accepted", input)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {security: {secret_scanning: false, private_vulnerability_reporting: true}}"))
	if err != nil || d.Security.SecretScanningEmpty() || d.Security.PrivateVulnerabilityReporting == nil {
		t.Fatal("independent fields coupled", d, err)
	}
}
