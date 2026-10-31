package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIConfigActionsControls(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      general:
        fork_pull_request_workflows:
          run_workflows_from_fork_pull_requests: true
          send_write_tokens_to_workflows_from_fork_pull_requests: false
          send_secrets_and_variables_to_workflows_from_fork_pull_requests: true
          require_approval_for_fork_pull_request_workflows: false
        approval_for_running_fork_pull_request_workflows_from_contributors: require_approval_for_first_time_contributors
        check_workflow_run_status_artifact_and_log_retention: 0
        actions_permissions:
          policy: allow_owner_and_select_non_owner_actions_and_reusable_workflows
          require_actions_to_be_pinned_to_a_full_length_commit_sha: true
          allow_actions_created_by_github: false
          allow_actions_by_marketplace_verified_creators: true
          allow_or_block_specified_actions_and_reusable_workflows: ['example/a@v1', '!example/b@*', 'example/a@v1']
        workflow_permissions:
          default_permissions: read_repository_contents_and_packages_permissions
          allow_github_actions_to_create_and_approve_pull_requests: false
`
	want := &uimodel.ActionsGeneral{
		ForkPullRequestWorkflows:                                   &uimodel.ForkPullRequestWorkflows{RunWorkflowsFromForkPullRequests: uiConfigPtr(true), SendWriteTokensToWorkflowsFromForkPullRequests: uiConfigPtr(false), SendSecretsAndVariablesToWorkflowsFromForkPullRequests: uiConfigPtr(true), RequireApprovalForForkPullRequestWorkflows: uiConfigPtr(false)},
		ApprovalForRunningForkPullRequestWorkflowsFromContributors: uiConfigPtr(uimodel.RequireApprovalForFirstTimeContributors), CheckWorkflowRunStatusArtifactAndLogRetention: uiConfigPtr(0),
		ActionsPermissions: uiConfigPtr(uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows), RequireActionsToBePinnedToAFullLengthCommitSHA: uiConfigPtr(true),
		AllowActionsCreatedByGitHub: uiConfigPtr(false), AllowActionsByMarketplaceVerifiedCreators: uiConfigPtr(true),
		AllowOrBlockSpecifiedActionsAndReusableWorkflows: uiConfigPtr([]string{"example/a@v1", "!example/b@*", "example/a@v1"}),
		WorkflowPermissions: uiConfigPtr(uimodel.ReadRepositoryContentsAndPackagesPermissions), AllowGitHubActionsToCreateAndApprovePullRequests: uiConfigPtr(false),
	}
	got, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	// The codec keeps syntax-level zero values; adapter validation owns ranges.
	if got.Settings == nil || got.Settings.Actions == nil || !reflect.DeepEqual(got.Settings.Actions.General, want) {
		t.Fatalf("Actions controls changed: %#v", got.Settings)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("Actions round trip changed intent: %s, %v", encoded, err)
	}
}

func TestUIConfigActionsRejectInvalidSyntax(t *testing.T) {
	for _, controls := range []string{
		"enabled: true", "allowed_actions: selected", "default_workflow_permissions: read",
		"fork_pull_request_workflows: null", "check_workflow_run_status_artifact_and_log_retention: '1'",
		"actions_permissions: {allow_actions_created_by_github: 'false'}", "actions_permissions: {allow_or_block_specified_actions_and_reusable_workflows: null}",
		"actions_permissions: {allow_or_block_specified_actions_and_reusable_workflows: [false]}",
		"actions_permissions: null", "workflow_permissions: null",
		"actions_permissions: {policy: false}", "workflow_permissions: {default_permissions: false}",
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {general: {" + controls + "}}}}\n"
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatalf("accepted invalid Actions syntax: %s", controls)
		}
	}
}
