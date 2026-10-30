package github

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func mergeMethodsRuleJSON(value string) json.RawMessage {
	return json.RawMessage(`[{"type":"pull_request","parameters":{"allowed_merge_methods":` + value + `}}]`)
}

func TestMergeMethodsProjectorPreservesTypedRawOrderAndWholeFallback(t *testing.T) {
	for _, tc := range []struct {
		value      string
		normalized bool
	}{
		{`["merge"]`, false}, {`["squash","rebase","merge"]`, false}, {`["future"]`, true}, {`["squash","future","merge"]`, true}, {`[""]`, true}, {`[" ","rebase"]`, true},
	} {
		raw := mergeMethodsRuleJSON(tc.value)
		original := bytes.Clone(raw)
		got, record, err := prepareRulesetMergeMethodsRules(raw)
		if err != nil || !bytes.Equal(raw, original) {
			t.Fatal(tc.value, record, err)
		}
		var methods []string
		json.Unmarshal([]byte(tc.value), &methods)
		want := settings.RulesetMergeMethodsNormalization{}
		if tc.normalized {
			want = settings.RulesetMergeMethodsNormalization{Path: settings.RulesetMergeMethodsPath, RawValue: methods, Fallback: []string{"merge", "squash", "rebase"}}
		}
		if !reflect.DeepEqual(record, want) {
			t.Fatal(record, want)
		}
		var rules []struct {
			Parameters struct {
				Methods []string `json:"allowed_merge_methods"`
			} `json:"parameters"`
		}
		if err := json.Unmarshal(got, &rules); err != nil {
			t.Fatal(err)
		}
		expected := methods
		if tc.normalized {
			expected = []string{"merge", "squash", "rebase"}
		}
		if !reflect.DeepEqual(rules[0].Parameters.Methods, expected) {
			t.Fatal(rules, expected)
		}
		if tc.normalized {
			record.RawValue[0] = "CORRUPT"
			record.Fallback[0] = "CORRUPT"
			if bytes.Contains(got, []byte("CORRUPT")) {
				t.Fatal("record aliases projection")
			}
		}
		got[0] = '!'
		if !bytes.Equal(raw, original) {
			t.Fatal("output aliases source")
		}
	}
	for _, raw := range []string{`[]`, `[{"type":"deletion"}]`} {
		got, record, err := prepareRulesetMergeMethodsRules(json.RawMessage(raw))
		var before, after any
		json.Unmarshal([]byte(raw), &before)
		json.Unmarshal(got, &after)
		if err != nil || !record.IsZero() || !reflect.DeepEqual(before, after) {
			t.Fatal("absence invented methods", record, err)
		}
	}
}

func TestMergeMethodsProjectorRejectsIncompleteMalformedOrDuplicateLists(t *testing.T) {
	invalid := []string{`null`, `{}`, `[null]`, `[{"type":null}]`, `[{"type":"pull_request"}]`, `[{"type":"pull_request","parameters":null}]`, `[{"type":"pull_request","parameters":{}}]`}
	for _, value := range []string{`null`, `[]`, `{}`, `"future"`, `1`, `[null]`, `["future",null]`, `[1]`, `[false]`, `[[]]`, `["future","future"]`, `["merge","merge","future"]`, `["future","", ""]`} {
		invalid = append(invalid, string(mergeMethodsRuleJSON(value)))
	}
	invalid = append(invalid, `[{"type":"pull_request","parameters":{"allowed_merge_methods":["future"]}},{"type":"pull_request","parameters":{"allowed_merge_methods":["merge"]}}]`)
	for _, raw := range invalid {
		input := json.RawMessage(raw)
		original := bytes.Clone(input)
		got, record, err := prepareRulesetMergeMethodsRules(input)
		if err == nil || got != nil || !record.IsZero() || !bytes.Equal(input, original) {
			t.Fatal("invalid acquisition escaped", raw, string(got), record, err)
		}
	}
	// Projection must retain unrelated fields for the complete policy validator.
	var policy map[string]json.RawMessage
	json.Unmarshal([]byte(approvalsRulesetJSON), &policy)
	raw := strings.Replace(string(policy["rules"]), `"allowed_merge_methods":["squash"]`, `"allowed_merge_methods":["future"],"unknown":true`, 1)
	got, record, err := prepareRulesetMergeMethodsRules(json.RawMessage(raw))
	if err != nil || record.IsZero() {
		t.Fatal(record, err)
	}
	policy["rules"] = got
	if _, err := decodeRulesetPolicy(policy); err == nil {
		t.Fatal("projection legalized unrelated unsupported member")
	}
}
