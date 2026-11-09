package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

// Existing response/readback tests supply a successful independent preflight.
// Dedicated unowned-protection tests exercise the real preflight failures/drift.
func rulesetMutationClient(t *testing.T, change settings.RulesetChange, next transportFunc) *Client {
	t.Helper()
	preflight := change.Operation() == "update"
	return topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		if !preflight {
			return next(req)
		}
		preflight = false
		if req.Method != "GET" || req.URL.Path != fmt.Sprintf("/repos/owner/current/rulesets/%d", change.ID()) || req.URL.Query().Get("includes_parents") != "false" {
			t.Fatal("missing independent preflight", req.Method, req.URL)
		}
		data, err := json.Marshal(change.Before())
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		body["id"], body["source_type"], body["source"] = change.ID(), "Repository", "owner/current"
		if record := change.Normalizations().QualitySeverity; record != (settings.ObservationNormalization{}) {
			for _, item := range body["rules"].([]any) {
				rule := item.(map[string]any)
				if rule["type"] == "code_quality" {
					rule["parameters"].(map[string]any)["severity"] = record.RawValue
				}
			}
		}
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		return labelsResponse(req, 200, string(data)), nil
	}))
}
