package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func exportQualityState(t *testing.T, id int64, name, raw string, both bool) settings.RulesetState {
	t.Helper()
	state := exportEnforcementState(t, id, name, "evaluate", both)
	if !both {
		*state.Policy.Enforcement = "active"
	}
	severity := "errors"
	state.Policy.Rules = &[]settings.RulesetRule{{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: &severity}}}
	state.QualitySeverityNormalization = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: raw, Fallback: "errors"}
	return state
}

func TestRulesetQualityExportPreservesOrderedIndependentRecordsAndRuleAbsence(t *testing.T) {
	type warning struct {
		id     int64
		name   string
		record settings.ObservationNormalization
	}
	for _, both := range []bool{false, true} {
		for _, test := range []struct {
			name                      string
			include, sink, normalized bool
			failure                   string
		}{
			{"selected", true, true, true, ""}, {"canonical", true, true, false, ""}, {"unrequested", false, true, true, ""}, {"nil sink", true, false, true, ""}, {"read error", true, true, true, "read"}, {"unacquired", true, true, true, "unacquired"}, {"later UI error", true, true, true, "ui"},
		} {
			t.Run(test.name, func(t *testing.T) {
				z := exportQualityState(t, 7, "z", "Z_SEVERITY", both)
				a := exportQualityState(t, 9, "a", "A_SEVERITY", false)
				if !test.normalized {
					z.EnforcementNormalization, z.QualitySeverityNormalization, a.QualitySeverityNormalization = settings.ObservationNormalization{}, settings.ObservationNormalization{}, settings.ObservationNormalization{}
				}
				canonical := exportEnforcementState(t, 8, "canonical", "", false)
				known := "warnings"
				canonical.Policy.Rules = &[]settings.RulesetRule{{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: &known}}}
				absent := exportEnforcementState(t, 10, "absent", "", false)
				if test.failure == "ui" {
					mode := "always"
					canonical.Policy.BypassActors = &[]settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: &mode}}
				}
				observed, err := settings.NewRulesetObservation([]settings.RulesetState{z, canonical, absent, a}, 0)
				if err != nil {
					t.Fatal(err)
				}
				reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
				cause := errors.New("quality acquisition failed")
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
				saved := false
				var got []warning
				checkout := ExportCheckout{Repository: checkoutRepository, IncludeRulesets: test.include, PrepareSave: func() (func(settings.Declaration) error, error) {
					return func(d settings.Declaration) error {
						ui, err := github.UIFromExportDeclaration(d)
						if err != nil {
							return err
						}
						if test.include {
							entries := *ui.Settings.Rulesets
							if len(entries) != 4 || entries[0].RulesetName != "a" || entries[1].RulesetName != "absent" || entries[2].RulesetName != "canonical" || entries[3].RulesetName != "z" {
								t.Fatal("export order lost", entries)
							}
							if entries[0].Rules.RequireCodeQualityResults == nil || *entries[0].Rules.RequireCodeQualityResults.Severity != "errors" || entries[1].Rules.RequireCodeQualityResults != nil || *entries[2].Rules.RequireCodeQualityResults.Severity != "warnings_and_higher" || *entries[3].Rules.RequireCodeQualityResults.Severity != "errors" {
								t.Fatal("effective/absent quality lost", entries)
							}
						}
						saved = true
						return nil
					}, nil
				}}
				if test.sink {
					checkout.RulesetNormalizationWarning = func(id int64, name string, n settings.ObservationNormalization) {
						got = append(got, warning{id, name, n})
					}
				}
				err = Export(checkout, reader)
				var want []warning
				if test.include && test.sink && test.normalized && test.failure != "read" && test.failure != "unacquired" {
					want = append(want, warning{9, "a", a.QualitySeverityNormalization})
					if both {
						want = append(want, warning{7, "z", z.EnforcementNormalization})
					}
					want = append(want, warning{7, "z", z.QualitySeverityNormalization})
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
}
