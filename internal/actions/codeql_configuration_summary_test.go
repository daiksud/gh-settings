package actions_test

import (
	"github.com/daiksud/gh-settings/internal/actions"
	"strings"
	"testing"
)

func TestCodeQLConfigurationSummaryEscapesRunnerLabel(t *testing.T) {
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, CodeQLConfiguration: []actions.PlanChange{{Field: "threat_model", Before: "remote_sources", After: "remote_and_local_sources"}, {Field: "runner", Before: map[string]any{"type": "standard_github_runner"}, After: map[string]any{"type": "labeled_runner", "label": "<&pool>"}}}}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(content, "codeql_analysis.threat_model:") || !strings.Contains(content, "remote_and_local_sources") || !strings.Contains(content, "repository.settings.advanced_security.code_security.codeql_analysis.runner:") || strings.Contains(content, "<&pool>") || !strings.Contains(content, "&lt;&amp;pool&gt;") {
		t.Fatal(content, err)
	}
}
