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

func TestRulesetRequiredApprovalsProjectorOwnsOnlyAcquiredInteger(t *testing.T) {
	for _, count := range []int{0, 1, 10, -1, 11, 100} {
		var policy map[string]json.RawMessage
		json.Unmarshal([]byte(fullReviewRulesetJSON), &policy)
		raw := json.RawMessage(strings.ReplaceAll(string(policy["rules"]), `"required_approving_review_count":1`, `"required_approving_review_count":`+strconv.Itoa(count)))
		before := bytes.Clone(raw)
		got, record, err := prepareRulesetRequiredApprovalsRules(raw)
		effective := count
		want := settings.RulesetRequiredApprovalsNormalization{}
		if count < 0 || count > 10 {
			effective = 0
			want = settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: count, Fallback: 0}
		}
		if err != nil || record != want || !bytes.Equal(raw, before) {
			t.Fatal("typed projection/source", record, err)
		}
		policy["rules"] = got
		decoded, err := decodeRulesetPolicy(policy)
		if err != nil {
			t.Fatal("strict projected policy", err)
		}
		found := false
		for _, r := range *decoded.Rules {
			if r.Type == "pull_request" {
				found = true
				if *r.Parameters.RequiredApprovingReviewCount != effective {
					t.Fatal("wrong effective count", r)
				}
			}
		}
		if !found {
			t.Fatal("lost PR rule")
		}
		got[0] = '!'
		if !bytes.Equal(raw, before) {
			t.Fatal("output aliases input")
		}
	}
	for _, raw := range []string{`[]`, `[{"type":"deletion"}]`} {
		got, record, err := prepareRulesetRequiredApprovalsRules(json.RawMessage(raw))
		var before, after any
		json.Unmarshal([]byte(raw), &before)
		json.Unmarshal(got, &after)
		if err != nil || record != (settings.RulesetRequiredApprovalsNormalization{}) || !reflect.DeepEqual(before, after) {
			t.Fatal("absence invented PR count", string(got), record, err)
		}
	}
}

func TestRulesetRequiredApprovalsProjectorRejectsMalformedAcquisition(t *testing.T) {
	for _, raw := range []string{
		`null`, `{}`, `[null]`, `[{"type":null}]`, `[{"type":"pull_request"}]`, `[{"type":"pull_request","parameters":null}]`, `[{"type":"pull_request","parameters":{}}]`,
		`[{"type":"pull_request","parameters":{"required_approving_review_count":null}}]`,
		`[{"type":"pull_request","parameters":{"required_approving_review_count":"11"}}]`,
		`[{"type":"pull_request","parameters":{"required_approving_review_count":true}}]`,
		`[{"type":"pull_request","parameters":{"required_approving_review_count":11.5}}]`,
		`[{"type":"pull_request","parameters":{"required_approving_review_count":1e100}}]`,
		`[{"type":"pull_request","parameters":{"required_approving_review_count":9223372036854775808}}]`,
		`[{"type":"pull_request","parameters":{"required_approving_review_count":11}},{"type":"pull_request","parameters":{"required_approving_review_count":12}}]`,
	} {
		input := json.RawMessage(raw)
		before := bytes.Clone(input)
		got, record, err := prepareRulesetRequiredApprovalsRules(input)
		if err == nil || got != nil || record != (settings.RulesetRequiredApprovalsNormalization{}) || !bytes.Equal(input, before) {
			t.Fatal("partial/invalid count escaped", raw, string(got), record, err)
		}
	}
	// Projection is not acquisition proof: unrelated unsupported members still reach strict complete validation.
	var policy map[string]json.RawMessage
	json.Unmarshal([]byte(fullReviewRulesetJSON), &policy)
	raw := strings.Replace(string(policy["rules"]), `"required_approving_review_count":1`, `"required_approving_review_count":11,"future":true`, 1)
	got, record, err := prepareRulesetRequiredApprovalsRules(json.RawMessage(raw))
	if err != nil || record.RawValue != 11 {
		t.Fatal(record, err)
	}
	policy["rules"] = got
	if _, err := decodeRulesetPolicy(policy); err == nil {
		t.Fatal("projection legalized unrelated unsupported field")
	}
}
