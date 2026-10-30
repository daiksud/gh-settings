package settings

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
)

func TestRulesetRequiredApprovalsMetadataRequiresTypedIntegerMembers(t *testing.T) {
	for _, raw := range []int{-1, 0, 11} {
		expected := RulesetRequiredApprovalsNormalization{Path: RulesetRequiredApprovalsPath, RawValue: raw, Fallback: 0}
		data := []byte(`{"path":"repository.settings.rulesets.rules.require_a_pull_request_before_merging.required_approvals","raw_value":` + strconv.Itoa(raw) + `,"fallback":0}`)
		var got RulesetRequiredApprovalsNormalization
		if err := json.Unmarshal(data, &got); err != nil || got != expected {
			t.Fatal("typed integer lost", got, err)
		}
		encoded, err := json.Marshal(got)
		if err != nil || !reflect.DeepEqual(encoded, data) {
			t.Fatal("integer serialized as another type", string(encoded), err)
		}
	}
	// Structural decoding retains explicit 0; semantic observation validation separately rejects supported raw values.
	for _, data := range []string{
		`null`, `{}`, `{"path":"x","raw_value":11}`, `{"path":"x","fallback":0}`, `{"raw_value":11,"fallback":0}`,
		`{"path":null,"raw_value":11,"fallback":0}`, `{"path":"x","raw_value":null,"fallback":0}`, `{"path":"x","raw_value":11,"fallback":null}`,
		`{"path":1,"raw_value":11,"fallback":0}`, `{"path":"x","raw_value":"11","fallback":0}`, `{"path":"x","raw_value":true,"fallback":0}`, `{"path":"x","raw_value":11.5,"fallback":0}`, `{"path":"x","raw_value":1e100,"fallback":0}`, `{"path":"x","raw_value":9223372036854775808,"fallback":0}`,
		`{"path":"x","raw_value":11,"fallback":"0"}`, `{"path":"x","raw_value":11,"fallback":0.1}`, `{"path":"x","raw_value":11,"fallback":1e100}`, `{"path":"x","raw_value":11,"fallback":0,"extra":true}`,
	} {
		before := RulesetRequiredApprovalsNormalization{Path: "sentinel", RawValue: 99, Fallback: 7}
		got := before
		if err := json.Unmarshal([]byte(data), &got); err == nil || got != before {
			t.Fatal("malformed metadata accepted or partially assigned", data, got, err)
		}
	}
}
