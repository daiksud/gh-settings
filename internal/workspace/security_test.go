package workspace

import (
	"strings"
	"testing"
)

func TestSecurityYAMLContract(t *testing.T) {
	for _, value := range []string{"{}", "{secret_scanning: true}", "{secret_scanning: false, secret_scanning_push_protection: false}", "{secret_scanning_push_protection: true}"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {security: " + value + "}"))
		if err != nil {
			t.Fatal(value, err)
		}
		b, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		round, err := DecodeConfig(b)
		if err != nil || round.Security.Empty() != d.Security.Empty() {
			t.Fatal(string(b), round, err)
		}
		if value == "{}" && (!d.Empty() || strings.Contains(string(b), "security:")) {
			t.Fatal("empty mapping became managed", string(b))
		}
		if strings.Contains(value, "secret_scanning: false") && (round.Security.SecretScanning == nil || *round.Security.SecretScanning) {
			t.Fatal("false lost", string(b))
		}
	}
	for _, value := range []string{"null", "[]", "{secret_scanning: null}", "{secret_scanning: 'true'}", "{secret_scanning: yes}", "{secret_scanning: false, secret_scanning_push_protection: true}", "{advanced_security: true}", "{dependabot_alerts: true}", "{secret_scanning: true, secret_scanning: false}"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {security: " + value + "}")); err == nil {
			t.Fatal("invalid security accepted", value)
		}
	}
}
