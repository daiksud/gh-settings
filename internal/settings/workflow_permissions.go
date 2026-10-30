package settings

import (
	"fmt"
)

type WorkflowPermissionsState struct {
	DefaultWorkflowPermissions   string
	CanApprovePullRequestReviews bool
	Normalization                ObservationNormalization
}

func ValidDefaultWorkflowPermissions(value string) bool { return value == "read" || value == "write" }

func (d ActionsPolicyDeclaration) WorkflowPermissionsEmpty() bool {
	return d.DefaultWorkflowPermissions == nil && d.CanApprovePullRequestReviews == nil
}
func (d ActionsPolicyDeclaration) Empty() bool {
	return d.ExecutionEmpty() && d.WorkflowPermissionsEmpty() && d.RetentionEmpty() && d.ForkApprovalPolicy == nil && d.PrivateFork.Empty()
}

// WithWorkflowPermissions adds an independently acquired complete policy.
func (d ActionsPolicyDeclaration) WithWorkflowPermissions(state WorkflowPermissionsState) (ActionsPolicyDeclaration, error) {
	if !ValidDefaultWorkflowPermissions(state.DefaultWorkflowPermissions) {
		return ActionsPolicyDeclaration{}, fmt.Errorf("unsupported observed default_workflow_permissions")
	}
	d.DefaultWorkflowPermissions = &state.DefaultWorkflowPermissions
	d.CanApprovePullRequestReviews = &state.CanApprovePullRequestReviews
	return d, nil
}

func CompareWorkflowPermissions(current WorkflowPermissionsState, desired ActionsPolicyDeclaration) (int, []ActionsPolicyChange, error) {
	if !ValidDefaultWorkflowPermissions(current.DefaultWorkflowPermissions) {
		return 0, nil, fmt.Errorf("unsupported observed default_workflow_permissions")
	}
	if err := desired.Validate(); err != nil {
		return 0, nil, err
	}
	managed := 0
	changes := []ActionsPolicyChange{}
	if value := desired.DefaultWorkflowPermissions; value != nil {
		managed++
		if current.Normalization != (ObservationNormalization{}) {
			change, err := NewActionsPolicyChangeWithNormalization("actions.default_workflow_permissions", current.DefaultWorkflowPermissions, *value, current.Normalization)
			if err != nil {
				return 0, nil, err
			}
			changes = append(changes, change)
		} else if *value != current.DefaultWorkflowPermissions {
			changes = append(changes, ActionsPolicyChange{field: "actions.default_workflow_permissions", before: current.DefaultWorkflowPermissions, after: *value})
		}
	}
	if value := desired.CanApprovePullRequestReviews; value != nil {
		managed++
		if *value != current.CanApprovePullRequestReviews {
			changes = append(changes, ActionsPolicyChange{field: "actions.can_approve_pull_request_reviews", before: current.CanApprovePullRequestReviews, after: *value})
		}
	}
	return managed, changes, nil
}
