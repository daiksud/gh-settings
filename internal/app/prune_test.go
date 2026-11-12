package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func mustPrune(t *testing.T, value string) settings.PruneScopes {
	t.Helper()
	p, err := settings.ParsePruneScopes(value)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestPrunePreflightsAllCandidatesBeforeAnyWrite(t *testing.T) {
	labels, _ := settings.NewLabelDeclarations(nil)
	rules, _ := settings.NewRulesetDeclarations(nil)
	yes := true
	d := settings.Declaration{Labels: &labels, Rulesets: &rules}
	d.Repository.HasIssues = &yes
	calls := []string{}
	_, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			calls = append(calls, "labels")
			return settings.NewLabelObservation([]settings.LabelState{{ID: 1, Name: "old", Color: "abcdef"}})
		},
		ReadRulesets: func(repository.Repository) (settings.RulesetObservation, error) {
			calls = append(calls, "rulesets")
			return settings.RulesetObservation{}, errors.New("page 2 denied")
		},
		Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
			t.Fatal("write before all candidates validated")
			return nil
		},
	}, mustPrune(t, "labels,rulesets"))
	if err == nil || !strings.Contains(err.Error(), "before any writes") || !reflect.DeepEqual(calls, []string{"labels", "rulesets"}) {
		t.Fatal(err, calls)
	}
}
func TestPruneLabelsWritesCreateBeforeFixedDeletionAndReportsPartial(t *testing.T) {
	for _, fail := range []bool{false, true} {
		color := "abcdef"
		labels, _ := settings.NewLabelDeclarations([]settings.LabelDeclaration{{Name: "new", Color: &color}})
		current, _ := settings.NewLabelObservation([]settings.LabelState{{ID: 7, Name: "old", Color: "abcdef"}})
		calls := []string{}
		result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Labels: &labels}, nil }, checkoutRepository, Writer{
			ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
				calls = append(calls, "read")
				return current, nil
			},
			CreateLabel: func(_ repository.Repository, c settings.LabelChange) (int64, error) {
				calls = append(calls, "create:"+c.Name())
				return 9, nil
			},
			DeleteLabel: func(_ repository.Repository, c settings.LabelChange) error {
				calls = append(calls, "delete:"+c.Name())
				if fail {
					return errors.New("readback failed; write may have applied")
				}
				return nil
			},
		}, mustPrune(t, "labels"))
		if (err != nil) != fail || !reflect.DeepEqual(calls, []string{"read", "read", "create:new", "delete:old"}) || len(result.CreatedIDs) != 1 || result.CreatedIDs[0].ID != 9 {
			t.Fatal(result, calls, err)
		}
		if fail && result.Outcome != ApplyPartialFailure {
			t.Fatal(result)
		}
	}
}
func TestPruneCandidateDriftFailsBeforeCollectionWrite(t *testing.T) {
	labels, _ := settings.NewLabelDeclarations(nil)
	reads := 0
	_, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Labels: &labels}, nil }, checkoutRepository, Writer{
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			reads++
			v := []settings.LabelState{{ID: 7, Name: "old", Color: "abcdef"}}
			if reads == 2 {
				v = append(v, settings.LabelState{ID: 8, Name: "concurrent", Color: "abcdef"})
			}
			return settings.NewLabelObservation(v)
		},
		DeleteLabel: func(repository.Repository, settings.LabelChange) error {
			t.Fatal("expanded candidate set deleted")
			return nil
		},
	}, mustPrune(t, "labels"))
	if err == nil || !strings.Contains(err.Error(), "changed after preflight") {
		t.Fatal(err)
	}
}
func TestPruneOmittedCollectionsRemainLocalNoop(t *testing.T) {
	p := mustPrune(t, "labels,rulesets")
	resolve := func() (repository.Repository, error) {
		t.Fatal("omitted collection resolved target")
		return repository.Repository{}, nil
	}
	load := func() (settings.Declaration, error) { return settings.Declaration{}, nil }
	if _, err := Plan(load, resolve, Reader{}, p); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(load, resolve, Writer{}, p); err != nil {
		t.Fatal(err)
	}
}

func TestPruneRulesetsCreatesBeforeAlphabeticallyEarlierDelete(t *testing.T) {
	state := appRulesetState(t)
	desired := appRulesets(t, `[{name: z-new, enforcement: active, conditions: {ref_name: {include: ["~ALL"]}}}]`)
	var calls []string
	result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Rulesets: desired}, nil }, checkoutRepository, Writer{
		ReadRulesets: func(repository.Repository) (settings.RulesetObservation, error) {
			return settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
		},
		Ruleset: func(_ repository.Repository, c settings.RulesetChange) (int64, error) {
			calls = append(calls, c.Operation()+":"+c.Name())
			if c.Operation() == "create" {
				return 9, nil
			}
			if c.ID() != 7 {
				t.Fatal("new object selected for deletion", c)
			}
			return c.ID(), nil
		},
	}, mustPrune(t, "rulesets"))
	if err != nil || !reflect.DeepEqual(calls, []string{"create:z-new", "delete:basic"}) || len(result.CreatedIDs) != 1 {
		t.Fatal(calls, result, err)
	}
}
func TestPruneLabelCreationCannotAdoptDeletionCandidateID(t *testing.T) {
	color := "abcdef"
	desired, _ := settings.NewLabelDeclarations([]settings.LabelDeclaration{{Name: "new", Color: &color}})
	_, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Labels: &desired}, nil }, checkoutRepository, Writer{
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			return settings.NewLabelObservation([]settings.LabelState{{ID: 7, Name: "old", Color: "abcdef"}})
		},
		CreateLabel: func(repository.Repository, settings.LabelChange) (int64, error) { return 7, nil },
		DeleteLabel: func(repository.Repository, settings.LabelChange) error {
			t.Fatal("creation reused candidate ID and deletion continued")
			return nil
		},
	}, mustPrune(t, "labels"))
	if err == nil || !strings.Contains(err.Error(), "already-known label id") {
		t.Fatal(err)
	}
}
