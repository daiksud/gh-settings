package github

import (
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"net/http"
	"reflect"
	"testing"
)

func TestUIActionsRawDisabledSelectionAndExactTransition(t *testing.T) {
	step := 0
	client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
		step++
		want := []struct{ method, path, body, response string }{
			{"GET", "", "", `{"enabled":false,"allowed_actions":"selected","sha_pinning_required":false,"selected_actions_url":"https://untrusted.invalid/never"}`},
			{"GET", "/selected-actions", "", `{"github_owned_allowed":true,"verified_allowed":true,"patterns_allowed":["latent/action@v1"]}`},
			{"PUT", "", `{"enabled":false,"allowed_actions":"selected","sha_pinning_required":true}`, ""},
			{"PUT", "/selected-actions", `{"github_owned_allowed":false,"verified_allowed":true,"patterns_allowed":["example/a@v1","!example/a@v2","example/a@v1"]}`, ""},
			{"PUT", "", `{"enabled":true}`, ""},
			{"GET", "", "", `{"enabled":true,"allowed_actions":"selected","sha_pinning_required":true}`},
			{"GET", "/selected-actions", "", `{"github_owned_allowed":false,"verified_allowed":true,"patterns_allowed":["example/a@v1","!example/a@v2","example/a@v1"]}`},
		}
		if step > len(want) {
			t.Fatal("unexpected request", req.Method, req.URL)
		}
		w := want[step-1]
		if req.Method != w.method || req.URL.Path != "/repos/owner/current/actions/permissions"+w.path {
			t.Fatal(step, req.Method, req.URL)
		}
		if w.method == "PUT" {
			var got, expected any
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(w.body), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatal(step, got, expected)
			}
			return labelsResponse(req, 204, ""), nil
		}
		return labelsResponse(req, 200, w.response), nil
	})
	ui := uimodel.ActionsGeneral{ActionsPermissions: actionsUIPtr(uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows), RequireActionsToBePinnedToAFullLengthCommitSHA: actionsUIPtr(true), AllowActionsCreatedByGitHub: actionsUIPtr(false), AllowActionsByMarketplaceVerifiedCreators: actionsUIPtr(true), AllowOrBlockSpecifiedActionsAndReusableWorkflows: actionsUIPtr([]string{"example/a@v1", "!example/a@v2", "example/a@v1"})}
	d, err := ActionsDeclaration(ui)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ApplyActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, d); err != nil || step != 7 {
		t.Fatal(step, err)
	}
}

func TestUIActionsRawAcquisitionScopeAndFailures(t *testing.T) {
	for _, selected := range []bool{false, true} {
		calls := 0
		client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return labelsResponse(req, 200, `{"enabled":false,"allowed_actions":"selected","sha_pinning_required":false}`), nil
			}
			return labelsResponse(req, 200, `{"github_owned_allowed":false,"verified_allowed":true,"patterns_allowed":[]}`), nil
		})
		raw, err := client.ReadActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, selected)
		if err != nil {
			t.Fatal(err)
		}
		view, err := ActionsExecutionObservation(raw)
		if err != nil || *view.General.ActionsPermissions != "disable_actions" || view.General.AllowActionsCreatedByGitHub != nil || (view.Raw.Selected != nil) != selected {
			t.Fatal(view, err)
		}
		wantCalls := 1
		if selected {
			wantCalls = 2
		}
		if calls != wantCalls {
			t.Fatal(calls)
		}
	}
	for _, body := range []string{`{"enabled":false,"allowed_actions":"selected"}`, `{"enabled":false,"allowed_actions":42,"sha_pinning_required":false}`} {
		client := actionsClient(t, func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, body), nil })
		if _, err := client.ReadActionsPolicy(repository.Repository{Owner: "owner", Name: "current"}, false); err == nil {
			t.Fatal("incomplete/invalid observation accepted")
		}
	}
	partial, err := ActionsDeclaration(uimodel.ActionsGeneral{ForkPullRequestWorkflows: &uimodel.ForkPullRequestWorkflows{RunWorkflowsFromForkPullRequests: actionsUIPtr(true)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := settings.ComparePrivateFork(settings.NewPrivateForkObservation([4]bool{}), partial.PrivateFork); err == nil {
		t.Fatal("latent credentials defaulted on enable")
	}
}
