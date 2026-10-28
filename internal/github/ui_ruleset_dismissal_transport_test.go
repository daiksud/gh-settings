package github

import (
	"encoding/json"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"net/http"
	"reflect"
	"testing"
)

const uiDismissalWire = `{"name":"Policy","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"pull_request","parameters":{"required_approving_review_count":0,"allowed_merge_methods":["merge","rebase","squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":false,"require_last_push_approval":false,"required_review_thread_resolution":false,"require_extra_approval_for_unattributed_changes":false,"required_reviewers":[],"dismissal_restriction":{"enabled":true,"allowed_actors":[{"type":"IntegrationInstallation","id":13},{"type":"RepositoryRole","id":5},{"type":"Team","id":12},{"type":"User","id":11}]}}}]}`

func uiDismissalResponse(t *testing.T, observed bool) string {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal([]byte(uiDismissalWire), &response); err != nil {
		t.Fatal(err)
	}
	response["id"], response["source_type"], response["source"], response["current_user_can_bypass"] = 7, "Repository", "owner/current", "never"
	if !observed {
		delete(response["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any), "dismissal_restriction")
	}
	body, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
func TestUIRulesetDismissalRawProjectionStillNotExportable(t *testing.T) {
	body := uiDismissalResponse(t, true)
	client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Fatal("projection wrote")
		}
		switch req.URL.Path {
		case "/repos/owner/current/rulesets":
			return labelsResponse(req, 200, `[{"id":7,"name":"Policy","source_type":"Repository","source":"owner/current"}]`), nil
		case "/repos/owner/current/rulesets/7":
			return labelsResponse(req, 200, body), nil
		default:
			t.Fatal(req.URL)
		}
		return nil, nil
	})
	raw, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := RulesetsObservation(raw)
	if err != nil || len(view.Rulesets) != 1 || view.Rulesets[0].UnverifiedReviewDismissalRestriction == nil || !*view.Rulesets[0].Rules.RequireAPullRequestBeforeMerging.RestrictWhoCanDismissPullRequestReviews || len(*view.Rulesets[0].Rules.RequireAPullRequestBeforeMerging.PeopleTeamsOrAppsThatCanDismissReviews) != 4 {
		t.Fatal(view, err)
	}
	exported, err := raw.ExportDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	if exported.Values()[0].HasWriteOnlyDismissal() {
		t.Fatal("prepared projection made raw restriction exportable")
	}
}
func TestUIRulesetDismissalExactWriteStillUnverifiedAndRepeated(t *testing.T) {
	for _, update := range []bool{false, true} {
		ui := uiRulesetBase()
		ui.Rules.RequireAPullRequestBeforeMerging = &uimodel.PullRequestRule{RestrictWhoCanDismissPullRequestReviews: rulesetPtr(true), PeopleTeamsOrAppsThatCanDismissReviews: rulesetPtr([]uimodel.ReviewDismissalActor{{User: rulesetPtr(int64(11))}, {Team: rulesetPtr(int64(12))}, {Role: rulesetPtr(uimodel.RepositoryAdmin)}, {App: rulesetPtr(int64(13))}})}
		method, path := "POST", "/repos/owner/current/rulesets"
		states := []settings.RulesetState{}
		if update {
			ui.ID = rulesetPtr(int64(7))
			method, path = "PUT", path+"/7"
			states = append(states, settings.RulesetState{ID: 7, Policy: apiRulesetBase()})
		}
		desired, err := RulesetsDeclaration(rulesetPtr([]uimodel.Ruleset{ui}))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(desired.WriteOnlyDismissalRulesets(), []string{"Policy"}) {
			t.Fatal("write-only application notice marker lost")
		}
		current, err := settings.NewRulesetObservation(states, 0)
		if err != nil {
			t.Fatal(err)
		}
		changes, err := settings.ReconcileRulesets(current, desired)
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		// Successful acceptance/readback deliberately omit this write-only object.
		response := uiDismissalResponse(t, false)
		calls := 0
		client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				var got, want any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(uiDismissalWire), &want); err != nil {
					t.Fatal(err)
				}
				if req.Method != method || req.URL.Path != path || !reflect.DeepEqual(got, want) {
					t.Fatal(req.Method, req.URL, got, want)
				}
			} else if calls != 2 || req.Method != "GET" || req.URL.Path != "/repos/owner/current/rulesets/7" {
				t.Fatal(req.Method, req.URL)
			}
			return labelsResponse(req, 200, response), nil
		})
		id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, changes[0])
		if err != nil || id != 7 || calls != 2 {
			t.Fatal(id, calls, err)
		}
		decoded, err := decodeRulesetPolicyFromTest([]byte(response))
		if err != nil {
			t.Fatal(err)
		}
		next, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: decoded}}, 0)
		if err != nil {
			t.Fatal(err)
		}
		entries := desired.Values()
		entries[0].ID = rulesetPtr(int64(7))
		again, err := settings.NewRulesetDeclarations(entries)
		if err != nil {
			t.Fatal(err)
		}
		repeated, err := settings.ReconcileRulesets(next, &again)
		if err != nil || len(repeated) != 1 {
			t.Fatal("acceptance falsely became verified no-op", fmt.Sprint(repeated), err)
		}
	}
}
