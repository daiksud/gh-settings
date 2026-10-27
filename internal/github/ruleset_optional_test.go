package github

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestOptionalDismissalResponsePresence(t *testing.T) {
	const original = `"dismissal_restriction":{"enabled":true,"allowed_actors":[{"type":"Team","id":21}]},`
	for _, tc := range []struct {
		name, replacement string
		valid, absent     bool
	}{
		{"absent", "", true, true},
		{"null", `"dismissal_restriction":null,`, false, false},
		{"empty", `"dismissal_restriction":{},`, false, false},
		{"missing enabled", `"dismissal_restriction":{"allowed_actors":[]},`, false, false},
		{"missing actors", `"dismissal_restriction":{"enabled":false},`, false, false},
		{"null actors", `"dismissal_restriction":{"enabled":false,"allowed_actors":null},`, false, false},
		{"disabled", `"dismissal_restriction":{"enabled":false,"allowed_actors":[]},`, true, false},
		{"enabled", original, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.Replace(fullReviewRulesetJSON, original, tc.replacement, 1)
			policy, err := decodeRulesetPolicyFromTest([]byte(body))
			if !tc.valid {
				if err == nil {
					t.Fatal("malformed present protection accepted", body)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			observed, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: policy}}, 0)
			if err != nil {
				t.Fatal(err)
			}
			exported, err := observed.ExportDeclarations()
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(exported.Values())
			if err != nil || strings.Contains(string(encoded), `"dismissal_restriction"`) {
				t.Fatalf("write-only field escaped export: %s %v", encoded, err)
			}
			_, err = settings.ReconcileRulesets(observed, &exported)
			if tc.absent {
				if err != nil {
					t.Fatalf("omitted wire value prevented observable round trip: %v", err)
				}
				complete, err := decodeRulesetPolicyFromTest([]byte(fullReviewRulesetJSON))
				if err != nil {
					t.Fatal(err)
				}
				var raw map[string]json.RawMessage
				if err := json.Unmarshal([]byte(body), &raw); err != nil {
					t.Fatal(err)
				}
				if err := verifyReviewRulesetResponse(complete, raw); err != nil {
					t.Fatal("write-only omission blocked API acceptance", err)
				}
			} else if err != nil {
				t.Fatalf("explicit observable policy rejected: %v", err)
			}
		})
	}
}
