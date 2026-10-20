package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

const fullReviewRulesetJSON = `{"id":7,"name":"default","source_type":"Repository","source":"owner/current","current_user_can_bypass":"never","target":"branch","enforcement":"active","bypass_actors":[],"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"rules":[{"type":"deletion"},{"type":"non_fast_forward"},{"type":"required_linear_history"},{"type":"pull_request","parameters":{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":true,"require_code_owner_review":true,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":1,"required_review_thread_resolution":true,"dismissal_restriction":{"enabled":true,"allowed_actors":[{"type":"Team","id":21}]},"required_reviewers":[{"reviewer":{"type":"Team","id":22},"minimum_approvals":1,"file_patterns":["**","!docs/**","docs/README.md","**"]}]}},{"type":"required_status_checks","parameters":{"do_not_enforce_on_create":false,"required_status_checks":[{"context":"pass","integration_id":15368},{"context":"gh-settings","integration_id":15368},{"context":"any-source","integration_id":null}],"strict_required_status_checks_policy":true}}]}`

func TestReadReviewRulesetCompleteAndAnySourceIdentity(t *testing.T) {
	policy, e := decodeRulesetPolicyFromTest([]byte(fullReviewRulesetJSON))
	if e != nil {
		t.Fatal(e)
	}
	for _, r := range *policy.Rules {
		switch r.Type {
		case "pull_request":
			if *r.Parameters.RequireExtraApprovalForUnattributedChanges || !reflect.DeepEqual(*(*r.Parameters.RequiredReviewers)[0].FilePatterns, []string{"**", "!docs/**", "docs/README.md", "**"}) {
				t.Fatal("read changed policy", r)
			}
		case "required_status_checks":
			checks := *r.Parameters.RequiredStatusChecks
			if checks[2].IntegrationID != nil {
				t.Fatal("null integration not any-source")
			}
		}
	}
}
func TestReadReviewRulesetRejectsNullUnknownAndIncomplete(t *testing.T) {
	for _, pair := range [][2]string{{`"require_last_push_approval":false,`, ``}, {`"required_approving_review_count":1`, `"required_approving_review_count":"1"`}, {`"allowed_merge_methods":["squash"]`, `"allowed_merge_methods":null`}, {`"require_extra_approval_for_unattributed_changes":false`, `"require_extra_approval_for_unattributed_changes":null`}, {`"require_last_push_approval":false`, `"require_last_push_approval":false,"future_switch":false`}, {`"enabled":true,`, `"enabled":true,"future":0,`}, {`"type":"Team","id":21`, `"type":"Team","id":21,"future":"x"`}, {`"minimum_approvals":1`, `"minimum_approvals":null`}, {`"type":"Team","id":22`, `"type":"Team","id":22,"future":true`}, {`"file_patterns":["**","!docs/**","docs/README.md","**"]`, `"file_patterns":null`}, {`"context":"pass","integration_id":15368`, `"context":"pass","integration_id":15368,"future":false`}, {`"strict_required_status_checks_policy":true`, `"strict_required_status_checks_policy":null`}, {`"do_not_enforce_on_create":false`, `"do_not_enforce_on_create":null`}, {`"required_status_checks":[`, `"Required_status_checks":[`}} {
		body := strings.Replace(fullReviewRulesetJSON, pair[0], pair[1], 1)
		if body == fullReviewRulesetJSON {
			t.Fatal(pair)
		}
		if _, e := decodeRulesetPolicyFromTest([]byte(body)); e == nil {
			t.Fatal("accepted lossy response", pair)
		}
	}
}
func TestReviewRulesetPartialEditExactPutPreservesCompanions(t *testing.T) {
	before, e := decodeRulesetPolicyFromTest([]byte(fullReviewRulesetJSON))
	if e != nil {
		t.Fatal(e)
	}
	obs, e := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: before}}, 0)
	if e != nil {
		t.Fatal(e)
	}
	d, e := workspace.DecodeConfig([]byte(`schemaVersion: 1
settings:
  rulesets:
    - name: default
      rules:
        - type: deletion
        - type: non_fast_forward
        - type: required_linear_history
        - type: pull_request
          parameters:
            required_approving_review_count: 0
            require_code_owner_review: false
            required_reviewers:
              - reviewer: {type: Team, id: 22}
                file_patterns: ['!docs/**', '**', 'docs/README.md', '**']
        - type: required_status_checks
          parameters: {required_status_checks: [], strict_required_status_checks_policy: false}
`))
	if e != nil {
		t.Fatal(e)
	}
	changes, e := settings.ReconcileRulesets(obs, d.Rulesets)
	if e != nil || len(changes) != 1 {
		t.Fatal(changes, e)
	}
	calls := 0
	client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/rulesets/7" {
			t.Fatal(r.Method, r.URL)
		}
		data, _ := io.ReadAll(r.Body)
		var body map[string]any
		if e := json.Unmarshal(data, &body); e != nil {
			t.Fatal(e)
		}
		if len(body) != 6 || body["source"] != nil || body["id"] != nil {
			t.Fatal("metadata replay", string(data))
		}
		for _, value := range body["rules"].([]any) {
			rule := value.(map[string]any)
			if rule["type"] == "pull_request" {
				p := rule["parameters"].(map[string]any)
				if p["required_approving_review_count"] != float64(0) || p["require_code_owner_review"] != false || p["dismiss_stale_reviews_on_push"] != true || p["require_extra_approval_for_unattributed_changes"] != false || p["require_last_push_approval"] != false || p["required_review_thread_resolution"] != true {
					t.Fatal("lost companions", p)
				}
				reviewer := p["required_reviewers"].([]any)[0].(map[string]any)
				if reviewer["minimum_approvals"] != float64(1) || !reflect.DeepEqual(reviewer["file_patterns"], []any{"!docs/**", "**", "docs/README.md", "**"}) {
					t.Fatal("reviewer changed", reviewer)
				}
				if p["dismissal_restriction"] == nil || p["allowed_merge_methods"] == nil {
					t.Fatal("optional siblings lost", p)
				}
			}
			if rule["type"] == "required_status_checks" {
				p := rule["parameters"].(map[string]any)
				if len(p["required_status_checks"].([]any)) != 0 || p["strict_required_status_checks_policy"] != false || p["do_not_enforce_on_create"] != false {
					t.Fatal("checks lost", p)
				}
			}
		}
		return labelsResponse(r, 200, string(data)), nil
	}))
	if e := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, changes[0]); e != nil || calls != 1 {
		t.Fatal(calls, e)
	}
	after := *changes[0].After()
	afterObs, _ := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: after}}, 0)
	if again, e := settings.ReconcileRulesets(afterObs, d.Rulesets); e != nil || len(again) != 0 {
		t.Fatal("repeat writes", again, e)
	}
}
func TestReadReviewRulesetLaterDetailFailureDiscardsObservation(t *testing.T) {
	client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/rulesets") {
			return labelsResponse(r, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":8,"source_type":"Repository","source":"owner/current"}]`), nil
		}
		if strings.HasSuffix(r.URL.Path, "/7") {
			return labelsResponse(r, 200, fullReviewRulesetJSON), nil
		}
		return labelsResponse(r, 403, fmt.Sprintf(`{"message":%q}`, "hidden")), nil
	}))
	observed, e := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
	if _, valueErr := observed.Values(); e == nil || valueErr == nil {
		t.Fatal("partial review policy returned", observed, e)
	}
}

func TestReviewRulesetResponseRefusesSilentPolicyLoss(t *testing.T) {
	before, e := decodeRulesetPolicyFromTest([]byte(fullReviewRulesetJSON))
	if e != nil {
		t.Fatal(e)
	}
	after := before
	zero := 0
	for i := range *after.Rules {
		r := &(*after.Rules)[i]
		if r.Type == "pull_request" {
			p := *r.Parameters
			p.RequiredApprovingReviewCount = &zero
			r.Parameters = &p
		}
	}
	afterData, _ := json.Marshal(after)
	for _, pair := range [][2]string{{`"require_extra_approval_for_unattributed_changes":false,`, ``}, {`"required_approving_review_count":0`, `"required_approving_review_count":1`}, {`"require_extra_approval_for_unattributed_changes":false`, `"require_extra_approval_for_unattributed_changes":true`}, {`"integration_id":15368`, `"integration_id":null`}, {`"file_patterns":["**","!docs/**","docs/README.md","**"]`, `"file_patterns":["!docs/**","**","docs/README.md","**"]`}} {
		bad := strings.Replace(string(afterData), pair[0], pair[1], 1)
		if bad == string(afterData) {
			t.Fatal("bad response fixture", pair)
		}
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(bad), &raw)
		if e := verifyReviewRulesetResponse(after, raw); e == nil {
			t.Fatal("silent policy loss accepted", pair)
		}
	}
	// Optional default fields absent from a new declaration may appear in a response.
	desired := after
	for i := range *desired.Rules {
		r := &(*desired.Rules)[i]
		if r.Type == "pull_request" {
			p := *r.Parameters
			p.AllowedMergeMethods = nil
			r.Parameters = &p
		}
	}
	var response map[string]json.RawMessage
	json.Unmarshal(afterData, &response)
	if e := verifyReviewRulesetResponse(desired, response); e != nil {
		t.Fatal("new optional default treated as managed", e)
	}
}

func TestCreateReviewRulesetExactPolicyAndResponse(t *testing.T) {
	policy, err := decodeRulesetPolicyFromTest([]byte(fullReviewRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	change, err := settings.NewRulesetChange("create", 0, nil, &policy)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := json.Marshal(change.After())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		data, err := io.ReadAll(r.Body)
		var got, want any
		json.Unmarshal(data, &got)
		json.Unmarshal(expected, &want)
		if err != nil || r.Method != "POST" || r.URL.Path != "/repos/owner/current/rulesets" || !reflect.DeepEqual(got, want) {
			t.Fatal(r.Method, r.URL, string(data), err)
		}
		return labelsResponse(r, 201, string(data)), nil
	}))
	if err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change); err != nil || calls != 1 {
		t.Fatal(calls, err)
	}
}
