package workspace

import (
	"strings"
	"testing"
)

func TestRemovedOrganizationSettingsFailLocally(t *testing.T) {
	for _, body := range []string{
		"  custom_properties: {}\n", "  custom_properties: null\n", "  custom_properties: {service: api}\n",
		"  repository: {allow_forking: false}\n", "  repository: {allow_forking: true}\n", "  repository: {allow_forking: null}\n",
	} {
		_, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n" + body))
		if err == nil || !strings.Contains(err.Error(), "unknown") {
			t.Fatalf("removed setting accepted: %q error=%v", body, err)
		}
	}
}
