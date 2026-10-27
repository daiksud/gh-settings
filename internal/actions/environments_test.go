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

func TestEnvironmentPlanReportSummaryAndClosedBoundary(t *testing.T) {
	zero, prevent, reviewers := 0, false, []settings.EnvironmentReviewer{}
	d, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "<production>", WaitTimer: &zero, PreventSelfReview: &prevent, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}})
	if err != nil {
		t.Fatal(err)
	}
	after := d.Values()[0]
	before := d.Values()[0]
	timer := 30
	before.WaitTimer = &timer
	change, err := settings.NewEnvironmentChange("update", &before, &after)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Environments: []settings.EnvironmentChange{change}}}, nil
	})
	data, err := os.ReadFile(path)
	if err != nil || status != 0 {
		t.Fatal(status, err, stderr.String())
	}
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "changed" || len(report.Environments) != 1 {
		t.Fatal(string(data))
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || strings.Contains(content, "<production>") {
		t.Fatal(content, err)
	}
	for _, want := range []string{`settings.environments: update`, `"wait_timer":30`, `"wait_timer":0`, `"prevent_self_review":false`, `"reviewers":[]`, `"deployment_branch_policy":null`} {
		if !strings.Contains(html.UnescapeString(content), want) {
			t.Fatal("missing default-derived change", want, content)
		}
	}
	for _, mutate := range []func(*actions.PlanReport){func(r *actions.PlanReport) { r.Managed = 0 }, func(r *actions.PlanReport) { r.Outcome = "no-difference" }, func(r *actions.PlanReport) { r.Environments = append(r.Environments, r.Environments[0]) }, func(r *actions.PlanReport) { r.Environments[0].Operation = "delete" }} {
		var invalid actions.PlanReport
		if err := json.Unmarshal(data, &invalid); err != nil {
			t.Fatal(err)
		}
		mutate(&invalid)
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatal("invalid report accepted", invalid)
		}
	}
	for _, body := range []string{
		strings.Replace(string(data), `"wait_timer":0`, `"wait_timer":null`, 1),
		strings.Replace(string(data), `"wait_timer":0`, `"Wait_timer":0`, 1),
		strings.Replace(string(data), `"wait_timer":0`, `"id":71,"wait_timer":0`, 1),
		strings.Replace(string(data), `"deployment_branch_policy":null`, `"unsupported":false`, 1),
		strings.Replace(string(data), `"prevent_self_review":false,`, "", 1),
	} {
		var invalid actions.PlanReport
		decodeErr := json.Unmarshal([]byte(body), &invalid)
		if decodeErr == nil {
			if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
				t.Fatal("incomplete/noncanonical environment report accepted", body)
			}
		}
	}
}

func TestEnvironmentPartialRequestReportDoesNotInventAPIOutcome(t *testing.T) {
	timer, prevent, reviewers := 30, false, []settings.EnvironmentReviewer{}
	before := settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer, PreventSelfReview: &prevent, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}
	after := settings.EnvironmentDeclaration{Name: "production"}
	change, err := settings.NewEnvironmentChange("update", &before, &after)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Environments: []settings.EnvironmentChange{change}}}, nil
	})
	data, err := os.ReadFile(path)
	if err != nil || status != 0 {
		t.Fatal(status, err, stderr.String())
	}
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(content, "API-controlled") || !strings.Contains(content, "unknown until apply") {
		t.Fatal(content, err)
	}
	var raw map[string]json.RawMessage
	afterJSON, _ := json.Marshal(report.Environments[0].After)
	json.Unmarshal(afterJSON, &raw)
	if len(raw) != 1 || raw["name"] == nil {
		t.Fatal("fabricated after state", string(afterJSON))
	}
}
