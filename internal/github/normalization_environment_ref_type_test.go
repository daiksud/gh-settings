package github

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestObservedEnvironmentPatternDecoderOwnsRawTypeWithoutWeakeningStrictConfirmation(t *testing.T) {
	for _, rawType := range []string{"branch", "tag", "future", "", " \t", "Branch"} {
		encoded, _ := json.Marshal(rawType)
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(`{"id":7,"name":"Release/*","type":`+string(encoded)+`,"node_id":"node"}`), &raw); err != nil {
			t.Fatal(err)
		}
		before, _ := json.Marshal(raw)
		got, err := decodeObservedEnvironmentPattern(raw)
		if err != nil || got.ID != 7 || got.Pattern.Name != "Release/*" {
			t.Fatal(got, err)
		}
		native, err := got.RawPattern()
		if err != nil || native != (settings.EnvironmentPattern{Type: rawType, Name: "Release/*"}) {
			t.Fatal("raw identity lost", native, err)
		}
		canonical := rawType == "branch" || rawType == "tag"
		if canonical {
			if got.Pattern.Type != rawType || got.RefTypeNormalization != (settings.ObservationNormalization{}) {
				t.Fatal("canonical projection changed", got)
			}
		} else if got.Pattern.Type != "branch" || got.RefTypeNormalization != (settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: rawType, Fallback: "branch"}) {
			t.Fatal("fallback provenance lost", got)
		}
		after, _ := json.Marshal(raw)
		if !bytes.Equal(before, after) {
			t.Fatal("input map mutated", string(before), string(after))
		}
		strict, err := decodeEnvironmentPattern(raw)
		if canonical && (err != nil || strict.Pattern != native) || !canonical && err == nil {
			t.Fatal("strict mutation confirmation relaxed", strict, err)
		}
	}
}

func TestObservedEnvironmentPatternDecoderRejectsAcquisitionFailuresWithZeroState(t *testing.T) {
	for _, body := range []string{
		`{}`, `{"id":7,"name":"main"}`, `{"id":7,"name":"main","type":null}`, `{"id":7,"name":"main","type":false}`,
		`{"id":7,"name":"main","type":[]}`, `{"id":7,"name":"main","type":{}}`, `{"id":null,"name":"main","type":"future"}`,
		`{"id":0,"name":"main","type":"future"}`, `{"id":7.5,"name":"main","type":"future"}`, `{"id":7,"name":null,"type":"future"}`,
		`{"id":7,"type":"future"}`, `{"id":7,"name":"","type":"future"}`, `{"id":7,"name":"bad\nname","type":"future"}`,
		`{"id":7,"name":"main","type":"future","extra":true}`,
	} {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &raw); err != nil {
			t.Fatal(err)
		}
		if got, err := decodeObservedEnvironmentPattern(raw); err == nil || got != (settings.EnvironmentPatternState{}) {
			t.Fatal("bad acquisition normalized", body, got, err)
		}
	}
}
