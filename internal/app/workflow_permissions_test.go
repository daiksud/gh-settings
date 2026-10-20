package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func TestWorkflowPermissionFailureRetainsEarlierExecutionProgress(t *testing.T) {
	d, err := workspace.DecodeConfig([]byte("schemaVersion: 1\nsettings: {actions: {enabled: false, default_workflow_permissions: read}}"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{Actions: func(repository.Repository, settings.ActionsPolicyDeclaration) error { return nil }, WorkflowPermissions: func(repository.Repository, settings.ActionsPolicyDeclaration) error {
		return errors.New("organization conflict")
	}})
	if err == nil || result.Outcome != ApplyPartialFailure || len(result.Operations) != 2 || result.Operations[0].Kind != ActionsPolicyUpdate || result.Operations[0].Status != OperationCompleted || result.Operations[1].Kind != WorkflowPermissionsUpdate || result.Operations[1].Status != OperationFailedUnknown {
		t.Fatal(result, err)
	}
	plan, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Reader{Actions: func(repository.Repository, bool) (settings.ActionsPolicyState, error) {
		return settings.ActionsPolicyState{Enabled: true, AllowedActions: "all"}, nil
	}, WorkflowPermissions: func(repository.Repository) (settings.WorkflowPermissionsState, error) {
		return settings.WorkflowPermissionsState{}, errors.New("unavailable")
	}})
	var failure *PlanError
	if !errors.As(err, &failure) || failure.Stage != "read-failure" || plan.Comparison.Managed != 0 || len(plan.Comparison.Actions) != 0 {
		t.Fatal("partial comparison escaped", plan, err)
	}
}
