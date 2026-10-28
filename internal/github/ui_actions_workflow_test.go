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

func TestUIActionsWorkflowChoices(t *testing.T) {
	for _, test := range []struct{ ui, api string }{{"read_and_write_permissions", "write"}, {"read_repository_contents_and_packages_permissions", "read"}} {
		got, err := ActionsDeclaration(uimodel.ActionsGeneral{WorkflowPermissions: actionsUIPtr(uimodel.WorkflowPermissionChoice(test.ui))})
		if err != nil || !reflect.DeepEqual(got, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &test.api}) {
			t.Fatal(got, err)
		}
		view, err := WithActionsWorkflowObservation(uimodel.ActionsGeneral{RequireActionsToBePinnedToAFullLengthCommitSHA: actionsUIPtr(false)}, settings.WorkflowPermissionsState{DefaultWorkflowPermissions: test.api, CanApprovePullRequestReviews: false})
		if err != nil || *view.WorkflowPermissions != uimodel.WorkflowPermissionChoice(test.ui) || view.AllowGitHubActionsToCreateAndApprovePullRequests == nil || *view.AllowGitHubActionsToCreateAndApprovePullRequests || view.RequireActionsToBePinnedToAFullLengthCommitSHA == nil || *view.RequireActionsToBePinnedToAFullLengthCommitSHA {
			t.Fatal(view, err)
		}
	}
	for _, value := range []bool{false, true} {
		got, err := ActionsDeclaration(uimodel.ActionsGeneral{AllowGitHubActionsToCreateAndApprovePullRequests: &value})
		if err != nil || !reflect.DeepEqual(got, settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: &value}) {
			t.Fatal(got, err)
		}
	}
}

func TestUIActionsWorkflowExactDeclaredOnlyWrite(t *testing.T) {
	calls := 0
	client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path != "/repos/owner/current/actions/permissions/workflow" {
			t.Fatal(req.URL)
		}
		if req.Method == "PUT" {
			var got map[string]any
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, map[string]any{"can_approve_pull_request_reviews": false}) {
				t.Fatal(got)
			}
			return labelsResponse(req, 204, ""), nil
		}
		if req.Method != "GET" {
			t.Fatal(req.Method)
		}
		if calls == 1 {
			return labelsResponse(req, 200, `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":true}`), nil
		}
		return labelsResponse(req, 200, `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`), nil
	})
	desired, err := ActionsDeclaration(uimodel.ActionsGeneral{AllowGitHubActionsToCreateAndApprovePullRequests: actionsUIPtr(false)})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ApplyWorkflowPermissions(repository.Repository{Owner: "owner", Name: "current"}, desired); err != nil || calls != 3 {
		t.Fatal(calls, err)
	}
}

func TestUIActionsWorkflowInvalidChoices(t *testing.T) {
	if _, err := ActionsDeclaration(uimodel.ActionsGeneral{WorkflowPermissions: actionsUIPtr(uimodel.WorkflowPermissionChoice("write"))}); err == nil {
		t.Fatal("API token accepted")
	}
	if _, err := WithActionsWorkflowObservation(uimodel.ActionsGeneral{}, settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "UNKNOWN"}); err == nil {
		t.Fatal("unknown workflow observation accepted")
	}
}
