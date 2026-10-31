package actions

import (
	"bytes"
	"encoding/json"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func scanningAlertsReportChange(t *testing.T, mode string, companions bool) settings.RulesetChange {
	t.Helper()
	base := teamApprovalsReportChange(t, "equal", true)
	before, after := base.Before(), base.After()
	before.Name, after.Name = "<SCAN_OBJECT>", "<SCAN_OBJECT>"
	for _, policy := range []*settings.RulesetDeclaration{before, after} {
		tools := []settings.CodeScanningTool{{Tool: "z tool", AlertsThreshold: "errors", SecurityAlertsThreshold: "all"}, {Tool: "<CodeQL>", AlertsThreshold: "errors", SecurityAlertsThreshold: "high_or_higher"}}
		if policy == after {
			switch mode {
			case "different":
				tools[0].AlertsThreshold = "all"
			case "tool removed":
				tools = tools[1:]
			case "replacement":
				tools[0].Tool = "zz new tool"
				tools[0].AlertsThreshold = "none"
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
	records.ScanningAlerts = []settings.RulesetScanningAlertsNormalization{{Tool: "z tool", Path: settings.RulesetScanningAlertsPath, RawValue: "FUTURE", Fallback: "errors"}, {Tool: "<CodeQL>", Path: settings.RulesetScanningAlertsPath, RawValue: "", Fallback: "errors"}}
	c, err := settings.NewRulesetChangeWithNormalizations("update", 7, before, after, records)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestScanningAlertsReportsRenderOwnedExactToolChoicesAndPrivacy(t *testing.T) {
	for _, mode := range []string{"equal", "different", "tool removed", "replacement", "rule removed"} {
		change := scanningAlertsReportChange(t, mode, true)
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
		expected := `"errors" (write required)`
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
		if status != 0 || err != nil || strings.Contains(string(got), "SCAN_OBJECT") || strings.Contains(string(got), "tool=") || strings.Contains(string(got), settings.RulesetScanningAlertsPath) {
			t.Fatal(status, string(got), err)
		}
		(*report.Rulesets[0].ScanningAlertsNormalizations)[0].Tool = "corrupt"
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

func TestScanningAlertsReportsRejectMalformedAndForgedRecords(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mutate := range []func(*PlanReport){
			func(r *PlanReport) { *r.Rulesets[0].ScanningAlertsNormalizations = nil },
			func(r *PlanReport) { (*r.Rulesets[0].ScanningAlertsNormalizations)[0].Tool = "other" },
			func(r *PlanReport) { (*r.Rulesets[0].ScanningAlertsNormalizations)[0].RawValue = "errors" },
			func(r *PlanReport) { (*r.Rulesets[0].ScanningAlertsNormalizations)[0].Fallback = "none" },
			func(r *PlanReport) { (*r.Rulesets[0].ScanningAlertsNormalizations)[0].Path = "other" },
			func(r *PlanReport) {
				(*r.Rulesets[0].ScanningAlertsNormalizations)[1] = (*r.Rulesets[0].ScanningAlertsNormalizations)[0]
			},
			func(r *PlanReport) { *r.Rulesets[0].After.ID = 9 },
		} {
			r, _, _ := writeBypassReport(t, scanningAlertsReportChange(t, "equal", companions))
			mutate(&r)
			if got, err := validatePlanReport(r, "owner/current"); err == nil || !reflect.DeepEqual(got, validatedPlanReport{}) {
				t.Fatal(got, err)
			}
		}
	}
	for _, member := range []string{"tool", "path", "raw_value", "fallback"} {
		for _, bad := range []any{nil, 1, true} {
			record := map[string]any{"tool": "z tool", "path": settings.RulesetScanningAlertsPath, "raw_value": "", "fallback": "errors"}
			record[member] = bad
			data, _ := json.Marshal(record)
			var got RulesetScanningAlertsNormalization
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
		record := map[string]any{"tool": "z tool", "path": settings.RulesetScanningAlertsPath, "raw_value": "", "fallback": "errors", key: "z tool"}
		data, _ := json.Marshal(record)
		var got RulesetScanningAlertsNormalization
		if err := json.Unmarshal(data, &got); err == nil {
			t.Fatal(string(data))
		}
	}
}
