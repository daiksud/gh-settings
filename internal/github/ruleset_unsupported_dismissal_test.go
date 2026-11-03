package github

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDismissalRestrictionStopsCompleteRulesetAcquisition(t *testing.T) {
	for _, restriction := range []string{`null`, `{}`, `{"enabled":false,"allowed_actors":[]}`, `{"enabled":true,"allowed_actors":[{"type":"Team","id":21}]}`} {
		t.Run(restriction, func(t *testing.T) {
			body := strings.Replace(fullReviewRulesetJSON, `"require_code_owner_review":true`, `"require_code_owner_review":true,"dismissal_restriction":`+restriction, 1)
			if body == fullReviewRulesetJSON {
				t.Fatal("missing injection target")
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal([]byte(body), &raw); err != nil {
				t.Fatal(err)
			}
			if _, _, err := decodeObservedRulesetPolicy(raw); err == nil || !strings.Contains(err.Error(), "dismissal_restriction") {
				t.Fatalf("acquired unsupported protection: %v", err)
			}
			if _, err := decodeRulesetPolicy(raw); err == nil || !strings.Contains(err.Error(), "dismissal_restriction") {
				t.Fatalf("strict readback acquired unsupported protection: %v", err)
			}
		})
	}
}
