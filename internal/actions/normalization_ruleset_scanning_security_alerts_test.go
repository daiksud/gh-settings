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

func scanningSecurityAlertsReportChange(t *testing.T, mode string, companions bool) settings.RulesetChange {
	t.Helper()
	base := teamApprovalsReportChange(t, "equal", true)
	before, after := base.Before(), base.After()
	before.Name, after.Name = "<SCAN_OBJECT>", "<SCAN_OBJECT>"
	for _, policy := range []*settings.RulesetDeclaration{before, after} {
		tools := []settings.CodeScanningTool{{Tool: "z tool", SecurityAlertsThreshold: "high_or_higher", AlertsThreshold: "all"}, {Tool: "<CodeQL>", SecurityAlertsThreshold: "high_or_higher", AlertsThreshold: "errors"}}
		if policy == after {
			switch mode {
			case "different":
				tools[0].SecurityAlertsThreshold = "all"
			case "tool removed":
				tools = tools[1:]
			case "replacement":
				tools[0].Tool = "zz new tool"
				tools[0].SecurityAlertsThreshold = "none"
			case "rule removed":
				continue
			}
		}
		*policy.Rules = append(*policy.Rules, settings.RulesetRule{Type: "code_scanning", Parameters: &settings.RuleParameters{CodeScanningTools: &tools}})
	}
	records := settings.RulesetNormalizations{}
	if companions {
		records = base.Normalizations()
	}
	if companions {
		for _, r := range *before.Rules {
			if r.Type == "code_scanning" {
				for i := range *r.Parameters.CodeScanningTools {
					(*r.Parameters.CodeScanningTools)[i].AlertsThreshold = "errors"
				}
			}
		}
		records.ScanningAlerts = []settings.RulesetScanningAlertsNormalization{{Tool: "z tool", Path: settings.RulesetScanningAlertsPath, RawValue: "ALERTS_FUTURE", Fallback: "errors"}, {Tool: "<CodeQL>", Path: settings.RulesetScanningAlertsPath, RawValue: "", Fallback: "errors"}}
	}

	records.ScanningSecurityAlerts = []settings.RulesetScanningSecurityAlertsNormalization{{Tool: "z tool", Path: settings.RulesetScanningSecurityAlertsPath, RawValue: "FUTURE", Fallback: "high_or_higher"}, {Tool: "<CodeQL>", Path: settings.RulesetScanningSecurityAlertsPath, RawValue: "", Fallback: "high_or_higher"}}
	c, err := settings.NewRulesetChangeWithNormalizations("update", 7, before, after, records)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestScanningSecurityAlertsReportsRenderOwnedExactToolChoicesAndPrivacy(t *testing.T) {
	for _, mode := range []string{"equal", "different", "tool removed", "replacement", "rule removed"} {
		change := scanningSecurityAlertsReportChange(t, mode, true)
		report, data, path := writeBypassReport(t, change)
		valid, err := validatePlanReport(report, "owner/current")
		if err != nil || report.Managed != 1 || len(valid.rulesets) != 1 || !reflect.DeepEqual(valid.rulesets[0].Normalizations(), change.Normalizations()) {
			t.Fatal(string(data), err)
		}
		if !bytes.Contains(data, []byte(`"raw_value":""`)) {
			t.Fatal("empty raw string lost", string(data))
		}
		line := presentation.RulesetLine(valid.rulesets[0])
		if strings.Index(line, `tool="<CodeQL>"`) > strings.Index(line, `tool="z tool"`) || strings.Count(line, "Effective policy:") != 1 {
			t.Fatal(line)
		}
		expected := `"high_or_higher" (write required)`
		switch mode {
		case "different":
			expected = `"all" (write required)`
		case "tool removed", "replacement":
			expected = "inactive (tool removed; write required)"
		case "rule removed":
			expected = "inactive (rule removed; write required)"
		}
		if !strings.Contains(line, expected) {
			t.Fatal(mode, line)
		}
		summary := renderSummary(valid, RunMetadata{Target: "owner/current"}, []string{"owner", "repo", ".github/workflows/settings.yml"})
		if strings.Contains(summary, "<SCAN_OBJECT>") || strings.Contains(summary, "<CodeQL>") || !strings.Contains(html.UnescapeString(summary), line) {
			t.Fatal(summary)
		}
		t.Setenv("SETTINGS_TARGET", "owner/current")
		private := filepath.Join(t.TempDir(), "status.md")
		var stdout, stderr bytes.Buffer
		status := Run([]string{"summary", path, private, "status-only"}, &stdout, &stderr, "dev", nil)
		got, err := os.ReadFile(private)
		if status != 0 || err != nil || strings.Contains(string(got), "SCAN_OBJECT") || strings.Contains(string(got), "tool=") || strings.Contains(string(got), settings.RulesetScanningSecurityAlertsPath) {
			t.Fatal(status, string(got), err)
		}
		(*report.Rulesets[0].ScanningSecurityAlertsNormalizations)[0].Tool = "corrupt"
		for _, r := range *report.Rulesets[0].Before.Rules {
			if r.Type == "code_scanning" {
				(*r.Parameters.CodeScanningTools)[0].Tool = "corrupt"
			}
		}
		if presentation.RulesetLine(valid.rulesets[0]) != line {
			t.Fatal("report alias")
		}
	}
}

func TestScanningSecurityAlertsReportsRejectMalformedAndForgedRecords(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mutate := range []func(*PlanReport){
			func(r *PlanReport) { *r.Rulesets[0].ScanningSecurityAlertsNormalizations = nil },
			func(r *PlanReport) { (*r.Rulesets[0].ScanningSecurityAlertsNormalizations)[0].Tool = "other" },
			func(r *PlanReport) {
				(*r.Rulesets[0].ScanningSecurityAlertsNormalizations)[0].RawValue = "high_or_higher"
			},
			func(r *PlanReport) { (*r.Rulesets[0].ScanningSecurityAlertsNormalizations)[0].Fallback = "none" },
			func(r *PlanReport) { (*r.Rulesets[0].ScanningSecurityAlertsNormalizations)[0].Path = "other" },
			func(r *PlanReport) {
				(*r.Rulesets[0].ScanningSecurityAlertsNormalizations)[1] = (*r.Rulesets[0].ScanningSecurityAlertsNormalizations)[0]
			},
			func(r *PlanReport) { *r.Rulesets[0].After.ID = 9 },
		} {
			r, _, _ := writeBypassReport(t, scanningSecurityAlertsReportChange(t, "equal", companions))
			mutate(&r)
			if got, err := validatePlanReport(r, "owner/current"); err == nil || !reflect.DeepEqual(got, validatedPlanReport{}) {
				t.Fatal(got, err)
			}
		}
	}
	for _, member := range []string{"tool", "path", "raw_value", "fallback"} {
		for _, bad := range []any{nil, 1, true} {
			record := map[string]any{"tool": "z tool", "path": settings.RulesetScanningSecurityAlertsPath, "raw_value": "", "fallback": "high_or_higher"}
			record[member] = bad
			data, _ := json.Marshal(record)
			var got RulesetScanningSecurityAlertsNormalization
			if err := json.Unmarshal(data, &got); err == nil {
				t.Fatal(string(data))
			}
			delete(record, member)
			data, _ = json.Marshal(record)
			if err := json.Unmarshal(data, &got); err == nil {
				t.Fatal("missing", string(data))
			}
		}
	}
	for _, key := range []string{"extra", "Tool"} {
		record := map[string]any{"tool": "z tool", "path": settings.RulesetScanningSecurityAlertsPath, "raw_value": "", "fallback": "high_or_higher", key: "z tool"}
		data, _ := json.Marshal(record)
		var got RulesetScanningSecurityAlertsNormalization
		if err := json.Unmarshal(data, &got); err == nil {
			t.Fatal(string(data))
		}
	}
}
