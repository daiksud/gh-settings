package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func appRulesets(t *testing.T, body string) *settings.RulesetDeclarations {
	t.Helper()
	d, err := workspace.DecodeConfig([]byte("schemaVersion: 1\nsettings: {rulesets: " + body + "}"))
	if err != nil {
		t.Fatal(err)
	}
	return d.Rulesets
}
func appRulesetState(t *testing.T) settings.RulesetState {
	d := appRulesets(t, `[{name: basic, target: branch, enforcement: active, conditions: {ref_name: {include: ['~ALL'], exclude: []}}, bypass_actors: [], rules: [{type: deletion}]}]`)
	return settings.RulesetState{ID: 7, Policy: d.Values()[0]}
}
func TestRulesetsAppFreshReadOrderAndFailures(t *testing.T) {
	for _, fail := range []string{"", "read", "write"} {
		var calls []string
		state := appRulesetState(t)
		desired := settings.Declaration{Rulesets: appRulesets(t, `[{name: basic, enforcement: disabled}]`)}
		yes := true
		desired.Repository.HasIssues = &yes
		labels, _ := settings.NewLabelDeclarations(nil)
		desired.Labels = &labels
		load := func() (settings.Declaration, error) { return desired, nil }
		target := func() (repository.Repository, error) {
			return repository.Repository{Owner: "owner", Name: "current"}, nil
		}
		result, err := Apply(load, target, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
			calls = append(calls, "repository")
			return nil
		}, ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			calls = append(calls, "labels")
			return settings.NewLabelObservation(nil)
		}, ReadRulesets: func(repository.Repository) (settings.RulesetObservation, error) {
			calls = append(calls, "rulesets-read")
			if fail == "read" {
				return settings.RulesetObservation{}, errors.New("denied")
			}
			return settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
		}, Ruleset: func(_ repository.Repository, c settings.RulesetChange) error {
			calls = append(calls, "ruleset-write")
			if c.ID() != 7 || *c.After().Enforcement != "disabled" || len(*c.After().Rules) != 1 {
				t.Fatal(c)
			}
			if fail == "write" {
				return errors.New("denied")
			}
			return nil
		}})
		want := []string{"repository", "rulesets-read"}
		if fail != "read" {
			want = append(want, "ruleset-write")
		}
		if !reflect.DeepEqual(calls, want) || (fail == "") != (err == nil) {
			t.Fatalf("%v %v", calls, err)
		}
		if fail != "" && result.Outcome != ApplyPartialFailure {
			t.Fatal(result)
		}
		if fail == "write" && result.Operations[len(result.Operations)-1].Status != OperationFailedUnknown {
			t.Fatal(result)
		}
	}
}
func TestRulesetsAppPreflightsWholeCollectionAndNoops(t *testing.T) {
	state := appRulesetState(t)
	observed, _ := settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
	for _, body := range []string{`[{name: basic}]`, `[{name: basic, enforcement: disabled},{name: z-missing}]`} {
		writes := 0
		result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Rulesets: appRulesets(t, body)}, nil }, checkoutRepository, Writer{ReadRulesets: func(repository.Repository) (settings.RulesetObservation, error) { return observed, nil }, Ruleset: func(repository.Repository, settings.RulesetChange) error { writes++; return nil }})
		if writes != 0 {
			t.Fatal("partial collection written")
		}
		if strings.Contains(body, "missing") {
			if err == nil || result.Completed() {
				t.Fatal("incomplete create passed")
			}
		} else if err != nil || !result.Completed() {
			t.Fatal(result, err)
		}
	}
	// Omitted resource must not instantiate any Rulesets client.
	if _, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{}, nil }, nil, Writer{}); err != nil {
		t.Fatal(err)
	}
}
func TestRulesetsPlanFailureDiscardsEarlierComparison(t *testing.T) {
	yes := true
	desired := settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &yes}, Rulesets: appRulesets(t, `[]`)}
	result, err := Plan(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Reader{ObserveRepository: observeBaseFixture(func(repository.Repository) (settings.RepositoryState, error) { return settings.RepositoryState{}, nil }), Rulesets: func(repository.Repository) (settings.RulesetObservation, error) {
		return settings.RulesetObservation{}, errors.New("unsupported policy")
	}})
	if err == nil || result.Comparison.Managed != 0 || len(result.Comparison.Changes) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestRulesetsExportOptInAndPreservesDestination(t *testing.T) {
	for _, include := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			saved := false
			readRulesets := 0
			excluded := 0
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reader.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
				readRulesets++
				if fail {
					return settings.RulesetObservation{}, errors.New("hidden bypass")
				}
				return settings.NewRulesetObservation([]settings.RulesetState{appRulesetState(t)}, 2)
			}
			err := Export(ExportCheckout{Repository: checkoutRepository, IncludeRulesets: include, ExcludedRulesets: func(n int) { excluded = n }, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saved = true
					if (d.Rulesets != nil) != include {
						t.Fatal("presence")
					}
					return nil
				}, nil
			}}, reader)
			if include {
				if readRulesets != 1 || (fail && saved) || (!fail && (!saved || excluded != 2)) {
					t.Fatal(saved, readRulesets, excluded)
				}
			} else if readRulesets != 0 || !saved || err != nil {
				t.Fatal("default export changed")
			}
			if (err != nil) != (include && fail) {
				t.Fatal(err)
			}
		}
	}
}
