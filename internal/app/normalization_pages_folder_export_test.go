package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
	"testing"
)

func TestPagesFolderExportWarningsRequireSelectedCompleteSource(t *testing.T) {
	for _, test := range []struct {
		name                          string
		include, normalized, workflow bool
		failure                       string
	}{
		{"normalized legacy", true, true, false, ""},
		{"canonical legacy", true, false, false, ""},
		{"inactive workflow source", true, false, true, ""},
		{"unrequested", false, true, false, ""},
		{"failed acquisition", true, true, false, "read"},
		{"unacquired", true, false, false, "unacquired"},
		{"later conversion failure", true, true, false, "conversion"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/docs"}, HTTPSEnforced: true, Public: false}
			if test.workflow {
				state.BuildType = "workflow"
			}
			record := settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"}
			var observed settings.PagesObservation
			var err error
			if test.normalized {
				state.Source.Path = "/"
				observed, err = settings.NewPagesObservationWithFolderNormalization(state, record)
			} else {
				observed, err = settings.NewPagesObservation(state)
			}
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("Pages unavailable")
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reads := 0
			reader.Pages = func(repository.Repository) (settings.PagesObservation, error) {
				reads++
				if test.failure == "read" {
					return observed, cause
				}
				if test.failure == "unacquired" {
					return settings.PagesObservation{}, nil
				}
				return observed, nil
			}
			reader.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
				if test.failure != "conversion" {
					t.Fatal("unrequested Rulesets")
				}
				state := appRulesetState(t)
				mode := "always"
				actors := []settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: &mode}}
				state.Policy.BypassActors = &actors
				return settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
			}
			saved := false
			var warnings []settings.ObservationNormalization
			err = Export(ExportCheckout{Repository: checkoutRepository, IncludePages: test.include, IncludeRulesets: test.failure == "conversion", NormalizationWarning: func(record settings.ObservationNormalization) { warnings = append(warnings, record) }, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					if _, err := github.UIFromExportDeclaration(d); err != nil {
						return err
					}
					if test.include {
						if d.Pages.BuildType == nil || *d.Pages.BuildType != state.BuildType {
							t.Fatal(d.Pages)
						}
						if test.workflow {
							if d.Pages.Source != nil {
								t.Fatal("inactive source exported", d.Pages)
							}
						} else if d.Pages.Source == nil || *d.Pages.Source != *state.Source {
							t.Fatal("effective source lost", d.Pages)
						}
					}
					saved = true
					return nil
				}, nil
			}}, reader)
			wantReads, wantWarnings := 0, 0
			if test.include {
				wantReads = 1
				if test.normalized && test.failure != "read" {
					wantWarnings = 1
				}
			}
			if (err != nil) != (test.failure != "") || saved != (test.failure == "") || reads != wantReads || len(warnings) != wantWarnings {
				t.Fatal(saved, reads, warnings, err)
			}
			if wantWarnings == 1 && warnings[0] != record {
				t.Fatal("raw evidence lost", warnings)
			}
			if test.failure == "read" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if test.failure == "conversion" && (err == nil || !strings.Contains(err.Error(), "export Rulesets")) {
				t.Fatal("conversion fixture missed target", err)
			}
		})
	}
}
