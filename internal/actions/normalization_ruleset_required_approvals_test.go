package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

const approvalsReportPolicy = `{"id":7,"name":"<APPROVAL_OBJECT>","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"pull_request","parameters":{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":false,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":0,"required_review_thread_resolution":false,"required_reviewers":[]}}]}`

func approvalsReportChange(t *testing.T, raw, count int, both bool) settings.RulesetChange {
	t.Helper()
	var before, after settings.RulesetDeclaration
	if err := json.Unmarshal([]byte(approvalsReportPolicy), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(approvalsReportPolicy), &after); err != nil {
		t.Fatal(err)
	}
	records := settings.RulesetNormalizations{RequiredApprovals: settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: raw, Fallback: 0}}
	if count == -1 {
		after.Rules = &[]settings.RulesetRule{}
	} else {
		(*after.Rules)[0].Parameters.RequiredApprovingReviewCount = &count
	}
	if both {
		disabled, severity := "disabled", "errors"
		before.Enforcement, after.Enforcement = &disabled, &disabled
		records.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
		records.QualitySeverity = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "<QUALITY_RAW>", Fallback: "errors"}
		*before.Rules = append(*before.Rules, settings.RulesetRule{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: &severity}})
		*after.Rules = append(*after.Rules, settings.RulesetRule{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: &severity}})
	}
	c, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRulesetRequiredApprovalsPrivateReportKeepsTypedEvidenceAndPrivacy(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, raw := range []int{-1, 11} {
			for _, count := range []int{0, 10, -1} {
				change := approvalsReportChange(t, raw, count, both)
				path := filepath.Join(t.TempDir(), "plan.json")
				var stdout, stderr bytes.Buffer
				status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
					return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Rulesets: []settings.RulesetChange{change}}}, nil
				})
				data, err := os.ReadFile(path)
				if err != nil || status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
					t.Fatal(status, err, stdout.String(), stderr.String())
				}
				var report actions.PlanReport
				if err := json.Unmarshal(data, &report); err != nil {
					t.Fatal(err)
				}
				if report.Outcome != "changed" || report.Managed != 1 || len(report.Rulesets) != 1 || report.Rulesets[0].RequiredApprovalsNormalization == nil || *report.Rulesets[0].RequiredApprovalsNormalization != change.Normalizations().RequiredApprovals {
					t.Fatal("typed metadata lost", string(data))
				}
				entry := report.Rulesets[0]
				if (entry.EnforcementNormalization != nil) != both || (entry.QualitySeverityNormalization != nil) != both || both && (*entry.EnforcementNormalization != change.Normalizations().Enforcement || *entry.QualitySeverityNormalization != change.Normalizations().QualitySeverity) {
					t.Fatal("companions lost or invented", entry)
				}
				content, err := actions.RenderSummary(report, summaryMetadata())
				if err != nil || !strings.Contains(html.UnescapeString(content), presentation.RulesetLine(change.Display())) || strings.Contains(content, "<APPROVAL_OBJECT>") || strings.Contains(content, "<QUALITY_RAW>") {
					t.Fatal(content, err)
				}
				report.Managed = 0
				if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
					t.Fatal("undercount accepted")
				}
				private := filepath.Join(t.TempDir(), "private.md")
				t.Setenv("SETTINGS_TARGET", "owner/current")
				stdout.Reset()
				stderr.Reset()
				status = actions.Run([]string{"summary", path, private, "status-only"}, &stdout, &stderr, "dev", nil)
				data, err = os.ReadFile(private)
				if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(data), "APPROVAL_OBJECT") || strings.Contains(string(data), "QUALITY_RAW") || strings.Contains(string(data), settings.RulesetRequiredApprovalsPath) || strings.Contains(string(data), "Ruleset 7") {
					t.Fatal(status, string(data), stdout.String(), stderr.String(), err)
				}
			}
		}
	}
}

func TestRulesetRequiredApprovalsReportRejectsForgeryAndPresentZero(t *testing.T) {
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) {
			r.Rulesets[0].RequiredApprovalsNormalization = nil
			r.Rulesets[0].EnforcementNormalization = nil
			r.Rulesets[0].QualitySeverityNormalization = nil
		},
		func(r *actions.PlanReport) {
			*r.Rulesets[0].RequiredApprovalsNormalization = settings.RulesetRequiredApprovalsNormalization{}
		},
		func(r *actions.PlanReport) {
			*r.Rulesets[0].EnforcementNormalization = settings.ObservationNormalization{}
		},
		func(r *actions.PlanReport) {
			*r.Rulesets[0].QualitySeverityNormalization = settings.ObservationNormalization{}
		},
		func(r *actions.PlanReport) { r.Rulesets[0].RequiredApprovalsNormalization.RawValue = 0 },
		func(r *actions.PlanReport) { r.Rulesets[0].RequiredApprovalsNormalization.Fallback = 1 },
		func(r *actions.PlanReport) { r.Rulesets[0].RequiredApprovalsNormalization.Path = "other" },
		func(r *actions.PlanReport) { r.Rulesets[0].Before.ID = nil },
		func(r *actions.PlanReport) { *r.Rulesets[0].After.ID = 9 },
		func(r *actions.PlanReport) {
			for _, rule := range *r.Rulesets[0].Before.Rules {
				if rule.Type == "pull_request" {
					one := 1
					rule.Parameters.RequiredApprovingReviewCount = &one
				}
			}
		},
		func(r *actions.PlanReport) {
			for _, rule := range *r.Rulesets[0].After.Rules {
				if rule.Type == "pull_request" {
					eleven := 11
					rule.Parameters.RequiredApprovingReviewCount = &eleven
				}
			}
		},
		func(r *actions.PlanReport) {
			r.Rulesets[0].Operation = "create"
			r.Rulesets[0].Before = nil
			r.Rulesets[0].After.ID = nil
		},
		func(r *actions.PlanReport) { r.Rulesets = append(r.Rulesets, r.Rulesets[0]) },
	} {
		change := approvalsReportChange(t, 11, 0, true)
		records := change.Normalizations()
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Rulesets: []actions.RulesetPlanChange{{Operation: "update", Before: change.Display().Before(), After: change.Display().After(), EnforcementNormalization: &records.Enforcement, QualitySeverityNormalization: &records.QualitySeverity, RequiredApprovalsNormalization: &records.RequiredApprovals}}}
		if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
			t.Fatal("valid triple proof rejected", err)
		}
		mutate(&report)
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
			t.Fatal("forged proof accepted", report)
		}
	}
	for _, raw := range []string{`null`, `{}`, `{"path":"x","raw_value":11}`, `{"path":"x","raw_value":"11","fallback":0}`, `{"path":"x","raw_value":null,"fallback":0}`, `{"path":"x","raw_value":11.5,"fallback":0}`, `{"path":"x","raw_value":11,"fallback":0,"extra":true}`} {
		var entry actions.RulesetPlanChange
		if err := json.Unmarshal([]byte(`{"operation":"update","required_approvals_normalization":`+raw+`}`), &entry); err == nil {
			t.Fatal("malformed integer metadata decoded", raw)
		}
	}
}
