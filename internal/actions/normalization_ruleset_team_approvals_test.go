package actions

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func teamApprovalsReportChange(t *testing.T, mode string, companions bool) settings.RulesetChange {
	t.Helper()
	base := bypassReportChange(t, "BYPASS_FUTURE", false, true)
	before, after := base.Before(), base.After()
	before.Name, after.Name = "<TEAM_OBJECT>", "<TEAM_OBJECT>"
	records := settings.RulesetNormalizations{}
	if companions {
		records = base.Normalizations()
	}
	for _, policy := range []*settings.RulesetDeclaration{before, after} {
		for _, rule := range *policy.Rules {
			if rule.Type == "pull_request" {
				zero := 0
				patterns := []string{"**", "!generated/**", "**"}
				rule.Parameters.RequiredReviewers = &[]settings.RequiredReviewer{
					{Reviewer: settings.ReviewActor{Type: "Team", ID: 2}, MinimumApprovals: &zero, FilePatterns: &patterns},
					{Reviewer: settings.ReviewActor{Type: "Team", ID: 10}, MinimumApprovals: &zero, FilePatterns: &patterns},
				}
			}
		}
	}
	rules := []settings.RulesetRule{}
	for _, rule := range *after.Rules {
		if rule.Type == "pull_request" {
			switch mode {
			case "different":
				ten := 10
				(*rule.Parameters.RequiredReviewers)[1].MinimumApprovals = &ten
			case "reviewer removed":
				*rule.Parameters.RequiredReviewers = (*rule.Parameters.RequiredReviewers)[:1]
			case "PR removed":
				continue
			}
		}
		rules = append(rules, rule)
	}
	after.Rules = &rules
	records.TeamApprovals = []settings.RulesetTeamApprovalsNormalization{{ReviewerID: 10, Path: settings.RulesetTeamApprovalsPath, RawValue: 11, Fallback: 0}, {ReviewerID: 2, Path: settings.RulesetTeamApprovalsPath, RawValue: -1, Fallback: 0}}
	c, err := settings.NewRulesetChangeWithNormalizations("update", 7, before, after, records)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTeamApprovalsPrivateReportsRetainTypedOwnedCountsAndPrivacy(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mode := range []string{"equal", "different", "reviewer removed", "PR removed"} {
			change := teamApprovalsReportChange(t, mode, companions)
			report, data, path := writeBypassReport(t, change)
			validated, err := validatePlanReport(report, "owner/current")
			if err != nil || report.Managed != 1 || report.Outcome != "changed" || len(validated.rulesets) != 1 || !reflect.DeepEqual(validated.rulesets[0].Normalizations(), change.Normalizations()) {
				t.Fatal("closed evidence lost", string(data), err)
			}
			var wire map[string]any
			json.Unmarshal(data, &wire)
			records := wire["rulesets"].([]any)[0].(map[string]any)["team_approvals_normalizations"].([]any)
			if len(records) != 2 || records[0].(map[string]any)["reviewer_id"] != float64(10) || records[0].(map[string]any)["raw_value"] != float64(11) || records[0].(map[string]any)["fallback"] != float64(0) || records[1].(map[string]any)["raw_value"] != float64(-1) {
				t.Fatal("typed counts lost", records)
			}
			rendered := renderSummary(validated, RunMetadata{Target: "owner/current"}, []string{"owner", "repo", ".github/workflows/settings.yml"})
			if !strings.Contains(html.UnescapeString(rendered), presentation.RulesetLine(change.Display())) || strings.Contains(rendered, "<TEAM_OBJECT>") {
				t.Fatal(rendered)
			}
			private := filepath.Join(t.TempDir(), "private.md")
			t.Setenv("SETTINGS_TARGET", "owner/current")
			var stdout, stderr bytes.Buffer
			status := Run([]string{"summary", path, private, "status-only"}, &stdout, &stderr, "dev", nil)
			privateData, err := os.ReadFile(private)
			if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(privateData), "TEAM_OBJECT") || strings.Contains(string(privateData), "reviewer=") || strings.Contains(string(privateData), settings.RulesetTeamApprovalsPath) {
				t.Fatal(status, string(privateData), err)
			}
			(*report.Rulesets[0].TeamApprovalsNormalizations)[0].ReviewerID = 99
			(*report.Rulesets[0].TeamApprovalsNormalizations)[1].RawValue = 10
			for _, rule := range *report.Rulesets[0].Before.Rules {
				if rule.Type == "pull_request" {
					(*(*rule.Parameters.RequiredReviewers)[0].FilePatterns)[0] = "CORRUPT"
				}
			}
			if !reflect.DeepEqual(validated.rulesets[0].Normalizations(), change.Normalizations()) || presentation.RulesetLine(validated.rulesets[0]) != presentation.RulesetLine(change.Display()) {
				t.Fatal("reconstructed evidence aliases report")
			}
		}
	}
}

func TestTeamApprovalsReportsRejectMissingMalformedAndConflictingEvidence(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for i, mutate := range []func(*PlanReport){
			func(r *PlanReport) { r.Rulesets[0].TeamApprovalsNormalizations = nil },
			func(r *PlanReport) {
				*r.Rulesets[0].TeamApprovalsNormalizations = []RulesetTeamApprovalsNormalization{}
			},
			func(r *PlanReport) {
				(*r.Rulesets[0].TeamApprovalsNormalizations)[0] = RulesetTeamApprovalsNormalization{}
			},
			func(r *PlanReport) { (*r.Rulesets[0].TeamApprovalsNormalizations)[0].ReviewerID = 99 },
			func(r *PlanReport) { (*r.Rulesets[0].TeamApprovalsNormalizations)[0].RawValue = 0 },
			func(r *PlanReport) { (*r.Rulesets[0].TeamApprovalsNormalizations)[0].Fallback = 1 },
			func(r *PlanReport) { (*r.Rulesets[0].TeamApprovalsNormalizations)[0].Path = "other" },
			func(r *PlanReport) {
				(*r.Rulesets[0].TeamApprovalsNormalizations)[1] = (*r.Rulesets[0].TeamApprovalsNormalizations)[0]
			},
			func(r *PlanReport) { r.Rulesets[0].Before.ID = nil },
			func(r *PlanReport) { *r.Rulesets[0].After.ID = 8 },
			func(r *PlanReport) { r.Managed = 0 },
			func(r *PlanReport) {
				for _, rule := range *r.Rulesets[0].Before.Rules {
					if rule.Type == "pull_request" {
						one := 1
						(*rule.Parameters.RequiredReviewers)[1].MinimumApprovals = &one
					}
				}
			},
		} {
			if i == 0 && companions {
				continue
			}
			report, _, _ := writeBypassReport(t, teamApprovalsReportChange(t, "equal", companions))
			mutate(&report)
			if got, err := validatePlanReport(report, "owner/current"); err == nil || !reflect.DeepEqual(got, validatedPlanReport{}) {
				t.Fatal(i, companions, got, err)
			}
		}
	}
	for _, raw := range []string{`null`, `[]`, `[null]`, `[{}]`, `[{"reviewer_id":10,"path":"x","raw_value":11}]`, `[{"reviewer_id":10,"path":"x","raw_value":null,"fallback":0}]`, `[{"reviewer_id":10,"path":"x","raw_value":11.5,"fallback":0}]`, `[{"reviewer_id":"10","path":"x","raw_value":11,"fallback":0}]`, `[{"reviewer_id":10,"path":"x","raw_value":11,"fallback":0,"extra":true}]`, `[{"Reviewer_id":10,"path":"x","raw_value":11,"fallback":0}]`} {
		raw = strings.ReplaceAll(raw, `"path":"x"`, `"path":"`+settings.RulesetTeamApprovalsPath+`"`)
		_, data, _ := writeBypassReport(t, teamApprovalsReportChange(t, "equal", true))
		var wire map[string]any
		json.Unmarshal(data, &wire)
		wire["rulesets"].([]any)[0].(map[string]any)["team_approvals_normalizations"] = json.RawMessage(raw)
		data, _ = json.Marshal(wire)
		var report PlanReport
		err := json.Unmarshal(data, &report)
		if err == nil {
			_, err = validatePlanReport(report, "owner/current")
		}
		if err == nil {
			t.Fatal("malformed metadata accepted", raw)
		}
	}
}
