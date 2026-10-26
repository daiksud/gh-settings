package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRulesetEmptySelectionNeedsNoTargetOrAPI(t *testing.T) {
	values, err := settings.NewRulesetDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	load := func() (settings.Declaration, error) { return settings.Declaration{Rulesets: &values}, nil }
	resolve := func() (repository.Repository, error) {
		t.Fatal("empty selection resolved remote target")
		return repository.Repository{}, nil
	}
	plan, err := Plan(load, resolve, Reader{})
	if err != nil || plan.Comparison.Managed != 0 || len(plan.Comparison.Rulesets) != 0 {
		t.Fatal(plan, err)
	}
	result, err := Apply(load, resolve, Writer{})
	if err != nil || result.Outcome != ApplyUnmanaged || len(result.Operations) != 0 {
		t.Fatal(result, err)
	}
}

func TestRulesetCreationCannotRecordAnAlreadyKnownID(t *testing.T) {
	state := appRulesetState(t)
	desired := state.Policy
	desired.Name = "new"
	declarations, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{desired})
	if err != nil {
		t.Fatal(err)
	}
	recorded := false
	result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Rulesets: &declarations}, nil }, checkoutRepository, Writer{
		ReadRulesets: func(repository.Repository) (settings.RulesetObservation, error) {
			return settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
		},
		Ruleset:   func(repository.Repository, settings.RulesetChange) (int64, error) { return 7, nil },
		CreatedID: func(string, string, int64) error { recorded = true; return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "already-known Ruleset id 7") || recorded || result.Outcome != ApplyFailed || len(result.CreatedIDs) != 1 || result.CreatedIDs[0].ID != 7 {
		t.Fatal(result, err, recorded)
	}
}

func TestRulesetCreationRecordsIDsAndPreservesEarlierLabelRecovery(t *testing.T) {
	for _, failure := range []string{"", "save", "later write", "verification", "missing ID"} {
		t.Run(failure, func(t *testing.T) {
			one, two := appRulesetState(t).Policy, appRulesetState(t).Policy
			one.Name, two.Name = "a", "b"
			rulesets, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{one, two})
			if err != nil {
				t.Fatal(err)
			}
			color := "abcdef"
			labels := appLabelDeclarations(t, settings.LabelDeclaration{Name: "label", Color: &color})
			var calls []string
			cause := errors.New("fixture failure")
			result, err := Apply(func() (settings.Declaration, error) {
				return settings.Declaration{Labels: &labels, Rulesets: &rulesets}, nil
			}, checkoutRepository, Writer{
				ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
					return settings.NewLabelObservation(nil)
				},
				CreateLabel: func(repository.Repository, settings.LabelChange) (int64, error) {
					calls = append(calls, "create label")
					return 17, nil
				},
				ReadRulesets: func(repository.Repository) (settings.RulesetObservation, error) {
					return settings.NewRulesetObservation(nil, 0)
				},
				Ruleset: func(_ repository.Repository, change settings.RulesetChange) (int64, error) {
					calls = append(calls, "create "+change.Name())
					if change.Name() == "b" {
						if failure == "later write" {
							return 0, cause
						}
						return 28, nil
					}
					if failure == "verification" {
						return 27, cause
					}
					if failure == "missing ID" {
						return 0, nil
					}
					return 27, nil
				},
				CreatedID: func(collection, name string, id int64) error {
					calls = append(calls, "save "+collection+"/"+name)
					if collection == "rulesets" && name == "a" && failure == "save" {
						return cause
					}
					return nil
				},
			})
			want := []string{"create label", "save labels/label", "create a", "save rulesets/a", "create b", "save rulesets/b"}
			count := 3
			switch failure {
			case "save":
				want = want[:4]
				count = 2
			case "later write":
				want = want[:5]
				count = 2
			case "verification":
				want = want[:3]
				count = 2
			case "missing ID":
				want = want[:3]
				count = 1
			}
			if !reflect.DeepEqual(calls, want) || len(result.CreatedIDs) != count || result.CreatedIDs[0].Collection != "labels" || result.CreatedIDs[0].ID != 17 {
				t.Fatal(result, calls, want, err)
			}
			if failure == "" {
				if err != nil || !result.Completed() {
					t.Fatal(result, err)
				}
			} else if err == nil || result.Outcome != ApplyPartialFailure {
				t.Fatal(result, err)
			}
		})
	}
}
