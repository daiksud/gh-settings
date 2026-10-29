package github

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// The use case recognizes this stage without depending on the API adapter.
type workflowPermissionsReadFailure struct{ cause error }

func (failure *workflowPermissionsReadFailure) Error() string  { return failure.cause.Error() }
func (failure *workflowPermissionsReadFailure) Unwrap() error  { return failure.cause }
func (*workflowPermissionsReadFailure) ReadFailedBeforeWrite() {}

func (c *ActionsPolicyClient) ReadWorkflowPermissions(repo repository.Repository) (settings.WorkflowPermissionsState, error) {
	var fields map[string]json.RawMessage
	if err := c.rest.Get(actionsPolicyPath(repo)+"/workflow", &fields); err != nil {
		return settings.WorkflowPermissionsState{}, fmt.Errorf("read default workflow permissions: %w", err)
	}
	var state settings.WorkflowPermissionsState
	if json.Unmarshal(fields["default_workflow_permissions"], &state.DefaultWorkflowPermissions) != nil || !settings.ValidDefaultWorkflowPermissions(state.DefaultWorkflowPermissions) {
		return settings.WorkflowPermissionsState{}, fmt.Errorf("GitHub returned missing/null or unsupported default_workflow_permissions")
	}
	value, err := actionsBool(fields, "can_approve_pull_request_reviews")
	if err != nil {
		return settings.WorkflowPermissionsState{}, err
	}
	state.CanApprovePullRequestReviews = value
	return state, nil
}

// ApplyWorkflowPermissions sends only declared optional PUT keys. Fresh complete
// reads verify desired and preserved companions without promising atomicity.
func (c *ActionsPolicyClient) ApplyWorkflowPermissions(repo repository.Repository, desired settings.ActionsPolicyDeclaration) error {
	if err := desired.Validate(); err != nil {
		return err
	}
	if desired.WorkflowPermissionsEmpty() {
		return nil
	}
	current, err := c.ReadWorkflowPermissions(repo)
	if err != nil {
		return &workflowPermissionsReadFailure{cause: err}
	}
	payload := map[string]any{}
	if value := desired.DefaultWorkflowPermissions; value != nil {
		payload["default_workflow_permissions"] = *value
	}
	if value := desired.CanApprovePullRequestReviews; value != nil {
		payload["can_approve_pull_request_reviews"] = *value
	}
	if err := c.putActionsPolicy(actionsPolicyPath(repo)+"/workflow", payload); err != nil {
		return fmt.Errorf("write default workflow permissions (this and earlier writes may already have applied): %w", err)
	}
	actual, err := c.ReadWorkflowPermissions(repo)
	if err != nil {
		return fmt.Errorf("verify default workflow permissions (writes may already have applied): %w", err)
	}
	if !workflowPermissionsMatch(current, actual, desired) {
		return fmt.Errorf("GitHub did not retain requested/default workflow permission companions (writes may already have applied)")
	}
	return nil
}

func workflowPermissionsMatch(before, actual settings.WorkflowPermissionsState, desired settings.ActionsPolicyDeclaration) bool {
	raw := func(state settings.WorkflowPermissionsState) string {
		if state.Normalization.Path != "" {
			return state.Normalization.RawValue
		}
		return state.DefaultWorkflowPermissions
	}
	permission, approval := raw(before), before.CanApprovePullRequestReviews
	if desired.DefaultWorkflowPermissions != nil {
		permission = *desired.DefaultWorkflowPermissions
	}
	if desired.CanApprovePullRequestReviews != nil {
		approval = *desired.CanApprovePullRequestReviews
	}
	return raw(actual) == permission && actual.CanApprovePullRequestReviews == approval
}
