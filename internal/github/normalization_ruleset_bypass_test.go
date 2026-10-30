package github

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRulesetBypassProjectionRetainsExactActorEvidence(t *testing.T) {
	for _, rawMode := range []string{"FUTURE_MODE", "", " \t", "for_pull_requests_only"} {
		input, _ := json.Marshal([]any{
			map[string]any{"actor_type": "Team", "actor_id": 9, "bypass_mode": rawMode, "future_field": true},
			map[string]any{"actor_type": "User", "actor_id": 9, "bypass_mode": "exempt"},
			map[string]any{"actor_type": "DeployKey", "actor_id": nil, "bypass_mode": "FUTURE_KEY_MODE"},
		})
		original := bytes.Clone(input)
		output, records, err := prepareRulesetBypassActors(input)
		id := int64(9)
		wantRecords := []settings.RulesetBypassModeNormalization{
			{ActorType: "Team", ActorID: &id, Normalization: settings.ObservationNormalization{Path: "repository.settings.rulesets.bypass_list.bypass_mode", RawValue: rawMode, Fallback: "always"}},
			{ActorType: "DeployKey", Normalization: settings.ObservationNormalization{Path: "repository.settings.rulesets.bypass_list.bypass_mode", RawValue: "FUTURE_KEY_MODE", Fallback: "always"}},
		}
		want := []any{
			map[string]any{"actor_type": "Team", "actor_id": float64(9), "bypass_mode": "always", "future_field": true},
			map[string]any{"actor_type": "User", "actor_id": float64(9), "bypass_mode": "exempt"},
			map[string]any{"actor_type": "DeployKey", "actor_id": nil, "bypass_mode": "always"},
		}
		var actual []any
		if err != nil || json.Unmarshal(output, &actual) != nil || !reflect.DeepEqual(actual, want) || !reflect.DeepEqual(records, wantRecords) || !bytes.Equal(input, original) {
			t.Fatal("lost actor-bound projection", string(output), records, err)
		}
		// Projection preserves fields for the caller's strict complete-policy check.
		var policy map[string]json.RawMessage
		json.Unmarshal([]byte(basicRulesetJSON), &policy)
		policy["bypass_actors"] = output
		if _, err := decodeRulesetPolicy(policy); err == nil {
			t.Fatal("unknown actor member escaped strict policy validation")
		}
		output[0], *records[0].ActorID = 'x', 123
		if !bytes.Equal(input, original) {
			t.Fatal("projection aliases acquired bytes")
		}
		_, again, err := prepareRulesetBypassActors(input)
		if err != nil || !reflect.DeepEqual(again, wantRecords) {
			t.Fatal("record identity aliases another projection", again, err)
		}
	}
}

func TestRulesetBypassProjectionPreservesCanonicalActorsAndKnownEmpty(t *testing.T) {
	for _, raw := range []string{
		`[]`,
		`[{"actor_type":"Team","actor_id":9,"bypass_mode":"always"},{"actor_type":"User","actor_id":9,"bypass_mode":"exempt"},{"actor_type":"Integration","actor_id":3,"bypass_mode":"pull_request"}]`,
		`[{"actor_type":"OrganizationAdmin","actor_id":99,"bypass_mode":"always"},{"actor_type":"DeployKey","actor_id":null,"bypass_mode":"always"}]`,
	} {
		input := []byte(raw)
		output, records, err := prepareRulesetBypassActors(input)
		var before, after any
		if err != nil || len(records) != 0 || json.Unmarshal(input, &before) != nil || json.Unmarshal(output, &after) != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("canonical or empty list changed", string(output), records, err)
		}
		output[0] = 'x'
		if string(input) != raw {
			t.Fatal("canonical output aliases input")
		}
	}
}

func TestRulesetBypassProjectionRejectsIncompleteAndAmbiguousAcquisition(t *testing.T) {
	for _, raw := range []string{
		``, `null`, `{}`, `[null]`, `[{}]`,
		`[{"actor_type":null,"actor_id":9,"bypass_mode":"FUTURE_MODE"}]`,
		`[{"actor_type":"Team","actor_id":9}]`,
		`[{"actor_type":"Team","actor_id":9,"bypass_mode":null}]`,
		`[{"actor_type":"Team","actor_id":9,"bypass_mode":42}]`,
		`[{"actor_type":"Team","bypass_mode":"FUTURE_MODE"}]`,
		`[{"actor_type":"Team","actor_id":null,"bypass_mode":"FUTURE_MODE"}]`,
		`[{"actor_type":"Team","actor_id":"9","bypass_mode":"FUTURE_MODE"}]`,
		`[{"actor_type":"Team","actor_id":1.5,"bypass_mode":"FUTURE_MODE"}]`,
		`[{"actor_type":"Team","actor_id":0,"bypass_mode":"FUTURE_MODE"}]`,
		`[{"actor_type":"DeployKey","actor_id":1,"bypass_mode":"FUTURE_MODE"}]`,
		`[{"actor_type":"OrganizationAdmin","actor_id":"9","bypass_mode":"FUTURE_MODE"}]`,
		`[{"actor_type":"Team","actor_id":9,"bypass_mode":"FUTURE_MODE"},{"actor_type":"Team","actor_id":9,"bypass_mode":"always"}]`,
		`[{"actor_type":"OrganizationAdmin","actor_id":1,"bypass_mode":"FUTURE_MODE"},{"actor_type":"OrganizationAdmin","actor_id":2,"bypass_mode":"always"}]`,
		`[{"actor_type":"Team","actor_id":9,"bypass_mode":"FUTURE_MODE"},{"actor_type":"User","actor_id":9,"bypass_mode":false}]`,
	} {
		input := []byte(raw)
		original := bytes.Clone(input)
		output, records, err := prepareRulesetBypassActors(input)
		if err == nil || output != nil || len(records) != 0 || !bytes.Equal(input, original) {
			t.Fatal("partial or malformed projection escaped", raw, string(output), records, err)
		}
	}
}
