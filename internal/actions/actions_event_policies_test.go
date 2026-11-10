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

func reportBool(value bool) *bool { return &value }

func policyReportFixture() PlanReport {
	return PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, ActionsPolicies: ActionsEventPolicyPlanChanges{{Operation: "create", After: &ActionsEventPolicyReport{Name: "private-policy", Enforcement: "active", RestrictEvents: ActionsEventPolicyEventsReport{AllowedEvents: []string{"push"}}, RestrictActors: ActionsEventPolicyActorsReport{Enabled: reportBool(false)}}}}}
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
	good := `{"operation":"create","after":{"policy_name":"policy","enforcement_status":"active","restrict_events":{"allowed_events":["push"]},"restrict_actors":{"enabled":false}}}`
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
	actors := []settings.ActionsPolicyActor{{Type: "User", ID: 1234567}}
	after := settings.ActionsEventPolicy{Name: "private-policy", Enforcement: "active", Events: []string{"push"}, Actors: &actors, TargetWorkflows: &settings.ActionsWorkflowTarget{Include: []string{"private-workflow"}, Exclude: []string{"secret-pattern"}}}
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
	for _, value := range []string{"private-policy", "push", "active", "actions.policies", "private-workflow", "secret-pattern", "target_workflows", "1234567", "restrict_actors", "allowed_actors"} {
		if strings.Contains(string(data), value) {
			t.Fatal("private summary disclosure", string(data))
		}
	}
	if !strings.Contains(string(data), "Outcome: changed") {
		t.Fatal(string(data))
	}
}

func TestActionsWorkflowTargetReportRoundTrip(t *testing.T) {
	report := policyReportFixture()
	report.ActionsPolicies[0].After.TargetWorkflows = &ActionsEventPolicyTargetReport{Include: []string{"<private-workflow>"}, Exclude: []string{}}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var decoded PlanReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	valid, err := validatePlanReport(decoded, "owner/current")
	if err != nil {
		t.Fatal(err)
	}
	detailed := renderSummary(valid, RunMetadata{}, []string{"owner", "current", "workflow"})
	if !strings.Contains(detailed, "target_workflows") || !strings.Contains(detailed, "private-workflow") || strings.Contains(detailed, "<private-workflow>") {
		t.Fatal(detailed)
	}
	report.ActionsPolicies[0].After.TargetWorkflows.Include[0] = "mutated"
	if valid.comparison.ActionsPolicies[0].After().TargetWorkflows.Include[0] != "<private-workflow>" {
		t.Fatal("target report aliased")
	}
	for _, target := range []string{`null`, `{"include":null,"exclude":[]}`, `{"include":[],"exclude":null}`, `{"include":[],"exclude":[],"actor":[]}`, `{"Include":[],"exclude":[]}`, `{"include":[null],"exclude":[]}`} {
		value := `{"operation":"create","after":{"policy_name":"policy","enforcement_status":"active","restrict_events":{"allowed_events":["push"]},"restrict_actors":{"enabled":false},"target_workflows":` + target + `}}`
		var entry ActionsEventPolicyPlanChange
		if err := json.Unmarshal([]byte(value), &entry); err == nil {
			t.Fatal("accepted malformed report", value)
		}
	}
	for _, target := range []*ActionsEventPolicyTargetReport{{}, {Include: []string{}, Exclude: []string{"~ALL"}}, {Include: []string{"~ALL", "x"}, Exclude: []string{}}} {
		r := policyReportFixture()
		r.ActionsPolicies[0].After.TargetWorkflows = target
		if _, err := validatePlanReport(r, "owner/current"); err == nil {
			t.Fatal("accepted incomplete/invalid target", target)
		}
	}
}

func TestActionsActorPolicyReportRoundTrip(t *testing.T) {
	for _, actors := range [][]settings.ActionsPolicyActor{
		{}, {{Type: "User", ID: 1234567}},
		{{Type: "RepositoryRole", ID: 4}, {Type: "RepositoryRole", ID: 2}, {Type: "RepositoryRole", ID: 5}},
	} {
		policy := settings.ActionsEventPolicy{Name: "<private-actor-policy>", Enforcement: "active", Actors: &actors}
		change, err := settings.NewActionsEventPolicyChange(nil, policy)
		if err != nil {
			t.Fatal(err)
		}
		report := actionsEventPolicyReport(change.After())
		data, err := json.Marshal(ActionsEventPolicyPlanChange{Operation: "create", After: report})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `"restrict_events":{"enabled":false}`) || !strings.Contains(string(data), `"allowed_actors":[`) {
			t.Fatal(string(data))
		}
		var decoded ActionsEventPolicyPlanChange
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		changes, err := validateActionsEventPolicyReport([]ActionsEventPolicyPlanChange{decoded})
		if err != nil {
			t.Fatal(err)
		}
		if changes[0].After().Events != nil || changes[0].After().Actors == nil || len(*changes[0].After().Actors) != len(actors) {
			t.Fatalf("lost actor policy: %#v", changes[0].After())
		}
		valid, err := validatePlanReport(PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, ActionsPolicies: ActionsEventPolicyPlanChanges{decoded}}, "owner/current")
		if err != nil {
			t.Fatal(err)
		}
		detailed := renderSummary(valid, RunMetadata{}, []string{"owner", "current", "workflow"})
		if strings.Contains(detailed, "<private-actor-policy>") || !strings.Contains(detailed, "private-actor-policy") || !strings.Contains(detailed, "restrict_actors.allowed_actors") {
			t.Fatal(detailed)
		}
		if valid.comparison.Managed != 1 || len(valid.comparison.ActionsPolicies) != 1 {
			t.Fatal("actor restriction changed operation counts")
		}
		if len(actors) > 0 {
			(*report.RestrictActors.AllowedActors)[0] = ActionsEventPolicyActorReport{}
			if (*changes[0].After().Actors)[0].ID == 0 {
				t.Fatal("actor report aliases model")
			}
		}
	}
}

func TestActionsActorPolicyReportClosedValues(t *testing.T) {
	for _, restriction := range []string{
		`{}`, `null`, `{"enabled":true}`, `{"enabled":false,"allowed_actors":[]}`,
		`{"allowed_actors":null}`, `{"allowed_actors":[null]}`,
		`{"allowed_actors":[{}]}`, `{"allowed_actors":[{"user":0}]}`,
		`{"allowed_actors":[{"user":7,"role":"write"}]}`,
		`{"allowed_actors":[{"user":7,"role":""}]}`,
		`{"allowed_actors":[{"role":""}]}`, `{"allowed_actors":[{"role":"admin"}]}`,
		`{"allowed_actors":[{"User":7}]}`, `{"allowed_actors":[{"team":7}]}`,
		`{"allowed_actors":[{"role":"write","extra":true}]}`,
	} {
		data := `{"operation":"create","after":{"policy_name":"policy","enforcement_status":"active","restrict_events":{"enabled":false},"restrict_actors":` + restriction + `}}`
		var entry ActionsEventPolicyPlanChange
		if err := json.Unmarshal([]byte(data), &entry); err != nil {
			continue
		}
		if _, err := validateActionsEventPolicyReport([]ActionsEventPolicyPlanChange{entry}); err == nil {
			t.Fatal("invalid actor report accepted", data)
		}
	}
	for _, restriction := range []ActionsEventPolicyEventsReport{{}, {Enabled: reportBool(true)}, {Enabled: reportBool(false), AllowedEvents: []string{"push"}}} {
		report := policyReportFixture()
		report.ActionsPolicies[0].After.RestrictEvents = restriction
		if _, err := validatePlanReport(report, "owner/current"); err == nil {
			t.Fatal("invalid events restriction accepted", restriction)
		}
	}
	report := policyReportFixture()
	report.ActionsPolicies[0].After.RestrictActors = ActionsEventPolicyActorsReport{}
	if _, err := validatePlanReport(report, "owner/current"); err == nil {
		t.Fatal("missing actor restriction accepted")
	}
}

func TestActionsPolicyRestrictionRemovalReportRoundTrip(t *testing.T) {
	for _, removeActors := range []bool{false, true} {
		for _, changeMetadata := range []bool{false, true} {
			actors := []settings.ActionsPolicyActor{{Type: "User", ID: 41}}
			before := settings.ActionsEventPolicy{ID: 7, Name: "policy", Enforcement: "active", Events: []string{"push"}, Actors: &actors}
			declaration := before
			if removeActors {
				declaration.Actors = nil
				declaration.DisableActors = true
			} else {
				declaration.Events = nil
				declaration.DisableEvents = true
			}
			if changeMetadata {
				declaration.Name = "renamed"
				declaration.Enforcement = "disabled"
			}
			change, err := settings.NewActionsEventPolicyChange(&before, declaration)
			if err != nil {
				t.Fatal(err)
			}
			originalAfter := change.After()
			entry := ActionsEventPolicyPlanChange{Operation: "update", Before: actionsEventPolicyReport(*change.Before()), After: actionsEventPolicyReport(originalAfter)}
			data, err := json.Marshal(entry)
			if err != nil {
				t.Fatal(err)
			}
			var decoded ActionsEventPolicyPlanChange
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			changes, err := validateActionsEventPolicyReport([]ActionsEventPolicyPlanChange{decoded})
			if err != nil {
				t.Fatal(err)
			}
			after := changes[0].After()
			if (after.Actors == nil) != removeActors || (after.Events == nil) == removeActors || after.Name != originalAfter.Name || after.Enforcement != originalAfter.Enforcement || after.DisableActors || after.DisableEvents {
				t.Fatalf("removal changed during report reconstruction: %#v", after)
			}
			report := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, ActionsPolicies: ActionsEventPolicyPlanChanges{decoded}}
			if _, err := validatePlanReport(report, "owner/current"); err != nil {
				t.Fatal(err)
			}
		}
	}
}
