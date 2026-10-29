package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type workflowReadFailure struct{ error }

func (*workflowReadFailure) ReadFailedBeforeWrite() {}

func TestWorkflowPreflightReadFailureHasNoUncertainWrite(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		permission := "read"
		d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &permission}}
		if earlier {
			value := false
			d.Actions.Enabled = &value
		}
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{Actions: func(repository.Repository, settings.ActionsPolicyDeclaration) error { return nil }, WorkflowPermissions: func(repository.Repository, settings.ActionsPolicyDeclaration) error {
			return fmt.Errorf("wrapped read: %w", &workflowReadFailure{errors.New("denied before PUT")})
		}})
		want := ApplyFailed
		if earlier {
			want = ApplyPartialFailure
		}
		if err == nil || result.Outcome != want || result.Operations[len(result.Operations)-1].Status != OperationFailedRead {
			t.Fatal(earlier, result, err)
		}
		if earlier && result.Operations[0].Status != OperationCompleted {
			t.Fatal("earlier progress lost", result)
		}
	}
}

func TestWorkflowPermissionFailureRetainsEarlierExecutionProgress(t *testing.T) {
	permission, enabled := "read", false
	d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &permission, Enabled: &enabled}}
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
