package app

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func exportEnforcementState(t *testing.T, id int64, name, raw string, normalized bool) settings.RulesetState {
	t.Helper()
	var policy settings.RulesetDeclaration
	if err := json.Unmarshal([]byte(`{"name":"policy","target":"branch","enforcement":"disabled","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[]}`), &policy); err != nil {
		t.Fatal(err)
	}
	policy.Name = name
	state := settings.RulesetState{ID: id, Policy: policy}
	if normalized {
		state.EnforcementNormalization = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: raw, Fallback: "disabled"}
	}
	return state
}

func TestRulesetEnforcementExportWarningsUseCompleteSelectedIDs(t *testing.T) {
	type warning struct {
		id     int64
		name   string
		record settings.ObservationNormalization
	}
	for _, test := range []struct {
		name          string
		include, sink bool
		failure       string
	}{
		{"selected", true, true, ""}, {"unrequested", false, true, ""}, {"nil sink", true, false, ""},
		{"read error with value", true, true, "read"}, {"unacquired", true, true, "unacquired"}, {"later UI error", true, true, "ui"},
	} {
		t.Run(test.name, func(t *testing.T) {
			z := exportEnforcementState(t, 7, "z", "FUTURE_MODE", true)
			a := exportEnforcementState(t, 9, "a", "evaluate", true)
			canonical := exportEnforcementState(t, 8, "canonical", "", false)
			if test.failure == "ui" {
				mode := "always"
				canonical.Policy.BypassActors = &[]settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: &mode}}
			}
			observed, err := settings.NewRulesetObservation([]settings.RulesetState{z, canonical, a}, 0)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("Ruleset acquisition failed")
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reads := 0
			reader.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
				reads++
				if test.failure == "read" {
					return observed, cause
				}
				if test.failure == "unacquired" {
					return settings.RulesetObservation{}, nil
				}
				return observed, nil
			}
			var got []warning
			saved := false
			checkout := ExportCheckout{Repository: checkoutRepository, IncludeRulesets: test.include, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					ui, err := github.UIFromExportDeclaration(d)
					if err != nil {
						return err
					}
					if test.include {
						entries := *ui.Settings.Rulesets
						if len(entries) != 3 || entries[0].RulesetName != "a" || entries[2].RulesetName != "z" || *entries[0].EnforcementStatus != "disabled" || *entries[2].EnforcementStatus != "disabled" {
							t.Fatal("effective sorted UI lost", entries)
						}
					}
					saved = true
					return nil
				}, nil
			}}
			if test.sink {
				checkout.RulesetNormalizationWarning = func(id int64, name string, record settings.ObservationNormalization) {
					got = append(got, warning{id, name, record})
				}
			}
			err = Export(checkout, reader)
			var want []warning
			if test.include && test.sink && test.failure != "read" && test.failure != "unacquired" {
				want = []warning{{9, "a", a.EnforcementNormalization}, {7, "z", z.EnforcementNormalization}}
			}
			wantReads := 0
			if test.include {
				wantReads = 1
			}
			if (err != nil) != (test.failure != "") || saved != (test.failure == "") || reads != wantReads || !reflect.DeepEqual(got, want) {
				t.Fatal(saved, reads, got, want, err)
			}
			if test.failure == "read" && !errors.Is(err, cause) {
				t.Fatal("cause lost", err)
			}
		})
	}
}
