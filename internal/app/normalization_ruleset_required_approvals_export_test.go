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

func exportApprovalsState(t *testing.T, id int64, name string, raw int, both bool) settings.RulesetState {
	t.Helper()
	state := exportEnforcementState(t, id, name, "evaluate", both)
	var rules []settings.RulesetRule
	const pr = `[{"type":"pull_request","parameters":{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":false,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":0,"required_review_thread_resolution":false,"required_reviewers":[]}}]`
	if err := json.Unmarshal([]byte(pr), &rules); err != nil {
		t.Fatal(err)
	}
	state.Policy.Rules = &rules
	state.RequiredApprovalsNormalization = settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: raw, Fallback: 0}
	if both {
		severity := "errors"
		*state.Policy.Rules = append(*state.Policy.Rules, settings.RulesetRule{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: &severity}})
		state.QualitySeverityNormalization = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "Q_SEVERITY", Fallback: "errors"}
	}
	return state
}

func TestRulesetRequiredApprovalsExportOrdersTypedWarningsAndPreservesAbsence(t *testing.T) {
	type warning struct {
		id      int64
		name    string
		text    settings.ObservationNormalization
		integer settings.RulesetRequiredApprovalsNormalization
	}
	for _, both := range []bool{false, true} {
		for _, sinks := range []int{0, 1, 2, 3} {
			for _, mode := range []string{"selected", "canonical", "unrequested", "read", "unacquired", "ui"} {
				z, a := exportApprovalsState(t, 7, "z", 11, both), exportApprovalsState(t, 9, "a", 12, false)
				if mode == "canonical" {
					z.RequiredApprovalsNormalization, a.RequiredApprovalsNormalization = settings.RulesetRequiredApprovalsNormalization{}, settings.RulesetRequiredApprovalsNormalization{}
					z.EnforcementNormalization, z.QualitySeverityNormalization = settings.ObservationNormalization{}, settings.ObservationNormalization{}
				}
				canonical := exportApprovalsState(t, 8, "canonical", 11, false)
				canonical.RequiredApprovalsNormalization = settings.RulesetRequiredApprovalsNormalization{}
				ten := 10
				(*canonical.Policy.Rules)[0].Parameters.RequiredApprovingReviewCount = &ten
				absent := exportEnforcementState(t, 10, "absent", "", false)
				if mode == "ui" {
					always := "always"
					canonical.Policy.BypassActors = &[]settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: &always}}
				}
				observed, err := settings.NewRulesetObservation([]settings.RulesetState{z, canonical, absent, a}, 0)
				if err != nil {
					t.Fatal(err)
				}
				reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
				cause := errors.New("approval acquisition failed")
				reads := 0
				reader.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
					reads++
					if mode == "read" {
						return observed, cause
					}
					if mode == "unacquired" {
						return settings.RulesetObservation{}, nil
					}
					return observed, nil
				}
				saved := false
				var got []warning
				checkout := ExportCheckout{Repository: checkoutRepository, IncludeRulesets: mode != "unrequested", PrepareSave: func() (func(settings.Declaration) error, error) {
					return func(d settings.Declaration) error {
						ui, err := github.UIFromExportDeclaration(d)
						if err != nil {
							return err
						}
						if mode != "unrequested" {
							entries := *ui.Settings.Rulesets
							if len(entries) != 4 || entries[0].RulesetName != "a" || entries[1].RulesetName != "absent" || entries[2].RulesetName != "canonical" || entries[3].RulesetName != "z" || *entries[0].Rules.RequireAPullRequestBeforeMerging.RequiredApprovals != 0 || entries[1].Rules.RequireAPullRequestBeforeMerging != nil || *entries[2].Rules.RequireAPullRequestBeforeMerging.RequiredApprovals != 10 || *entries[3].Rules.RequireAPullRequestBeforeMerging.RequiredApprovals != 0 {
								t.Fatal("effective/absent integer UI export", entries)
							}
						}
						saved = true
						return nil
					}, nil
				}}
				if sinks&1 != 0 {
					checkout.RulesetNormalizationWarning = func(id int64, name string, n settings.ObservationNormalization) {
						got = append(got, warning{id: id, name: name, text: n})
					}
				}
				if sinks&2 != 0 {
					checkout.RulesetRequiredApprovalsNormalizationWarning = func(id int64, name string, n settings.RulesetRequiredApprovalsNormalization) {
						got = append(got, warning{id: id, name: name, integer: n})
					}
				}
				err = Export(checkout, reader)
				var want []warning
				if mode == "selected" || mode == "ui" {
					if sinks&2 != 0 {
						want = append(want, warning{id: 9, name: "a", integer: settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: 12, Fallback: 0}})
					}
					if both && sinks&1 != 0 {
						want = append(want, warning{id: 7, name: "z", text: settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}})
					}
					if sinks&2 != 0 {
						want = append(want, warning{id: 7, name: "z", integer: settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: 11, Fallback: 0}})
					}
					if both && sinks&1 != 0 {
						want = append(want, warning{id: 7, name: "z", text: settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "Q_SEVERITY", Fallback: "errors"}})
					}
				}
				failed := mode == "read" || mode == "unacquired" || mode == "ui"
				wantReads := 1
				if mode == "unrequested" {
					wantReads = 0
				}
				if (err != nil) != failed || saved == failed || reads != wantReads || !reflect.DeepEqual(got, want) {
					t.Fatal(mode, sinks, both, saved, reads, got, want, err)
				}
				if mode == "read" && !errors.Is(err, cause) {
					t.Fatal("acquisition cause lost", err)
				}
			}
		}
	}
}
