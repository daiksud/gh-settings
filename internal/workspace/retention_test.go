package workspace

import (
	"strings"
	"testing"
)

func TestRetentionYAML(t *testing.T) {
	for _, value := range []string{"1", "90", "400"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: {artifact_and_log_retention_days: " + value + "}}"))
		if err != nil {
			t.Fatal(value, err)
		}
		if d.Empty() || !d.Actions.ExecutionEmpty() || !d.Actions.WorkflowPermissionsEmpty() {
			t.Fatal(d)
		}
		data, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || again.Actions.RetentionDays == nil || *again.Actions.RetentionDays != *d.Actions.RetentionDays || strings.Contains(string(data), "maximum_allowed_days") {
			t.Fatal(string(data), again, err)
		}
	}
	for _, value := range []string{"0", "-1", "401", "null", "1.5", "'90'", "true", "[]"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: {artifact_and_log_retention_days: " + value + "}}")); err == nil {
			t.Fatal("invalid retention accepted", value)
		}
	}
	if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: {maximum_allowed_days: 90}}")); err == nil {
		t.Fatal("readonly metadata accepted")
	}
}
