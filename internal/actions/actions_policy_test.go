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
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsNativePlanAndSummaryUnavailableValues(t *testing.T) {
	selected, githubOwned, verified := "selected", false, false
	patterns := []string{"org/*", "!org/<blocked>@*", "org/*"}
	d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{AllowedActions: &selected, SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: &githubOwned, VerifiedAllowed: &verified, PatternsAllowed: &patterns}}}
	comparison, err := settings.CompareActionsPolicy(settings.ActionsPolicyState{AllowedActions: "all"}, d.Actions)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: comparison.Managed, Actions: comparison.Changes}}, nil
	})
	data, err := os.ReadFile(path)
	if status != 0 || err != nil {
		t.Fatal(status, err, stderr.String())
	}
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "changed" || report.Managed != 4 || len(report.Actions) != 4 || report.Actions[1].Before != nil || report.Actions[1].After != false {
		t.Fatal(string(data))
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || strings.Contains(content, "<blocked>") || !strings.Contains(html.UnescapeString(content), "(unavailable while policy is not selected) → false") {
		t.Fatal(content, err)
	}
	for _, before := range []any{true, false} {
		forged := report
		forged.Actions = append([]actions.PlanChange{}, report.Actions...)
		forged.Actions[1].Before = before
		if _, err := actions.RenderSummary(forged, summaryMetadata()); err == nil {
			t.Fatal("inactive selected value presented as observed", before)
		}
	}
	forged := report
	forged.Actions = append([]actions.PlanChange{}, report.Actions...)
	forged.Actions[1].Before = true
	forged.Actions[2].Before = true
	forged.Actions[3].Before = []any{"invented/*"}
	if _, err := actions.RenderSummary(forged, summaryMetadata()); err == nil {
		t.Fatal("all inactive selected values presented as observed")
	}
	for _, invalid := range []actions.PlanReport{
		{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Actions: report.Actions[1:2]},
		{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: "actions.enabled", Before: nil, After: false}}},
		{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: "actions.allowed_actions", Before: "all", After: "unknown"}}},
		{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: "actions.selected_actions.patterns_allowed", Before: []any{}, After: []any{false}}}},
		{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: "actions.allowed_actions", Before: "selected", After: "all"}, {Field: "actions.selected_actions.verified_allowed", Before: true, After: false}}},
		{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Actions: []actions.PlanChange{{Field: "actions.allowed_actions", Before: "all", After: "selected"}}},
	} {
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatal("invalid native/unknown report accepted", invalid)
		}
	}
}
