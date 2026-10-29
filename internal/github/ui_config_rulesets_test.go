package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

const rulesetUIConfig = `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - ruleset_name: Policy
        enforcement_status: active
        target_branches: {include_default_branch: true}
        bypass_list: [{role: maintain, bypass_mode: always}]
        rules:
          restrict_deletions: true
          block_force_pushes: true
          require_linear_history: true
          require_signed_commits: true
          require_a_pull_request_before_merging:
            required_approvals: 1
            allowed_merge_methods: [squash]
            dismiss_stale_pull_request_approvals_when_new_commits_are_pushed: false
            require_review_from_code_owners: true
            require_approval_of_the_most_recent_reviewable_push: false
            require_conversation_resolution_before_merging: true
            require_an_additional_approval_for_unattributed_copilot_pull_requests: false
            require_review_from_specific_teams:
              - {reviewer: 12, approvals: 2, file_patterns: ['*.go', '!generated/*']}
          require_status_checks_to_pass:
            required_status_checks: [{check: ci, source: {app: 17}}]
            require_branches_to_be_up_to_date_before_merging: true
            do_not_require_status_checks_on_creation: false
          require_code_scanning_results:
            code_scanning_tools: [{tool_name: CodeQL, security_alerts: high_or_higher, alerts: errors}]
          require_code_quality_results: {severity: warnings_and_higher}
`

func TestUIConfigRulesetExactRequests(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprintf("selected=%t", update), func(t *testing.T) {
			input := rulesetUIConfig
			if update {
				input = strings.Replace(input, "      - ruleset_name: Policy", "      - id: 7\n        ruleset_name: Policy", 1)
			}
			ui, err := workspace.DecodeUIConfig([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			id, method, path := int64(19), "POST", "/repos/owner/current/rulesets"
			states := []settings.RulesetState{}
			if update {
				id, method, path = 7, "PUT", "/repos/owner/current/rulesets/7"
				unlisted := apiRulesetBase()
				unlisted.Name = "Unlisted"
				states = []settings.RulesetState{{ID: 7, Policy: apiRulesetBase()}, {ID: 8, Policy: unlisted}}
			}
			desired, err := DeclarationFromUI(ui)
			if err != nil {
				t.Fatal(err)
			}
			observed, err := settings.NewRulesetObservation(states, 0)
			if err != nil {
				t.Fatal(err)
			}
			changes, err := settings.ReconcileRulesets(observed, desired.Rulesets)
			if err != nil || len(changes) != 1 {
				t.Fatalf("changes=%v error=%v", changes, err)
			}
			var want, response map[string]any
			if err := json.Unmarshal([]byte(uiRulesetExpectedPayload), &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(uiRulesetExpectedPayload), &response); err != nil {
				t.Fatal(err)
			}
			response["id"], response["source_type"], response["source"] = id, "Repository", "owner/current"
			response["current_user_can_bypass"] = "never"
			body, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					var got map[string]any
					if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
						t.Fatal(err)
					}
					if req.Method != method || req.URL.Path != path || !reflect.DeepEqual(got, want) {
						t.Fatalf("request=%s %s payload=%v; want %v", req.Method, req.URL, got, want)
					}
				} else if calls != 2 || req.Method != "GET" || req.URL.Path != fmt.Sprintf("/repos/owner/current/rulesets/%d", id) || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatalf("unexpected readback: %s %s", req.Method, req.URL)
				}
				return labelsResponse(req, 200, string(body)), nil
			})
			gotID, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, changes[0])
			if err != nil || gotID != id || calls != 2 {
				t.Fatalf("id=%d calls=%d error=%v", gotID, calls, err)
			}
		})
	}
}
