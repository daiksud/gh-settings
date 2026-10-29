package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIActionsPublicForkChoices(t *testing.T) {
	for _, test := range []struct{ ui, api string }{
		{"require_approval_for_first_time_contributors_who_are_new_to_github", "first_time_contributors_new_to_github"},
		{"require_approval_for_first_time_contributors", "first_time_contributors"},
		{"require_approval_for_all_external_contributors", "all_external_contributors"},
	} {
		got, err := ActionsDeclaration(uimodel.ActionsGeneral{ApprovalForRunningForkPullRequestWorkflowsFromContributors: actionsUIPtr(uimodel.ForkApprovalChoice(test.ui))})
		if err != nil || !reflect.DeepEqual(got, settings.ActionsPolicyDeclaration{ForkApprovalPolicy: &test.api}) {
			t.Fatal(got, err)
		}
		o, err := settings.NewForkApprovalObservation(test.api)
		if err != nil {
			t.Fatal(err)
		}
		view, err := WithActionsForkApprovalObservation(uimodel.ActionsGeneral{AllowGitHubActionsToCreateAndApprovePullRequests: actionsUIPtr(false)}, o)
		if err != nil || *view.ApprovalForRunningForkPullRequestWorkflowsFromContributors != uimodel.ForkApprovalChoice(test.ui) || view.AllowGitHubActionsToCreateAndApprovePullRequests == nil {
			t.Fatal(view, err)
		}
	}
	if _, err := ActionsDeclaration(uimodel.ActionsGeneral{ApprovalForRunningForkPullRequestWorkflowsFromContributors: actionsUIPtr(uimodel.ForkApprovalChoice("first_time_contributors"))}); err == nil {
		t.Fatal("API token accepted")
	}
	if _, err := WithActionsForkApprovalObservation(uimodel.ActionsGeneral{}, settings.ForkApprovalObservation{}); err == nil {
		t.Fatal("missing observation defaulted")
	}
}

func TestUIActionsPublicForkExactWriteAndRead(t *testing.T) {
	written := false
	calls := 0
	client := &ForkWorkflowsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path == "/repos/owner/current" && req.Method == "GET" {
			return labelsResponse(req, 200, `{"visibility":"public"}`), nil
		}
		if req.URL.Path != "/repos/owner/current/actions/permissions/fork-pr-contributor-approval" {
			t.Fatal(req.URL)
		}
		if req.Method == "PUT" {
			var got map[string]any
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, map[string]any{"approval_policy": "all_external_contributors"}) {
				t.Fatal(got)
			}
			written = true
			return labelsResponse(req, 204, ""), nil
		}
		if req.Method != "GET" {
			t.Fatal(req.Method)
		}
		if written {
			return labelsResponse(req, 200, `{"approval_policy":"all_external_contributors"}`), nil
		}
		return labelsResponse(req, 200, `{"approval_policy":"first_time_contributors"}`), nil
	}).rest}
	d, err := ActionsDeclaration(uimodel.ActionsGeneral{ApprovalForRunningForkPullRequestWorkflowsFromContributors: actionsUIPtr(uimodel.ForkApprovalChoice("require_approval_for_all_external_contributors"))})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := client.UpdateApproval(repository.Repository{Owner: "owner", Name: "current"}, *d.ForkApprovalPolicy)
	if err != nil || !changed || calls != 5 {
		t.Fatal(changed, calls, err)
	}
}
