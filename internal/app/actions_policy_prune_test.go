package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsPolicyPrunePreflightAndFixedOrder(t *testing.T) {
	old := eventAppPolicy(7, "old")
	d := settings.Declaration{ActionsPolicies: eventPolicyDeclaration(t, eventAppPolicy(0, "new"))}
	for _, mode := range []string{"ok", "preflight", "drift", "delete-fails", "known-create-id"} {
		t.Run(mode, func(t *testing.T) {
			reads := 0
			ops := []string{}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
				ReadActionsPolicies: func(repository.Repository) (settings.ActionsPolicyObservation, error) {
					reads++
					if mode == "preflight" {
						return settings.ActionsPolicyObservation{}, errors.New("page denied")
					}
					value := old
					if mode == "drift" && reads > 1 {
						value.Enforcement = "active"
					}
					return eventPolicyObservation(t, value), nil
				},
				ActionsEventPolicy: func(_ repository.Repository, c settings.ActionsEventPolicyChange) (int64, error) {
					ops = append(ops, c.Operation())
					if c.Operation() == "create" {
						if mode == "known-create-id" {
							return 7, nil
						}
						return 9, nil
					}
					if c.ID() != 7 {
						t.Fatal("new creation selected for deletion")
					}
					if mode == "delete-fails" {
						return 0, errors.New("403")
					}
					return 7, nil
				},
			}, mustPrune(t, "actions-policies"))
			if mode == "ok" {
				if err != nil || !reflect.DeepEqual(ops, []string{"create", "delete"}) || len(result.CreatedIDs) != 1 {
					t.Fatal(result, err, ops)
				}
				return
			}
			if err == nil {
				t.Fatal("expected failure", result, ops)
			}
			if mode == "preflight" || mode == "drift" {
				if len(ops) > 0 {
					t.Fatal("writes before safe inventory", ops)
				}
			}
			if mode == "delete-fails" && (result.Outcome != ApplyPartialFailure || len(result.CreatedIDs) != 1 || !strings.Contains(err.Error(), "earlier successful writes")) {
				t.Fatal(result, err)
			}
			if mode == "known-create-id" && !reflect.DeepEqual(ops, []string{"create"}) {
				t.Fatal(ops)
			}
		})
	}
}
func TestActionsPolicyPruneUnsupportedBlocksUnrelatedWrites(t *testing.T) {
	yes := true
	d := settings.Declaration{ActionsPolicies: eventPolicyDeclaration(t)}
	d.Repository.HasIssues = &yes
	_, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
		ReadActionsPolicies: func(repository.Repository) (settings.ActionsPolicyObservation, error) {
			return settings.NewActionsPolicyObservation([]settings.ActionsEventPolicyState{{ID: 7, Name: "unknown", Unsupported: "future rule"}})
		},
		Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
			t.Fatal("unrelated write before candidate validation")
			return nil
		},
	}, mustPrune(t, "actions-policies"))
	if err == nil {
		t.Fatal("unsupported accepted")
	}
}
