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

func TestUIActionsPrivateForkControls(t *testing.T) {
	for i := 0; i < 4; i++ {
		for _, value := range []bool{false, true} {
			ui := uimodel.ForkPullRequestWorkflows{}
			fields := []**bool{&ui.RunWorkflowsFromForkPullRequests, &ui.SendWriteTokensToWorkflowsFromForkPullRequests, &ui.SendSecretsAndVariablesToWorkflowsFromForkPullRequests, &ui.RequireApprovalForForkPullRequestWorkflows}
			*fields[i] = actionsUIPtr(value)
			got, err := ActionsDeclaration(uimodel.ActionsGeneral{ForkPullRequestWorkflows: &ui})
			var want settings.PrivateForkDeclaration
			target := []**bool{&want.Run, &want.WriteTokens, &want.Secrets, &want.Approval}
			*target[i] = &value
			if err != nil || !reflect.DeepEqual(got.PrivateFork, want) {
				t.Fatal(i, value, got, err)
			}
		}
	}
	if _, err := ActionsDeclaration(uimodel.ActionsGeneral{ForkPullRequestWorkflows: &uimodel.ForkPullRequestWorkflows{RunWorkflowsFromForkPullRequests: actionsUIPtr(false), SendWriteTokensToWorkflowsFromForkPullRequests: actionsUIPtr(false)}}); err == nil {
		t.Fatal("hidden desired control accepted")
	}
	for _, run := range []bool{false, true} {
		raw := [4]bool{run, true, false, true}
		view, err := WithActionsPrivateForkObservation(uimodel.ActionsGeneral{AllowGitHubActionsToCreateAndApprovePullRequests: actionsUIPtr(false)}, settings.NewPrivateForkObservation(raw))
		if err != nil || view.Raw != raw || view.General.AllowGitHubActionsToCreateAndApprovePullRequests == nil {
			t.Fatal(view, err)
		}
		got := view.General.ForkPullRequestWorkflows
		if got == nil || *got.RunWorkflowsFromForkPullRequests != run {
			t.Fatal(got)
		}
		if !run {
			if got.SendWriteTokensToWorkflowsFromForkPullRequests != nil || got.SendSecretsAndVariablesToWorkflowsFromForkPullRequests != nil || got.RequireApprovalForForkPullRequestWorkflows != nil {
				t.Fatal("latent controls exposed", got)
			}
		} else if !*got.SendWriteTokensToWorkflowsFromForkPullRequests || *got.SendSecretsAndVariablesToWorkflowsFromForkPullRequests || !*got.RequireApprovalForForkPullRequestWorkflows {
			t.Fatal(got)
		}
	}
	if _, err := WithActionsPrivateForkObservation(uimodel.ActionsGeneral{}, settings.PrivateForkObservation{}); err == nil {
		t.Fatal("unacquired state defaulted")
	}
}

func TestUIActionsPrivateForkExactWriteAndRead(t *testing.T) {
	written := false
	calls := 0
	client := &ForkWorkflowsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path == "/repos/owner/current" && req.Method == "GET" {
			return labelsResponse(req, 200, `{"visibility":"private"}`), nil
		}
		if req.URL.Path != "/repos/owner/current/actions/permissions/fork-pr-workflows-private-repos" {
			t.Fatal(req.URL)
		}
		if req.Method == "PUT" {
			var got map[string]bool
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			want := map[string]bool{"run_workflows_from_fork_pull_requests": true, "send_write_tokens_to_workflows": false, "send_secrets_and_variables": false, "require_approval_for_fork_pr_workflows": true}
			if !reflect.DeepEqual(got, want) {
				t.Fatal(got)
			}
			written = true
			return labelsResponse(req, 204, ""), nil
		}
		if req.Method != "GET" {
			t.Fatal(req.Method)
		}
		if written {
			return labelsResponse(req, 200, `{"run_workflows_from_fork_pull_requests":true,"send_write_tokens_to_workflows":false,"send_secrets_and_variables":false,"require_approval_for_fork_pr_workflows":true}`), nil
		}
		return labelsResponse(req, 200, `{"run_workflows_from_fork_pull_requests":false,"send_write_tokens_to_workflows":true,"send_secrets_and_variables":true,"require_approval_for_fork_pr_workflows":false}`), nil
	}).rest}
	d, err := ActionsDeclaration(uimodel.ActionsGeneral{ForkPullRequestWorkflows: &uimodel.ForkPullRequestWorkflows{RunWorkflowsFromForkPullRequests: actionsUIPtr(true), SendWriteTokensToWorkflowsFromForkPullRequests: actionsUIPtr(false), SendSecretsAndVariablesToWorkflowsFromForkPullRequests: actionsUIPtr(false), RequireApprovalForForkPullRequestWorkflows: actionsUIPtr(true)}})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := client.UpdatePrivate(repository.Repository{Owner: "owner", Name: "current"}, d.PrivateFork)
	if err != nil || !changed || calls != 5 {
		t.Fatal(changed, calls, err)
	}
}
