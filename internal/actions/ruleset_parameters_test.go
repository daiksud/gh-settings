package actions_test

import (
	"encoding/json"
	"html"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRulesetReviewReportPreservesOrderFalseAndZero(t *testing.T) {
	d, e := rulesetReportFixture([]byte(`
- name: '<review>'
  target: branch
  enforcement: active
  conditions: {ref_name: {include: ['~ALL'], exclude: []}}
  bypass_actors: []
  rules:
    - type: pull_request
      parameters:
        dismiss_stale_reviews_on_push: true
        require_code_owner_review: true
        require_last_push_approval: false
        required_approving_review_count: 1
        required_review_thread_resolution: true
        required_reviewers: [{reviewer: {type: Team, id: 21}, minimum_approvals: 0, file_patterns: ['**', '!docs/**', '**']}]
    - type: required_status_checks
      parameters:
        required_status_checks: [{context: '<ci>', integration_id: 9}]
        strict_required_status_checks_policy: true
`))
	if e != nil {
		t.Fatal(e)
	}
	before := d.Rulesets.Values()[0]
	after := d.Rulesets.Values()[0]
	id := int64(7)
	before.ID, after.ID = &id, &id
	for i := range *after.Rules {
		r := &(*after.Rules)[i]
		if r.Type == "pull_request" {
			zero := 0
			no := false
			r.Parameters.RequiredApprovingReviewCount = &zero
			r.Parameters.RequireCodeOwnerReview = &no
			patterns := (*r.Parameters.RequiredReviewers)[0].FilePatterns
			(*patterns)[0], (*patterns)[1] = (*patterns)[1], (*patterns)[0]
		} else if r.Type == "required_status_checks" {
			empty := []settings.RequiredStatusCheck{}
			r.Parameters.RequiredStatusChecks = &empty
		}
	}
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Rulesets: []actions.RulesetPlanChange{{Operation: "update", Before: &before, After: &after}}}
	data, e := json.Marshal(report)
	if e != nil {
		t.Fatal(e)
	}
	var round actions.PlanReport
	if e := json.Unmarshal(data, &round); e != nil {
		t.Fatal(e)
	}
	rendered, e := actions.RenderSummary(round, summaryMetadata())
	if e != nil {
		t.Fatal(e)
	}
	visible := html.UnescapeString(rendered)
	for _, want := range []string{`"required_approving_review_count":0`, `"require_code_owner_review":false`, `"required_status_checks":[]`, `"file_patterns":["!docs/**","**","**"]`} {
		if !strings.Contains(visible, want) {
			t.Fatal("missing typed/order value", want, visible)
		}
	}
	if strings.Contains(rendered, "<review>") || strings.Contains(rendered, "<ci>") {
		t.Fatal("unescaped report")
	}
	// Reports remain a closed presentation protocol, never a raw policy passthrough.
	writeOnly := strings.Replace(string(data), `"require_code_owner_review":true`, `"require_code_owner_review":true,"dismissal_restriction":{"enabled":true,"allowed_actors":[]}`, 1)
	if writeOnly == string(data) {
		t.Fatal("missing injection target")
	}
	if err := json.Unmarshal([]byte(writeOnly), &round); err != nil {
		t.Fatal(err)
	}
	if _, err := actions.RenderSummary(round, summaryMetadata()); err == nil {
		t.Fatal("plan report accepted an unsupported write-only value")
	}
	for _, pair := range [][2]string{{`"minimum_approvals":0`, `"minimum_approvals":null`}, {`"strict_required_status_checks_policy":true`, `"Strict_required_status_checks_policy":true`}, {`"required_approving_review_count":0`, `"required_approving_review_count":0,"future":false`}} {
		invalid := strings.Replace(string(data), pair[0], pair[1], 1)
		if invalid == string(data) {
			t.Fatal(pair)
		}
		if e := json.Unmarshal([]byte(invalid), &round); e == nil {
			t.Fatal("invalid nested report accepted", pair)
		}
	}
}
