package github

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRulesetQualityRulesProjectionOwnsOnlyPresentSeverity(t *testing.T) {
	for _, raw := range []string{"errors", "warnings", "notes", "all", "FUTURE_SEVERITY", "", " \t", "warnings_and_higher"} {
		inputRules := []any{map[string]any{"type": "deletion"}, map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": raw, "unrelated": true}}, map[string]any{"type": "non_fast_forward"}}
		input, _ := json.Marshal(inputRules)
		original := bytes.Clone(input)
		output, record, err := prepareRulesetQualityRules(input)
		wantSeverity := raw
		wantRecord := settings.ObservationNormalization{}
		if raw != "errors" && raw != "warnings" && raw != "notes" && raw != "all" {
			wantSeverity = "errors"
			wantRecord = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: raw, Fallback: "errors"}
		}
		want := []any{map[string]any{"type": "deletion"}, map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": wantSeverity, "unrelated": true}}, map[string]any{"type": "non_fast_forward"}}
		var actual []any
		if err != nil || json.Unmarshal(output, &actual) != nil || !reflect.DeepEqual(actual, want) || record != wantRecord || !bytes.Equal(input, original) {
			t.Fatal(actual, record, err)
		}
		output[0] = 'x'
		if !bytes.Equal(input, original) {
			t.Fatal("projection output aliases input")
		}
		// This scalar projector does not approve unrelated writable fields.
		var policy map[string]json.RawMessage
		json.Unmarshal([]byte(basicRulesetJSON), &policy)
		validOutput, _, err := prepareRulesetQualityRules(input)
		if err != nil {
			t.Fatal(err)
		}
		policy["rules"] = validOutput
		if _, err := decodeRulesetPolicy(policy); err == nil {
			t.Fatal("strict decoder lost unrelated-field validation")
		}
	}
	for _, raw := range []string{`[]`, `[{"type":"deletion"}]`} {
		input := []byte(raw)
		output, record, err := prepareRulesetQualityRules(input)
		if err != nil || record != (settings.ObservationNormalization{}) || !bytes.Equal(input, output) {
			t.Fatal("absent rule invented", string(output), record, err)
		}
		output[0] = 'x'
		if string(input) != raw {
			t.Fatal("canonical output aliases input")
		}
	}
}

func TestRulesetQualityRulesProjectionRejectsMissingOrAmbiguousValues(t *testing.T) {
	for _, raw := range []string{
		``, `null`, `{}`, `[null]`, `[{"type":null}]`, `[{"type":42}]`,
		`[{"type":"code_quality"}]`, `[{"type":"code_quality","parameters":null}]`, `[{"type":"code_quality","parameters":[]}]`,
		`[{"type":"code_quality","parameters":{}}]`, `[{"type":"code_quality","parameters":{"severity":null}}]`, `[{"type":"code_quality","parameters":{"severity":42}}]`, `[{"type":"code_quality","parameters":{"severity":false}}]`, `[{"type":"code_quality","parameters":{"severity":[]}}]`,
		`[{"type":"code_quality","parameters":{"severity":"errors"}},{"type":"code_quality","parameters":{"severity":"FUTURE_SEVERITY"}}]`,
	} {
		input := []byte(raw)
		original := bytes.Clone(input)
		output, record, err := prepareRulesetQualityRules(input)
		if err == nil || output != nil || record != (settings.ObservationNormalization{}) || !bytes.Equal(input, original) {
			t.Fatal("invalid projection escaped", raw, string(output), record, err)
		}
	}
}
