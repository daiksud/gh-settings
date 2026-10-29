package github

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestObservedRulesetEnforcementDecoderPreservesSourceAndStrictVerification(t *testing.T) {
	for _, token := range []string{"active", "disabled", "evaluate", "FUTURE_MODE", "", " \t"} {
		t.Run(token, func(t *testing.T) {
			var raw map[string]json.RawMessage
			if err := json.Unmarshal([]byte(basicRulesetJSON), &raw); err != nil {
				t.Fatal(err)
			}
			raw["enforcement"], _ = json.Marshal(token)
			before := map[string]json.RawMessage{}
			for key, value := range raw {
				before[key] = bytes.Clone(value)
			}
			policy, record, err := decodeObservedRulesetPolicy(raw)
			want := token
			wantRecord := settings.ObservationNormalization{}
			if token != "active" && token != "disabled" {
				want = "disabled"
				wantRecord = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: token, Fallback: "disabled"}
			}
			if err != nil || policy.Enforcement == nil || *policy.Enforcement != want || record != wantRecord || !reflect.DeepEqual(raw, before) {
				t.Fatal(policy, record, err, raw, before)
			}
			if token != "active" && token != "disabled" {
				if err := verifyReviewRulesetResponse(policy, raw); err == nil {
					t.Fatal("projected equality confirmed raw write", token)
				}
				_, err := decodeRulesetPolicy(raw)
				if token != "evaluate" && err == nil {
					t.Fatal("strict mutation decoder accepted unknown", token)
				}
			}
		})
	}
}

func TestObservedRulesetEnforcementDecoderRejectsIncompletePolicy(t *testing.T) {
	for _, mutate := range []func(map[string]json.RawMessage){
		func(raw map[string]json.RawMessage) { delete(raw, "enforcement") },
		func(raw map[string]json.RawMessage) { raw["enforcement"] = json.RawMessage(`null`) },
		func(raw map[string]json.RawMessage) { raw["enforcement"] = json.RawMessage(`42`) },
		func(raw map[string]json.RawMessage) { raw["rules"] = json.RawMessage(`null`) },
		func(raw map[string]json.RawMessage) { delete(raw, "bypass_actors") },
		func(raw map[string]json.RawMessage) {
			raw["conditions"] = json.RawMessage(`{"ref_name":{"include":[],"exclude":null}}`)
		},
		func(raw map[string]json.RawMessage) {
			raw["conditions"] = json.RawMessage(`{"ref_name":{"include":[],"exclude":[],"future":true}}`)
		},
		func(raw map[string]json.RawMessage) { raw["rules"] = json.RawMessage(`[{"type":"future"}]`) },
	} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(basicRulesetJSON), &raw)
		raw["enforcement"] = json.RawMessage(`"FUTURE_MODE"`)
		mutate(raw)
		before := map[string]json.RawMessage{}
		for key, value := range raw {
			before[key] = bytes.Clone(value)
		}
		policy, record, err := decodeObservedRulesetPolicy(raw)
		if err == nil || !reflect.DeepEqual(policy, settings.RulesetDeclaration{}) || record != (settings.ObservationNormalization{}) || !reflect.DeepEqual(raw, before) {
			t.Fatal("invalid partial policy escaped", policy, record, err)
		}
	}
}
