package github

import (
	"encoding/json"
	"fmt"
	"strings"

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
	var permission *string
	if json.Unmarshal(fields["default_workflow_permissions"], &permission) != nil || permission == nil {
		return settings.WorkflowPermissionsState{}, fmt.Errorf("GitHub returned missing/null or invalid default_workflow_permissions")
	}
	state.DefaultWorkflowPermissions = *permission
	value, err := actionsBool(fields, "can_approve_pull_request_reviews")
	if err != nil {
		return settings.WorkflowPermissionsState{}, err
	}
	state.CanApprovePullRequestReviews = value
	return NormalizeWorkflowPermissions(state), nil
}

// ApplyWorkflowPermissions sends only changed declared PUT keys and reports a
// verified write. An error after an attempted PUT can still have write effects.
func (c *ActionsPolicyClient) ApplyWorkflowPermissions(repo repository.Repository, desired settings.ActionsPolicyDeclaration) (bool, error) {
	if err := desired.Validate(); err != nil {
		return false, err
	}
	if desired.WorkflowPermissionsEmpty() {
		return false, nil
	}
	current, err := c.ReadWorkflowPermissions(repo)
	if err != nil {
		return false, &workflowPermissionsReadFailure{cause: err}
	}
	c.reportWorkflowNormalization(current, desired)
	_, changes, err := settings.CompareWorkflowPermissions(current, desired)
	if err != nil {
		return false, &workflowPermissionsReadFailure{cause: err}
	}
	if len(changes) == 0 {
		return false, nil
	}
	payload := map[string]any{}
	for _, change := range changes {
		_, after := change.Values()
		payload[strings.TrimPrefix(change.Field(), "actions.")] = after
	}
	if err := c.putActionsPolicy(actionsPolicyPath(repo)+"/workflow", payload); err != nil {
		return false, fmt.Errorf("write default workflow permissions (this and earlier writes may already have applied): %w", err)
	}
	actual, err := c.ReadWorkflowPermissions(repo)
	if err != nil {
		return false, fmt.Errorf("verify default workflow permissions (writes may already have applied): %w", err)
	}
	c.reportWorkflowNormalization(actual, desired)
	if !workflowPermissionsMatch(current, actual, desired) {
		return false, fmt.Errorf("GitHub did not retain requested/default workflow permission companions (writes may already have applied)")
	}
	return true, nil
}

func (c *ActionsPolicyClient) reportWorkflowNormalization(state settings.WorkflowPermissionsState, desired settings.ActionsPolicyDeclaration) {
	if desired.DefaultWorkflowPermissions != nil && state.Normalization.Path != "" && c.NormalizationWarning != nil {
		c.NormalizationWarning(state.Normalization)
	}
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
