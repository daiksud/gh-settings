package actions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func policyReportFixture() PlanReport {
	return PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, ActionsPolicies: ActionsEventPolicyPlanChanges{{Operation: "create", After: &ActionsEventPolicyReport{Name: "private-policy", Enforcement: "active", RestrictEvents: ActionsEventPolicyEventsReport{AllowedEvents: []string{"push"}}}}}}
}
func TestActionsEventPolicyReportBoundary(t *testing.T) {
	report := policyReportFixture()
	valid, err := validatePlanReport(report, "owner/current")
	if err != nil {
		t.Fatal(err)
	}
	detailed := renderSummary(valid, RunMetadata{WorkflowSHA: strings.Repeat("a", 40), ConfigSHA: strings.Repeat("a", 40)}, []string{"owner", "current", "workflow"})
	if !strings.Contains(detailed, "repository.settings.actions.policies") || !strings.Contains(detailed, "private-policy") || !strings.Contains(detailed, "restrict_events.allowed_events") {
		t.Fatal(detailed)
	}
	report.ActionsPolicies[0].After.RestrictEvents.AllowedEvents[0] = "issues"
	if valid.comparison.ActionsPolicies[0].After().Events[0] != "push" {
		t.Fatal("report aliases validated model")
	}
	for _, mutate := range []func(*PlanReport){
		func(r *PlanReport) { r.Managed = 0 },
		func(r *PlanReport) { r.Outcome = "no-difference" },
		func(r *PlanReport) { r.ActionsPolicies[0].After = nil },
		func(r *PlanReport) { r.ActionsPolicies[0].Operation = "delete" },
		func(r *PlanReport) { r.ActionsPolicies[0].Operation = "update" },
		func(r *PlanReport) { id := int64(7); r.ActionsPolicies[0].After.ID = &id },
		func(r *PlanReport) { r.ActionsPolicies[0].After.RestrictEvents.AllowedEvents = []string{"unknown"} },
		func(r *PlanReport) { r.ActionsPolicies = append(r.ActionsPolicies, r.ActionsPolicies[0]) },
	} {
		r := policyReportFixture()
		mutate(&r)
		if _, err := validatePlanReport(r, "owner/current"); err == nil {
			t.Fatal("invalid report accepted")
		}
	}
	before := settings.ActionsEventPolicy{ID: 7, Name: "before", Enforcement: "disabled", Events: []string{"push"}}
	after := before
	after.Name = "after"
	change, err := settings.NewActionsEventPolicyChange(&before, after)
	if err != nil {
		t.Fatal(err)
	}
	entry := ActionsEventPolicyPlanChange{Operation: change.Operation(), Before: actionsEventPolicyReport(*change.Before()), After: actionsEventPolicyReport(change.After())}
	if _, err := validateActionsEventPolicyReport([]ActionsEventPolicyPlanChange{entry}); err != nil {
		t.Fatal(err)
	}
	*entry.After.ID = 8
	if _, err := validateActionsEventPolicyReport([]ActionsEventPolicyPlanChange{entry}); err == nil {
		t.Fatal("mismatched IDs accepted")
	}
}
func TestActionsEventPolicyJSONClosedSchema(t *testing.T) {
	good := `{"operation":"create","after":{"policy_name":"policy","enforcement_status":"active","restrict_events":{"allowed_events":["push"]}}}`
	for _, data := range []string{
		strings.Replace(good, `"policy_name"`, `"Name"`, 1),
		strings.Replace(good, `"active"`, `null`, 1),
		strings.Replace(good, `"allowed_events":["push"]`, `"allowed_events":["push"],"rules":[]`, 1),
		strings.Replace(good, `"operation":"create"`, `"operation":"create","before":null`, 1),
	} {
		var entry ActionsEventPolicyPlanChange
		if err := json.Unmarshal([]byte(data), &entry); err == nil {
			t.Fatal("invalid JSON accepted", data)
		}
	}
	var report PlanReport
	if err := json.Unmarshal([]byte(`{"actions_policies":null}`), &report); err == nil {
		t.Fatal("null collection accepted")
	}
	var entry ActionsEventPolicyPlanChange
	if err := json.Unmarshal([]byte(good), &entry); err != nil {
		t.Fatal(err)
	}
}
func TestActionsEventPolicyPrivateSummary(t *testing.T) {
	dir := t.TempDir()
	reportPath, summaryPath := filepath.Join(dir, "report"), filepath.Join(dir, "summary")
	t.Setenv("SETTINGS_TARGET", "owner/current")
	after := settings.ActionsEventPolicy{Name: "private-policy", Enforcement: "active", Events: []string{"push"}}
	change, err := settings.NewActionsEventPolicyChange(nil, after)
	if err != nil {
		t.Fatal(err)
	}
	if err := writePlanReport(reportPath, func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, ActionsPolicies: []settings.ActionsEventPolicyChange{change}}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeSummaryMode(reportPath, summaryPath, "status-only", nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"private-policy", "push", "active", "actions.policies"} {
		if strings.Contains(string(data), value) {
			t.Fatal("private summary disclosure", string(data))
		}
	}
	if !strings.Contains(string(data), "Outcome: changed") {
		t.Fatal(string(data))
	}
}
