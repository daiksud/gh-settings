package github

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func teamApprovalsRulesJSON(count string) json.RawMessage {
	return json.RawMessage(`[{"type":"pull_request","parameters":{"required_reviewers":[{"reviewer":{"type":"Team","id":10},"minimum_approvals":` + count + `,"file_patterns":["**","!generated/**","**"]},{"reviewer":{"type":"Team","id":2},"minimum_approvals":3,"file_patterns":[]}]}}]`)
}

func TestTeamApprovalsProjectorBindsCountsWithoutChangingPatterns(t *testing.T) {
	for _, count := range []int{-1, 0, 1, 10, 11, 100} {
		raw := teamApprovalsRulesJSON(strconv.Itoa(count))
		original := bytes.Clone(raw)
		got, records, err := prepareRulesetTeamApprovalsRules(raw)
		var want []settings.RulesetTeamApprovalsNormalization
		effective := count
		if count < 0 || count > 10 {
			effective = 0
			want = []settings.RulesetTeamApprovalsNormalization{{ReviewerID: 10, Path: settings.RulesetTeamApprovalsPath, RawValue: count, Fallback: 0}}
		}
		if err != nil || !reflect.DeepEqual(records, want) || !bytes.Equal(raw, original) {
			t.Fatal(count, records, err)
		}
		var before, after any
		json.Unmarshal(teamApprovalsRulesJSON(strconv.Itoa(effective)), &before)
		json.Unmarshal(got, &after)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("identity/patterns/canonical companion changed", string(got))
		}
		got[0] = '!'
		if !bytes.Equal(raw, original) {
			t.Fatal("source alias")
		}
	}
	for _, raw := range []string{`[]`, `[{"type":"deletion"}]`, `[{"type":"pull_request","parameters":{"required_reviewers":[]}}]`} {
		got, records, err := prepareRulesetTeamApprovalsRules(json.RawMessage(raw))
		var before, after any
		json.Unmarshal([]byte(raw), &before)
		json.Unmarshal(got, &after)
		if err != nil || len(records) != 0 || !reflect.DeepEqual(before, after) {
			t.Fatal("absence invented reviewer", string(got), records, err)
		}
	}
}

func TestTeamApprovalsProjectorRejectsMissingTypedCountsAndInvalidIdentity(t *testing.T) {
	invalid := []string{`null`, `{}`, `[null]`, `[{"type":"pull_request"}]`, `[{"type":"pull_request","parameters":{}}]`, `[{"type":"pull_request","parameters":{"required_reviewers":null}}]`, `[{"type":"pull_request","parameters":{"required_reviewers":[null]}}]`}
	for _, raw := range []string{`null`, `"11"`, `true`, `11.5`, `1e100`, `9223372036854775808`} {
		invalid = append(invalid, string(teamApprovalsRulesJSON(raw)))
	}
	base := string(teamApprovalsRulesJSON("11"))
	for _, pair := range [][2]string{{`"minimum_approvals":11,`, ``}, {`"id":10`, `"id":0`}, {`"id":10`, `"id":-1`}, {`"id":10`, `"id":null`}, {`"id":10`, `"id":"10"`}, {`"id":10`, `"id":9223372036854775808`}, {`"type":"Team"`, `"type":"User"`}, {`"id":2`, `"id":10`}} {
		invalid = append(invalid, strings.Replace(base, pair[0], pair[1], 1))
	}
	invalid = append(invalid, strings.TrimSuffix(base, "]")+","+strings.TrimPrefix(string(teamApprovalsRulesJSON("12")), "["))
	var rules []map[string]any
	json.Unmarshal([]byte(base), &rules)
	reviewers := rules[0]["parameters"].(map[string]any)["required_reviewers"].([]any)
	for id := 100; id < 114; id++ {
		reviewers = append(reviewers, map[string]any{"reviewer": map[string]any{"type": "Team", "id": id}, "minimum_approvals": 0, "file_patterns": []any{}})
	}
	rules[0]["parameters"].(map[string]any)["required_reviewers"] = reviewers
	encoded, _ := json.Marshal(rules)
	invalid = append(invalid, string(encoded))
	for _, raw := range invalid {
		input := json.RawMessage(raw)
		original := bytes.Clone(input)
		got, records, err := prepareRulesetTeamApprovalsRules(input)
		if err == nil || got != nil || len(records) != 0 || !bytes.Equal(input, original) {
			t.Fatal("malformed partial projection", raw, string(got), records, err)
		}
	}
	// Pure count conversion is not proof of full policy/pattern acquisition.
	body := teamApprovalsPayload("equal")
	params := body["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)
	reviewer := params["required_reviewers"].([]any)[2].(map[string]any)
	reviewer["minimum_approvals"] = 11
	delete(reviewer, "file_patterns")
	encoded, _ = json.Marshal(body)
	var raw map[string]json.RawMessage
	json.Unmarshal(encoded, &raw)
	got, records, err := prepareRulesetTeamApprovalsRules(raw["rules"])
	if err != nil || len(records) != 1 {
		t.Fatal(records, err)
	}
	raw["rules"] = got
	if _, err := decodeRulesetPolicy(raw); err == nil {
		t.Fatal("projection legalized missing patterns")
	}
}
