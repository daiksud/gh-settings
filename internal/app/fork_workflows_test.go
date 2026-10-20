package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func forkAppBool(v bool) *bool { return &v }
func TestForkIndependentPlanApplyAndPartialProgress(t *testing.T) {
	for _, private := range []bool{false, true} {
		policy := "all_external_contributors"
		d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{ForkApprovalPolicy: &policy}}
		if private {
			d.Actions = settings.ActionsPolicyDeclaration{PrivateFork: settings.PrivateForkDeclaration{Run: forkAppBool(true), Secrets: forkAppBool(false)}}
		}
		values := [4]bool{true, false, true, false}
		approval := "first_time_contributors"
		writes := 0
		read := Reader{ForkApproval: func(repository.Repository) (settings.ForkApprovalObservation, error) {
			return settings.NewForkApprovalObservation(approval)
		}, PrivateFork: func(repository.Repository) (settings.PrivateForkObservation, error) {
			return settings.NewPrivateForkObservation(values), nil
		}}
		plan, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, read)
		if err != nil || len(plan.Comparison.Actions) != 1 {
			t.Fatal(plan, err)
		}
		writer := Writer{ReadForkApproval: read.ForkApproval, ReadPrivateFork: read.PrivateFork, ForkApproval: func(_ repository.Repository, v string) (bool, error) { writes++; approval = v; return true, nil }, PrivateFork: func(_ repository.Repository, v settings.PrivateForkDeclaration) (bool, error) {
			writes++
			values[2] = *v.Secrets
			return true, nil
		}}
		for i := 0; i < 2; i++ {
			r, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, writer)
			if err != nil || !r.Completed() {
				t.Fatal(r, err)
			}
		}
		if writes != 1 {
			t.Fatal(writes)
		}
	}
	for _, earlier := range []bool{false, true} {
		for _, mode := range []string{"read", "prewrite", "write", "skip"} {
			p := "all_external_contributors"
			d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{ForkApprovalPolicy: &p, PrivateFork: settings.PrivateForkDeclaration{Run: forkAppBool(false)}}}
			o, _ := settings.NewForkApprovalObservation("first_time_contributors")
			writer := Writer{ReadForkApproval: func(repository.Repository) (settings.ForkApprovalObservation, error) { return o, nil }, ForkApproval: func(repository.Repository, string) (bool, error) { return earlier, nil }, ReadPrivateFork: func(repository.Repository) (settings.PrivateForkObservation, error) {
				if mode == "read" {
					return settings.PrivateForkObservation{}, errors.New("inapplicable private policy")
				}
				return settings.NewPrivateForkObservation([4]bool{true, false, false, true}), nil
			}, PrivateFork: func(repository.Repository, settings.PrivateForkDeclaration) (bool, error) {
				if mode == "prewrite" {
					return false, &workflowReadFailure{errors.New("fresh read")}
				}
				if mode == "skip" {
					return false, nil
				}
				return false, errors.New("PUT effects unknown")
			}}
			r, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, writer)
			if mode == "skip" {
				if err != nil || !r.Completed() {
					t.Fatal(r, err)
				}
				continue
			}
			want := ApplyFailed
			if earlier {
				want = ApplyPartialFailure
			}
			status := OperationFailedRead
			if mode == "write" {
				status = OperationFailedUnknown
			}
			if err == nil || r.Outcome != want || r.Operations[len(r.Operations)-1].Status != status {
				t.Fatal(earlier, mode, r, err)
			}
			// A later failure discards an earlier comparison, unlike direct apply progress.
			read := Reader{ForkApproval: writer.ReadForkApproval, PrivateFork: func(repository.Repository) (settings.PrivateForkObservation, error) {
				return settings.PrivateForkObservation{}, errors.New("unavailable")
			}}
			rplan, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, read)
			if err == nil || rplan.Comparison.Managed != 0 || len(rplan.Comparison.Actions) != 0 {
				t.Fatal(rplan, err)
			}
		}
	}
}
