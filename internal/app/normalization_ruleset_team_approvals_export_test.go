package app

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestTeamApprovalsExportKeepsSelectedCompleteEvidenceAndPatterns(t *testing.T) {
	for _, mode := range []string{"selected", "only team sink", "nil sinks", "canonical", "unrequested", "read failure", "unacquired", "UI failure"} {
		t.Run(mode, func(t *testing.T) {
			a, z := exportApprovalsState(t, 9, "a", 11, false), exportApprovalsState(t, 7, "z", 11, true)
			for _, state := range []*settings.RulesetState{&a, &z} {
				zero := 0
				empty, patterns := []string{}, []string{"**", "!generated/**", "**"}
				(*state.Policy.Rules)[0].Parameters.RequiredReviewers = &[]settings.RequiredReviewer{
					{Reviewer: settings.ReviewActor{Type: "Team", ID: 10}, MinimumApprovals: &zero, FilePatterns: &patterns},
					{Reviewer: settings.ReviewActor{Type: "Team", ID: 2}, MinimumApprovals: &zero, FilePatterns: &empty},
				}
				state.TeamApprovalsNormalizations = []settings.RulesetTeamApprovalsNormalization{{ReviewerID: 10, Path: settings.RulesetTeamApprovalsPath, RawValue: 11, Fallback: 0}, {ReviewerID: 2, Path: settings.RulesetTeamApprovalsPath, RawValue: -1, Fallback: 0}}
				if mode == "canonical" {
					state.TeamApprovalsNormalizations = nil
					state.RequiredApprovalsNormalization = settings.RulesetRequiredApprovalsNormalization{}
					state.EnforcementNormalization = settings.ObservationNormalization{}
					state.QualitySeverityNormalization = settings.ObservationNormalization{}
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
			cause := errors.New("team acquisition failed")
			reads := 0
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
			var events []string
			saved := false
			checkout := ExportCheckout{Repository: checkoutRepository, IncludeRulesets: mode != "unrequested", PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					ui, err := github.UIFromExportDeclaration(d)
					if err != nil {
						return err
					}
					if mode != "unrequested" {
						entries := *ui.Settings.Rulesets
						if len(entries) != 3 || entries[0].RulesetName != "a" || entries[1].RulesetName != "absent" || entries[1].Rules.RequireAPullRequestBeforeMerging != nil || entries[2].RulesetName != "z" {
							t.Fatal(entries)
						}
						for _, i := range []int{0, 2} {
							teams := *entries[i].Rules.RequireAPullRequestBeforeMerging.RequireReviewFromSpecificTeams
							if len(teams) != 2 || teams[0].Reviewer != 2 || teams[1].Reviewer != 10 || *teams[0].Approvals != 0 || *teams[1].Approvals != 0 || teams[0].FilePatterns == nil || len(*teams[0].FilePatterns) != 0 || !reflect.DeepEqual(*teams[1].FilePatterns, []string{"**", "!generated/**", "**"}) {
								t.Fatal("UI team policy differs", teams)
							}
						}
					}
					saved = true
					return nil
				}, nil
			}}
			if mode != "nil sinks" {
				checkout.RulesetTeamApprovalsNormalizationWarning = func(id int64, name string, n settings.RulesetTeamApprovalsNormalization) {
					events = append(events, fmt.Sprintf("%d:%s:team%d:%d", id, name, n.ReviewerID, n.RawValue))
					n.ReviewerID = 999
				}
				if mode != "only team sink" {
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
				want = []string{"9:a:approvals", "9:a:team2:-1", "9:a:team10:11", "7:z:" + settings.RulesetEnforcementPath, "7:z:approvals", "7:z:team2:-1", "7:z:team10:11", "7:z:" + settings.RulesetQualitySeverityPath}
			case "only team sink":
				want = []string{"9:a:team2:-1", "9:a:team10:11", "7:z:team2:-1", "7:z:team10:11"}
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
				t.Fatal("lost cause", err)
			}
		})
	}
}
