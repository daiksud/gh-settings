package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIRulesetCollectionPresenceAndIdentity(t *testing.T) {
	if got, err := RulesetsDeclaration(nil); err != nil || got != nil {
		t.Fatal(got, err)
	}
	empty := []uimodel.Ruleset{}
	if got, err := RulesetsDeclaration(&empty); err != nil || got == nil || got.Len() != 0 {
		t.Fatal(got, err)
	}
	policy := apiRulesetBase()
	zulu := apiRulesetBase()
	zulu.Name = "Zulu"
	observed, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 8, Policy: zulu}, {ID: 7, Policy: policy}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RulesetsObservation(observed)
	if err != nil || got.ExcludedPush != 2 || len(got.Rulesets) != 2 || got.Rulesets[0].ID == nil || *got.Rulesets[0].ID != 7 || got.Rulesets[1].RulesetName != "Zulu" {
		t.Fatal(got, err)
	}
	if _, err := RulesetsObservation(settings.RulesetObservation{}); err == nil {
		t.Fatal("missing acquisition accepted")
	}
	p := completeUIAPIPR()
	p.DismissalRestriction = &settings.ReviewDismissalRestriction{Enabled: rulesetPtr(true), AllowedActors: rulesetPtr([]settings.ReviewActor{{Type: "RepositoryRole", ID: 99}})}
	policy.Name = "Unrepresentable"
	policy.Rules = rulesetPtr([]settings.RulesetRule{{Type: "pull_request", Parameters: p}})
	partial, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: apiRulesetBase()}, {ID: 8, Policy: policy}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := RulesetsObservation(partial); err == nil || !reflect.DeepEqual(got, PreparedRulesets{}) {
		t.Fatal("partial UI collection returned", got, err)
	}
}

func TestUIRulesetsRawReadComposition(t *testing.T) {
	var fields map[string]any
	if err := json.Unmarshal([]byte(uiRulesetExpectedPayload), &fields); err != nil {
		t.Fatal(err)
	}
	fields["id"] = 7
	fields["source_type"] = "Repository"
	fields["source"] = "owner/current"
	fields["current_user_can_bypass"] = "never"
	body, _ := json.Marshal(fields)
	client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
			t.Fatal(req.Method, req.URL)
		}
		switch req.URL.Path {
		case "/repos/owner/current/rulesets":
			return labelsResponse(req, 200, `[{"id":7,"name":"Policy","source_type":"Repository","source":"owner/current"},{"id":8,"name":"Push","source_type":"Repository","source":"owner/current"},{"id":9,"name":"Inherited","source_type":"Organization","source":"owner"}]`), nil
		case "/repos/owner/current/rulesets/7":
			return labelsResponse(req, 200, string(body)), nil
		case "/repos/owner/current/rulesets/8":
			return labelsResponse(req, 200, `{"id":8,"source_type":"Repository","source":"owner/current","target":"push"}`), nil
		default:
			t.Fatal(req.URL)
			return nil, nil
		}
	})
	raw, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := RulesetsObservation(raw)
	if err != nil || len(got.Rulesets) != 1 || got.ExcludedPush != 1 {
		t.Fatal(got, err)
	}
	ui := got.Rulesets[0]
	if ui.ID == nil || *ui.ID != 7 || ui.RulesetName != "Policy" || *ui.BypassList == nil || *(*ui.BypassList)[0].Role != "maintain" || *ui.Rules.RequireCodeQualityResults.Severity != "warnings_and_higher" || (*ui.Rules.RequireCodeScanningResults.CodeScanningTools)[0].ToolName != "CodeQL" || *ui.Rules.RequireAPullRequestBeforeMerging.RequiredApprovals != 1 || *(*ui.Rules.RequireStatusChecksToPass.RequiredStatusChecks)[0].Source.App != 17 {
		t.Fatalf("mapped policy=%+v", ui)
	}
}

const uiRulesetExpectedPayload = `{
 "name":"Policy","target":"branch","enforcement":"active",
 "conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},
 "bypass_actors":[{"actor_type":"RepositoryRole","actor_id":2,"bypass_mode":"always"}],
 "rules":[
  {"type":"code_quality","parameters":{"severity":"warnings"}},
  {"type":"code_scanning","parameters":{"code_scanning_tools":[{"tool":"CodeQL","alerts_threshold":"errors","security_alerts_threshold":"high_or_higher"}]}},
  {"type":"deletion"}, {"type":"non_fast_forward"},
  {"type":"pull_request","parameters":{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":true,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":1,"required_review_thread_resolution":true,"required_reviewers":[{"reviewer":{"type":"Team","id":12},"minimum_approvals":2,"file_patterns":["*.go","!generated/*"]}]}},
  {"type":"required_linear_history"}, {"type":"required_signatures"},
  {"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"ci","integration_id":17}],"strict_required_status_checks_policy":true,"do_not_enforce_on_create":false}}
 ]
}`

func TestUIRulesetExactCreateAndSelectedUpdate(t *testing.T) {
	for _, update := range []bool{false, true} {
		ui := uiRulesetBase()
		ui.TargetBranches.IncludeDefaultBranch = rulesetPtr(true)
		ui.BypassList = rulesetPtr([]uimodel.BypassEntry{{Role: rulesetPtr(uimodel.BypassRole("maintain")), BypassMode: rulesetPtr(uimodel.BypassMode("always"))}})
		ui.Rules = uimodel.RulesetRules{RestrictDeletions: rulesetPtr(true), BlockForcePushes: rulesetPtr(true), RequireLinearHistory: rulesetPtr(true), RequireSignedCommits: rulesetPtr(true),
			RequireAPullRequestBeforeMerging: &uimodel.PullRequestRule{RequiredApprovals: rulesetPtr(1), AllowedMergeMethods: rulesetPtr([]uimodel.MergeMethod{"squash"}), RequireReviewFromCodeOwners: rulesetPtr(true), RequireConversationResolutionBeforeMerging: rulesetPtr(true), RequireReviewFromSpecificTeams: rulesetPtr([]uimodel.TeamReview{{Reviewer: 12, Approvals: rulesetPtr(2), FilePatterns: rulesetPtr([]string{"*.go", "!generated/*"})}})},
			RequireStatusChecksToPass:        &uimodel.StatusChecksRule{RequiredStatusChecks: rulesetPtr([]uimodel.StatusCheck{{Check: "ci", Source: uimodel.CheckSource{App: rulesetPtr(int64(17))}}}), RequireBranchesToBeUpToDateBeforeMerging: rulesetPtr(true)},
			RequireCodeScanningResults:       &uimodel.CodeScanningRule{CodeScanningTools: rulesetPtr([]uimodel.CodeScanningTool{{ToolName: "CodeQL", Alerts: "errors", SecurityAlerts: "high_or_higher"}})}, RequireCodeQualityResults: &uimodel.CodeQualityRule{Severity: rulesetPtr(uimodel.CodeQualitySeverity("warnings_and_higher"))},
		}
		states := []settings.RulesetState{}
		id, method, path := int64(19), "POST", "/repos/owner/current/rulesets"
		if update {
			ui.ID = rulesetPtr(int64(7))
			id, method, path = 7, "PUT", "/repos/owner/current/rulesets/7"
			other := apiRulesetBase()
			other.Name = "Unlisted"
			states = []settings.RulesetState{{ID: 7, Policy: apiRulesetBase()}, {ID: 8, Policy: other}}
		}
		observed, err := settings.NewRulesetObservation(states, 0)
		if err != nil {
			t.Fatal(err)
		}
		desired, err := RulesetsDeclaration(rulesetPtr([]uimodel.Ruleset{ui}))
		if err != nil {
			t.Fatal(err)
		}
		changes, err := settings.ReconcileRulesets(observed, desired)
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		var want map[string]any
		if err := json.Unmarshal([]byte(uiRulesetExpectedPayload), &want); err != nil {
			t.Fatal(err)
		}
		response := map[string]any{}
		if err := json.Unmarshal([]byte(uiRulesetExpectedPayload), &response); err != nil {
			t.Fatal(err)
		}
		response["id"] = id
		response["source_type"] = "Repository"
		response["source"] = "owner/current"
		response["current_user_can_bypass"] = "never"
		body, _ := json.Marshal(response)
		calls := 0
		client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				var got map[string]any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if req.Method != method || req.URL.Path != path || !reflect.DeepEqual(got, want) {
					t.Fatalf("request=%s %s payload=%v want %v", req.Method, req.URL, got, want)
				}
			} else if calls != 2 || req.Method != "GET" || req.URL.Path != fmt.Sprintf("/repos/owner/current/rulesets/%d", id) || req.URL.Query().Get("includes_parents") != "false" {
				t.Fatal("unexpected readback", req.Method, req.URL)
			}
			return labelsResponse(req, 200, string(body)), nil
		})
		gotID, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, changes[0])
		if err != nil || gotID != id || calls != 2 {
			t.Fatal(gotID, calls, err)
		}
	}
}
