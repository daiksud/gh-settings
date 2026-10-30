package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func mergeMethodsReportChange(t *testing.T, methods []string, companion bool) settings.RulesetChange {
	t.Helper()
	var before, after settings.RulesetDeclaration
	fixture := strings.Replace(approvalsReportPolicy, `["squash"]`, `["merge","rebase","squash"]`, 1)
	if err := json.Unmarshal([]byte(fixture), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(fixture), &after); err != nil {
		t.Fatal(err)
	}
	if methods == nil {
		after.Rules = &[]settings.RulesetRule{}
	} else {
		(*after.Rules)[0].Parameters.AllowedMergeMethods = &methods
	}
	records := settings.RulesetNormalizations{MergeMethods: settings.RulesetMergeMethodsNormalization{Path: settings.RulesetMergeMethodsPath, RawValue: []string{"squash", "<RAW_METHOD>\n"}, Fallback: []string{"merge", "squash", "rebase"}}}
	if companion {
		records.RequiredApprovals = settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: 11, Fallback: 0}
	}
	c, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestMergeMethodsPrivateReportsKeepTypedListsAndPrivacy(t *testing.T) {
	for _, methods := range [][]string{{"merge", "rebase", "squash"}, {"squash"}, nil} {
		change := mergeMethodsReportChange(t, methods, true)
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
		if report.Managed != 1 || report.Outcome != "changed" || len(report.Rulesets) != 1 {
			t.Fatal(string(data))
		}
		record := report.Rulesets[0].MergeMethodsNormalization
		if record == nil || record.Path != settings.RulesetMergeMethodsPath || !reflect.DeepEqual(record.RawValue, []string{"squash", "<RAW_METHOD>\n"}) || !reflect.DeepEqual(record.Fallback, []string{"merge", "squash", "rebase"}) || report.Rulesets[0].RequiredApprovalsNormalization == nil {
			t.Fatal("typed lists/companion lost", string(data))
		}
		rendered, err := actions.RenderSummary(report, summaryMetadata())
		if err != nil || !strings.Contains(html.UnescapeString(rendered), presentation.RulesetLine(change.Display())) || strings.Contains(rendered, "<RAW_METHOD>") || strings.Contains(rendered, "<APPROVAL_OBJECT>") {
			t.Fatal(rendered, err)
		}
		report.Managed = 0
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
			t.Fatal("undercount accepted")
		}
		output := filepath.Join(t.TempDir(), "summary.md")
		t.Setenv("SETTINGS_TARGET", "owner/current")
		stdout.Reset()
		stderr.Reset()
		status = actions.Run([]string{"summary", path, output, "status-only"}, &stdout, &stderr, "dev", nil)
		data, err = os.ReadFile(output)
		if err != nil || status != 0 || stderr.Len() != 0 || stdout.String() != "plan-outcome=changed\n" || strings.Contains(string(data), "RAW_METHOD") || strings.Contains(string(data), "APPROVAL_OBJECT") || strings.Contains(string(data), "allowed_merge_methods") || strings.Contains(string(data), "Ruleset 7") {
			t.Fatal(status, err, string(data), stderr.String())
		}
	}
}

func TestMergeMethodsReportsRejectUnboundOrMalformedEvidence(t *testing.T) {
	for _, companion := range []bool{false, true} {
		for _, mutate := range []func(*actions.RulesetPlanChange){
			func(e *actions.RulesetPlanChange) {
				e.MergeMethodsNormalization = &actions.RulesetMergeMethodsNormalization{}
			},
			func(e *actions.RulesetPlanChange) { e.MergeMethodsNormalization.RawValue = nil },
			func(e *actions.RulesetPlanChange) { e.MergeMethodsNormalization.RawValue = []string{} },
			func(e *actions.RulesetPlanChange) { e.MergeMethodsNormalization.RawValue = []string{"merge"} },
			func(e *actions.RulesetPlanChange) {
				e.MergeMethodsNormalization.RawValue = []string{"future", "future"}
			},
			func(e *actions.RulesetPlanChange) { e.MergeMethodsNormalization.Fallback = []string{"merge"} },
			func(e *actions.RulesetPlanChange) { e.MergeMethodsNormalization.Path = "other" },
			func(e *actions.RulesetPlanChange) { e.Before.ID = nil },
			func(e *actions.RulesetPlanChange) { *e.After.ID = 9 },
			func(e *actions.RulesetPlanChange) {
				(*e.Before.Rules)[0].Parameters.AllowedMergeMethods = &[]string{"squash"}
			},
			func(e *actions.RulesetPlanChange) { e.Operation = "create"; e.Before = nil; e.After.ID = nil },
		} {
			change := mergeMethodsReportChange(t, []string{"merge", "rebase", "squash"}, companion)
			records := change.Normalizations()
			e := actions.RulesetPlanChange{Operation: "update", Before: change.Display().Before(), After: change.Display().After(), MergeMethodsNormalization: &actions.RulesetMergeMethodsNormalization{Path: records.MergeMethods.Path, RawValue: records.MergeMethods.RawValue, Fallback: records.MergeMethods.Fallback}}
			if companion {
				e.RequiredApprovalsNormalization = &records.RequiredApprovals
			}
			report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Rulesets: []actions.RulesetPlanChange{e}}
			if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
				t.Fatal("valid report", err)
			}
			mutate(&report.Rulesets[0])
			if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
				t.Fatal("forged or empty record accepted", report)
			}
		}
	}
	for _, raw := range []string{`null`, `{}`, `{"path":"x","raw_value":["future"]}`, `{"path":"x","raw_value":null,"fallback":[]}`, `{"path":"x","raw_value":[null],"fallback":[]}`, `{"path":"x","raw_value":"future","fallback":[]}`, `{"path":"x","raw_value":[1],"fallback":[]}`, `{"path":"x","raw_value":[],"fallback":null}`, `{"path":"x","raw_value":[],"fallback":[],"extra":true}`, `{"Path":"x","raw_value":[],"fallback":[]}`} {
		var entry actions.RulesetPlanChange
		if err := json.Unmarshal([]byte(`{"operation":"update","merge_methods_normalization":`+raw+`}`), &entry); err == nil {
			t.Fatal("malformed lists decoded", raw)
		}
	}
}
