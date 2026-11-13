package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsExecutionReconcileProgress(t *testing.T) {
	for _, stage := range []string{"noop", "read failure", "write failure", "verified write"} {
		for _, earlier := range []bool{false, true} {
			enabled, days := true, 30
			d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{Enabled: &enabled}}
			if earlier {
				d.Topics.Names = &[]string{"prior"}
			}
			writer := Writer{
				Topics: func(repository.Repository, settings.TopicsDeclaration) (bool, error) { return true, nil },
				Actions: func(repository.Repository, settings.ActionsPolicyDeclaration) (bool, error) {
					switch stage {
					case "read failure":
						return false, &workflowReadFailure{errors.New("unavailable")}
					case "write failure":
						return false, errors.New("uncertain")
					case "verified write":
						return true, nil
					default:
						return false, nil
					}
				},
				ReadRetention: func(repository.Repository) (settings.RetentionObservation, error) {
					return settings.RetentionObservation{}, errors.New("later read failed")
				},
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, writer)
			if strings.Contains(stage, "failure") {
				want := ApplyFailed
				if earlier {
					want = ApplyPartialFailure
				}
				status := OperationFailedUnknown
				if stage == "read failure" {
					status = OperationFailedRead
				}
				if err == nil || result.Outcome != want || result.Operations[len(result.Operations)-1].Status != status || strings.Contains(err.Error(), "earlier resource writes remain applied") != earlier {
					t.Fatal(stage, earlier, result, err)
				}
				continue
			}
			want := ApplyMatchingLabels
			status := OperationUnchanged
			if stage == "verified write" {
				status = OperationCompleted
			}
			if earlier || stage == "verified write" {
				want = ApplyCompleted
			}
			if err != nil || result.Outcome != want || result.Operations[len(result.Operations)-1].Status != status {
				t.Fatal(stage, earlier, result, err)
			}
			d.Actions.RetentionDays = &days
			result, err = Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, writer)
			want = ApplyFailed
			if earlier || stage == "verified write" {
				want = ApplyPartialFailure
			}
			if err == nil || result.Outcome != want {
				t.Fatal("incorrect prior progress", stage, earlier, result, err)
			}
		}
	}
}
