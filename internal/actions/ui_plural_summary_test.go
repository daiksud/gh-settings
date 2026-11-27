package actions_test

import (
	"encoding/json"
	"github.com/daiksud/gh-settings/internal/actions"
	"strings"
	"testing"
)

func TestCodeQLQuerySuiteSummaryUsesScanSettingsPath(t *testing.T) {
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, CodeQLConfiguration: []actions.PlanChange{{Field: "query_suite", Before: "default", After: "extended"}}}
	text, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(text, "codeql_analysis.scan_settings.query_suite:") {
		t.Fatal(text, err)
	}
	data, err := json.Marshal(report)
	if err != nil || !strings.Contains(string(data), `"field":"query_suite"`) || strings.Contains(string(data), `"field":"query_suites"`) {
		t.Fatalf("native report changed: %s %v", data, err)
	}
}
