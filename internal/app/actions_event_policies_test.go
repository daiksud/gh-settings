package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func eventPolicyDeclaration(t *testing.T, values ...settings.ActionsEventPolicy) *settings.ActionsPolicyDeclarations {
	t.Helper()
	d, err := settings.NewActionsPolicyDeclarations(values)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}
func eventPolicyObservation(t *testing.T, values ...settings.ActionsEventPolicy) settings.ActionsPolicyObservation {
	t.Helper()
	states := []settings.ActionsEventPolicyState{}
	for _, v := range values {
		v := v
		states = append(states, settings.ActionsEventPolicyState{ID: v.ID, Name: v.Name, Policy: &v})
	}
	o, err := settings.NewActionsPolicyObservation(states)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func eventAppPolicy(id int64, name string) settings.ActionsEventPolicy {
	return settings.ActionsEventPolicy{ID: id, Name: name, Enforcement: "disabled", Events: []string{"push"}}
}
func TestActionsEventPoliciesPlanAndEmptyNoop(t *testing.T) {
	empty := eventPolicyDeclaration(t)
	for _, d := range []*settings.ActionsPolicyDeclarations{nil, empty} {
		load := func() (settings.Declaration, error) { return settings.Declaration{ActionsPolicies: d}, nil }
		if r, e := Plan(load, nil, Reader{}); e != nil || r.Comparison.Managed != 0 {
			t.Fatal(r, e)
		}
		if r, e := Apply(load, nil, Writer{}); e != nil || r.Outcome != ApplyUnmanaged {
			t.Fatal(r, e)
		}
	}
	before := eventAppPolicy(7, "policy")
	after := before
	after.Enforcement = "active"
	load := func() (settings.Declaration, error) {
		return settings.Declaration{ActionsPolicies: eventPolicyDeclaration(t, after)}, nil
	}
	r, e := Plan(load, checkoutRepository, Reader{ActionsPolicies: func(repository.Repository) (settings.ActionsPolicyObservation, error) {
		return eventPolicyObservation(t, before), nil
	}})
	if e != nil || r.Comparison.Managed != 1 || len(r.Comparison.ActionsPolicies) != 1 || !r.Comparison.HasChanges() {
		t.Fatal(r, e)
	}
	_, e = Plan(load, checkoutRepository, Reader{ActionsPolicies: func(repository.Repository) (settings.ActionsPolicyObservation, error) {
		return settings.ActionsPolicyObservation{}, nil
	}})
	var pe *PlanError
	if !errors.As(e, &pe) || pe.Path != "repository.settings.actions.policies" {
		t.Fatal(e)
	}
}
func TestActionsEventPoliciesApplyPreflightAndOrdering(t *testing.T) {
	for _, fail := range []string{"", "read", "preflight", "write"} {
		t.Run(fail, func(t *testing.T) {
			before := eventAppPolicy(7, "policy")
			after := before
			after.Enforcement = "active"
			desired := settings.Declaration{ActionsPolicies: eventPolicyDeclaration(t, after)}
			yes := true
			desired.Repository.HasIssues = &yes
			var calls []string
			r, e := Apply(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
				calls = append(calls, "repository")
				return nil
			}, ReadActionsPolicies: func(repository.Repository) (settings.ActionsPolicyObservation, error) {
				calls = append(calls, "read")
				if fail == "read" {
					return settings.ActionsPolicyObservation{}, errors.New("denied")
				}
				if fail == "preflight" {
					return eventPolicyObservation(t), nil
				}
				return eventPolicyObservation(t, before), nil
			}, ActionsEventPolicy: func(_ repository.Repository, c settings.ActionsEventPolicyChange) (int64, error) {
				calls = append(calls, "write")
				if c.ID() != 7 || c.After().Enforcement != "active" {
					t.Fatal(c)
				}
				if fail == "write" {
					return 0, errors.New("uncertain")
				}
				return 7, nil
			}})
			want := []string{"repository", "read"}
			if fail == "" || fail == "write" {
				want = append(want, "write")
			}
			if !reflect.DeepEqual(calls, want) || (e == nil) != (fail == "") {
				t.Fatal(calls, r, e)
			}
			if fail != "" && r.Outcome != ApplyPartialFailure {
				t.Fatal(r, e)
			}
		})
	}
	// A later invalid selected ID prevents all writes in this collection.
	d := eventPolicyDeclaration(t, eventAppPolicy(0, "a-new"), eventAppPolicy(99, "z-missing"))
	writes := 0
	r, e := Apply(func() (settings.Declaration, error) { return settings.Declaration{ActionsPolicies: d}, nil }, checkoutRepository, Writer{ReadActionsPolicies: func(repository.Repository) (settings.ActionsPolicyObservation, error) {
		return eventPolicyObservation(t), nil
	}, ActionsEventPolicy: func(repository.Repository, settings.ActionsEventPolicyChange) (int64, error) {
		writes++
		return 17, nil
	}})
	if e == nil || writes != 0 || r.Outcome != ApplyFailed || r.Operations[0].Status != OperationFailedRead {
		t.Fatal(r, e, writes)
	}
}
func TestActionsEventPoliciesCreatedIDRecovery(t *testing.T) {
	for _, mode := range []string{"success", "writeback", "readback", "known-id", "missing-id", "later-failure"} {
		t.Run(mode, func(t *testing.T) {
			desired := eventPolicyDeclaration(t, eventAppPolicy(0, "a-new"))
			if mode == "later-failure" {
				desired = eventPolicyDeclaration(t, eventAppPolicy(0, "a-new"), eventAppPolicy(0, "z-new"))
			}
			var calls []string
			writes := 0
			r, e := Apply(func() (settings.Declaration, error) { return settings.Declaration{ActionsPolicies: desired}, nil }, checkoutRepository, Writer{ReadActionsPolicies: func(repository.Repository) (settings.ActionsPolicyObservation, error) {
				return eventPolicyObservation(t, eventAppPolicy(9, "unlisted")), nil
			}, ActionsEventPolicy: func(_ repository.Repository, c settings.ActionsEventPolicyChange) (int64, error) {
				writes++
				calls = append(calls, "write:"+c.Name())
				if mode == "known-id" {
					return 9, nil
				}
				if mode == "missing-id" {
					return 0, nil
				}
				if mode == "readback" {
					return 17, errors.New("readback failed")
				}
				if writes == 2 {
					return 0, errors.New("uncertain")
				}
				return 17, nil
			}, CreatedID: func(collection, name string, id int64) error {
				calls = append(calls, "record:"+name)
				if collection != "actions_policies" || id != 17 {
					t.Fatal(collection, name, id)
				}
				if mode == "writeback" {
					return errors.New("edited")
				}
				return nil
			}})
			if (e == nil) != (mode == "success") {
				t.Fatal(r, e)
			}
			expectRecovery := mode != "known-id" && mode != "missing-id"
			if (len(r.CreatedIDs) == 1) != expectRecovery {
				t.Fatal(r, e)
			}
			if mode == "writeback" || mode == "later-failure" {
				if r.Outcome != ApplyPartialFailure {
					t.Fatal(r, e)
				}
			}
			if mode == "readback" && (len(calls) != 1 || !strings.Contains(e.Error(), "readback")) {
				t.Fatal(calls, e)
			}
			if mode == "later-failure" && !reflect.DeepEqual(calls, []string{"write:a-new", "record:a-new", "write:z-new"}) {
				t.Fatal(calls)
			}
		})
	}
}
func TestActionsEventPoliciesExportSelection(t *testing.T) {
	for _, mode := range []string{"selected"} {
		t.Run(mode, func(t *testing.T) {
			reads, saved := 0, false
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reader.ActionsPolicies = func(repository.Repository) (settings.ActionsPolicyObservation, error) {
				reads++
				if mode == "read-error" {
					return settings.ActionsPolicyObservation{}, errors.New("denied")
				}
				if mode == "unsupported" {
					return settings.NewActionsPolicyObservation([]settings.ActionsEventPolicyState{{ID: 7, Name: "actor", Unsupported: "actors"}})
				}
				return eventPolicyObservation(t, eventAppPolicy(7, "policy")), nil
			}
			e := Export(ExportCheckout{Repository: checkoutRepository, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saved = true
					if (d.ActionsPolicies != nil) != (mode != "omitted") {
						t.Fatal(d.ActionsPolicies)
					}
					return nil
				}, nil
			}}, reader)
			fail := mode == "read-error" || mode == "unsupported"
			if (e != nil) != fail || saved == fail || (mode == "omitted" && reads != 0) || (mode != "omitted" && reads != 1) {
				t.Fatal(e, saved, reads)
			}
		})
	}
}

type eventPolicyPreflightError struct{}

func (eventPolicyPreflightError) Error() string          { return "fresh read failed" }
func (eventPolicyPreflightError) ReadFailedBeforeWrite() {}
func TestActionsEventPoliciesFreshFailureStatusAndArchiveGuard(t *testing.T) {
	before := eventAppPolicy(7, "policy")
	after := before
	after.Enforcement = "active"
	d := settings.Declaration{ActionsPolicies: eventPolicyDeclaration(t, after)}
	result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadActionsPolicies: func(repository.Repository) (settings.ActionsPolicyObservation, error) {
		return eventPolicyObservation(t, before), nil
	}, ActionsEventPolicy: func(repository.Repository, settings.ActionsEventPolicyChange) (int64, error) {
		return 0, eventPolicyPreflightError{}
	}})
	if err == nil || result.Outcome != ApplyFailed || result.Operations[1].Status != OperationFailedRead {
		t.Fatal(result, err)
	}
	yes := true
	d.Repository.Archived = &yes
	result, err = Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{ReadArchive: func(repo repository.Repository) (settings.ArchiveState, error) {
		return settings.ArchiveState{ID: 1, FullName: repo.Owner + "/" + repo.Name, Archived: true}, nil
	}})
	if err == nil || !strings.Contains(err.Error(), "cannot update other managed resources") || len(result.Operations) != 1 {
		t.Fatal(result, err)
	}
}
