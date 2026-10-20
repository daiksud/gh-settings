package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func appRetInt(v int) *int { return &v }
func TestRetentionIndependentReadAndMatchingApply(t *testing.T) {
	d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{RetentionDays: appRetInt(30)}}
	o, _ := settings.NewRetentionObservation(90, 90)
	plan, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Reader{Retention: func(repository.Repository) (settings.RetentionObservation, error) { return o, nil }})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.Actions) != 1 {
		t.Fatal(plan, err)
	}
	days := 90
	writes := 0
	writer := Writer{ReadRetention: func(repository.Repository) (settings.RetentionObservation, error) {
		return settings.NewRetentionObservation(days, 90)
	}, Retention: func(_ repository.Repository, value int) (bool, error) { writes++; days = value; return true, nil }}
	for i := 0; i < 2; i++ {
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, writer)
		if err != nil || !result.Completed() {
			t.Fatal(result, err)
		}
	}
	if writes != 1 || days != 30 {
		t.Fatal(writes, days)
	}
}
func TestRetentionFailureDoesNotInventEarlierWrites(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		for _, failure := range []string{"read", "cap", "prewrite", "write"} {
			d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{RetentionDays: appRetInt(30)}}
			value := "write"
			if earlier {
				d.Actions.DefaultWorkflowPermissions = &value
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{WorkflowPermissions: func(repository.Repository, settings.ActionsPolicyDeclaration) error { return nil }, ReadRetention: func(repository.Repository) (settings.RetentionObservation, error) {
				if failure == "read" {
					return settings.RetentionObservation{}, errors.New("missing cap")
				}
				max := 90
				if failure == "cap" {
					max = 14
				}
				return settings.NewRetentionObservation(90, max)
			}, Retention: func(repository.Repository, int) (bool, error) {
				if failure == "prewrite" {
					return false, &workflowReadFailure{errors.New("fresh cap unavailable")}
				}
				return false, errors.New("write/readback unknown")
			}})
			want := ApplyFailed
			if earlier {
				want = ApplyPartialFailure
			}
			status := OperationFailedRead
			if failure == "write" {
				status = OperationFailedUnknown
			}
			if err == nil || result.Outcome != want || result.Operations[len(result.Operations)-1].Status != status {
				t.Fatal(earlier, failure, result, err)
			}
		}
	}
}
