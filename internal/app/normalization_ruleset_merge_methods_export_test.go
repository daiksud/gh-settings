package app

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestMergeMethodsExportKeepsCompleteOwnedEvidenceAndScope(t *testing.T) {
	for _, mode := range []string{"selected", "only methods sink", "nil sinks", "canonical", "unrequested", "read failure", "unacquired", "UI failure"} {
		t.Run(mode, func(t *testing.T) {
			a, z := exportApprovalsState(t, 9, "a", 11, false), exportApprovalsState(t, 7, "z", 11, true)
			for _, s := range []*settings.RulesetState{&a, &z} {
				(*s.Policy.Rules)[0].Parameters.AllowedMergeMethods = &[]string{"merge", "rebase", "squash"}
				s.MergeMethodsNormalization = settings.RulesetMergeMethodsNormalization{Path: settings.RulesetMergeMethodsPath, RawValue: []string{s.Policy.Name + "_METHOD", "squash"}, Fallback: []string{"merge", "squash", "rebase"}}
				if mode == "canonical" {
					s.MergeMethodsNormalization = settings.RulesetMergeMethodsNormalization{}
					s.RequiredApprovalsNormalization = settings.RulesetRequiredApprovalsNormalization{}
					s.EnforcementNormalization = settings.ObservationNormalization{}
					s.QualitySeverityNormalization = settings.ObservationNormalization{}
				}
			}
			absent := exportEnforcementState(t, 8, "absent", "", false)
			if mode == "UI failure" {
				always := "always"
				absent.Policy.BypassActors = &[]settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: &always}}
			}
			observed, err := settings.NewRulesetObservation([]settings.RulesetState{z, absent, a}, 0)
			if err != nil {
				t.Fatal(err)
			}
			original, _ := observed.Values()
			reads := 0
			cause := errors.New("incomplete acquisition")
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reader.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
				reads++
				if mode == "read failure" {
					return observed, cause
				}
				if mode == "unacquired" {
					return settings.RulesetObservation{}, nil
				}
				return observed, nil
			}
			saved := false
			var events []string
			checkout := ExportCheckout{Repository: checkoutRepository, IncludeRulesets: mode != "unrequested", PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					ui, err := github.UIFromExportDeclaration(d)
					if err != nil {
						return err
					}
					if mode != "unrequested" {
						entries := *ui.Settings.Rulesets
						if len(entries) != 3 || entries[0].RulesetName != "a" || entries[1].RulesetName != "absent" || entries[2].RulesetName != "z" || entries[1].Rules.RequireAPullRequestBeforeMerging != nil {
							t.Fatal(entries)
						}
						for _, i := range []int{0, 2} {
							methods := *entries[i].Rules.RequireAPullRequestBeforeMerging.AllowedMergeMethods
							if len(methods) != 3 || methods[0] != "merge" || methods[1] != "rebase" || methods[2] != "squash" {
								t.Fatal("effective export corrupted", methods)
							}
						}
					}
					saved = true
					return nil
				}, nil
			}}
			if mode != "nil sinks" {
				checkout.RulesetMergeMethodsNormalizationWarning = func(id int64, name string, n settings.RulesetMergeMethodsNormalization) {
					events = append(events, fmt.Sprintf("%d:%s:methods", id, name))
					if !slices.Equal(n.RawValue, []string{name + "_METHOD", "squash"}) || !slices.Equal(n.Fallback, []string{"merge", "squash", "rebase"}) {
						t.Fatal(n)
					}
					n.RawValue[0], n.Fallback[0] = "CORRUPT", "CORRUPT"
				}
				if mode != "only methods sink" {
					checkout.RulesetNormalizationWarning = func(id int64, name string, n settings.ObservationNormalization) {
						events = append(events, fmt.Sprintf("%d:%s:%s", id, name, n.Path))
					}
					checkout.RulesetRequiredApprovalsNormalizationWarning = func(id int64, name string, n settings.RulesetRequiredApprovalsNormalization) {
						events = append(events, fmt.Sprintf("%d:%s:approvals", id, name))
					}
				}
			}
			err = Export(checkout, reader)
			var want []string
			switch mode {
			case "selected", "UI failure":
				want = []string{"9:a:approvals", "9:a:methods", "7:z:" + settings.RulesetEnforcementPath, "7:z:approvals", "7:z:methods", "7:z:" + settings.RulesetQualitySeverityPath}
			case "only methods sink":
				want = []string{"9:a:methods", "7:z:methods"}
			}
			failed := mode == "read failure" || mode == "unacquired" || mode == "UI failure"
			wantReads := 1
			if mode == "unrequested" {
				wantReads = 0
			}
			after, _ := observed.Values()
			if (err != nil) != failed || saved == failed || reads != wantReads || !reflect.DeepEqual(events, want) || !reflect.DeepEqual(original, after) {
				t.Fatal(mode, err, saved, reads, events, want)
			}
			if mode == "read failure" && !errors.Is(err, cause) {
				t.Fatal("lost acquisition cause", err)
			}
		})
	}
}
