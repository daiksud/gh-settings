package settings

import (
	"encoding/json"
	"testing"
)

func TestSecurityStatusNormalizationRequiresTypedAcquiredMembers(t *testing.T) {
	valid := SecurityStatusNormalization{Path: "repository.settings.security_and_quality.advanced_security.secret_protection", RawValue: "", Fallback: false}
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	var decoded SecurityStatusNormalization
	if err := json.Unmarshal(data, &decoded); err != nil || decoded != valid {
		t.Fatal(string(data), decoded, err)
	}
	for _, field := range []string{"path", "raw_value", "fallback"} {
		invalidValues := []any{nil, 1, []any{}, map[string]any{}}
		if field == "fallback" {
			invalidValues = append(invalidValues, "false")
		} else {
			invalidValues = append(invalidValues, false)
		}
		for _, value := range invalidValues {
			fields := map[string]any{"path": valid.Path, "raw_value": "", "fallback": false}
			fields[field] = value
			data, _ := json.Marshal(fields)
			original := SecurityStatusNormalization{Path: "unchanged", RawValue: "unchanged", Fallback: true}
			decoded = original
			if err := json.Unmarshal(data, &decoded); err == nil || decoded != original {
				t.Fatalf("invalid typed metadata accepted or assigned: %s %+v %v", data, decoded, err)
			}
		}
		fields := map[string]any{"path": valid.Path, "raw_value": "", "fallback": false}
		delete(fields, field)
		data, _ := json.Marshal(fields)
		if err := json.Unmarshal(data, &decoded); err == nil {
			t.Fatalf("missing %s accepted: %s", field, data)
		}
	}
	for _, data := range []string{`null`, `{}`, `{"path":"x","raw_value":"","fallback":false,"extra":true}`} {
		if err := json.Unmarshal([]byte(data), &decoded); err == nil {
			t.Fatal("invalid record accepted", data)
		}
	}
}
