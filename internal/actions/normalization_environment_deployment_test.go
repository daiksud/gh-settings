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
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func environmentModeReportChange(t *testing.T, raw bool, choice uimodel.EnvironmentDeploymentChoice, partial bool) settings.EnvironmentChange {
	t.Helper()
	zero, off, reviewers := 0, false, []settings.EnvironmentReviewer{}
	before := settings.EnvironmentDeclaration{Name: "<MODE_OBJECT>", WaitTimer: &zero, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}
	after := before
	if partial {
		after = settings.EnvironmentDeclaration{Name: before.Name, DeploymentBranchPolicyDeclared: true}
	}
	policy, ok := settings.EnvironmentDeploymentPolicyForChoice(choice)
	if !ok {
		t.Fatal(choice)
	}
	after.DeploymentBranchPolicy = policy
	record := settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{ProtectedBranches: raw, CustomBranchPolicies: raw}, Fallback: uimodel.NoRestriction}
	change, err := settings.NewEnvironmentChangeWithDeploymentNormalization("update", &before, &after, record)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func TestEnvironmentDeploymentPrivateReportRetainsPairAndActualPartialRequest(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, choice := range []uimodel.EnvironmentDeploymentChoice{uimodel.NoRestriction, uimodel.ProtectedBranchesOnly, uimodel.SelectedBranchesAndTags} {
			for _, partial := range []bool{false, true} {
				change := environmentModeReportChange(t, raw, choice, partial)
				path := filepath.Join(t.TempDir(), "plan.json")
				var stdout, stderr bytes.Buffer
				status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
					return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Environments: []settings.EnvironmentChange{change}}}, nil
				})
				data, err := os.ReadFile(path)
				if err != nil || status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
					t.Fatal(status, err, stdout.String(), stderr.String())
				}
				var report actions.PlanReport
				if err := json.Unmarshal(data, &report); err != nil {
					t.Fatal(err)
				}
				if report.Outcome != "changed" || report.Managed != 1 || len(report.Environments) != 1 || report.Environments[0].DeploymentModeNormalization == nil || *report.Environments[0].DeploymentModeNormalization != change.DeploymentModeNormalization() || report.Environments[0].Before.DeploymentBranchPatterns != nil {
					t.Fatal("raw pair or effective policy lost", string(data))
				}
				if partial {
					encoded, _ := json.Marshal(report.Environments[0].After)
					var fields map[string]json.RawMessage
					json.Unmarshal(encoded, &fields)
					if len(fields) != 2 || fields["name"] == nil || fields["deployment_branch_policy"] == nil || !settings.EnvironmentHasAPIOmissions(*report.Environments[0].After) {
						t.Fatal("partial request gained defaults", string(encoded))
					}
				}
				content, err := actions.RenderSummary(report, summaryMetadata())
				if err != nil || !strings.Contains(html.UnescapeString(content), presentation.EnvironmentLine(change)) || strings.Contains(content, "<MODE_OBJECT>") || !strings.Contains(content, "effective before policy") || partial && !strings.Contains(content, "API-controlled") {
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
				if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(data), "MODE_OBJECT") || strings.Contains(string(data), settings.EnvironmentDeploymentModePath) || strings.Contains(string(data), "protected_branches") || strings.Contains(string(data), "custom_branch_policies") {
					t.Fatal("status privacy", status, string(data), err)
				}
			}
		}
	}
}

func TestEnvironmentDeploymentReportRejectsForgedAndUnmanagedPairEvidence(t *testing.T) {
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) { r.Environments[0].DeploymentModeNormalization = nil },
		func(r *actions.PlanReport) {
			*r.Environments[0].DeploymentModeNormalization = settings.EnvironmentDeploymentNormalization{}
		},
		func(r *actions.PlanReport) { r.Environments[0].DeploymentModeNormalization.Path = "other" },
		func(r *actions.PlanReport) {
			r.Environments[0].DeploymentModeNormalization.Fallback = uimodel.SelectedBranchesAndTags
		},
		func(r *actions.PlanReport) {
			r.Environments[0].DeploymentModeNormalization.RawPolicy.CustomBranchPolicies = false
		},
		func(r *actions.PlanReport) { r.Environments[0].Before.WaitTimer = nil },
		func(r *actions.PlanReport) {
			r.Environments[0].Before.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{ProtectedBranches: true}
		},
		func(r *actions.PlanReport) { r.Environments[0].After.DeploymentBranchPolicyDeclared = false },
		func(r *actions.PlanReport) { r.Environments[0].After.Name = "other" },
		func(r *actions.PlanReport) { r.Environments[0].Operation = "create"; r.Environments[0].Before = nil },
		func(r *actions.PlanReport) { r.Environments = append(r.Environments, r.Environments[0]) },
	} {
		change := environmentModeReportChange(t, true, uimodel.NoRestriction, false)
		record := change.DeploymentModeNormalization()
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Environments: []actions.EnvironmentPlanChange{{Operation: "update", Before: change.Before(), After: change.After(), DeploymentModeNormalization: &record}}}
		if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
			t.Fatal("valid pair proof rejected", err)
		}
		mutate(&report)
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
			t.Fatal("forged pair proof accepted", report)
		}
	}
	for _, raw := range []string{`null`, `{}`, `{"path":"x","raw_policy":{"protected_branches":false},"fallback":"no_restriction"}`, `{"path":"x","raw_policy":{"protected_branches":null,"custom_branch_policies":false},"fallback":"no_restriction"}`, `{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":false}}`, `{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":false,"extra":true},"fallback":"no_restriction"}`} {
		var entry actions.EnvironmentPlanChange
		if err := json.Unmarshal([]byte(`{"operation":"update","deployment_mode_normalization":`+raw+`}`), &entry); err == nil {
			t.Fatal("malformed pair metadata decoded", raw)
		}
	}
}
