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
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestCommitMessageContextBindsAnUnchangedCompanion(t *testing.T) {
	context := settings.CommitMessageContext{Menu: settings.SquashCommitMenu, Before: settings.CommitMessagePair{Title: "COMMIT_OR_PR_TITLE", Message: "COMMIT_MESSAGES"}, After: settings.CommitMessagePair{Title: "PR_TITLE", Message: "COMMIT_MESSAGES"}}
	changed := []PlanChange{{Field: "repository.squash_merge_commit_title", Before: "COMMIT_OR_PR_TITLE", After: "PR_TITLE"}}
	if err := validateCommitMessageContexts([]settings.CommitMessageContext{context}, changed); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		contexts []settings.CommitMessageContext
		changes  []PlanChange
	}{
		{"missing context", nil, changed},
		{"duplicate", []settings.CommitMessageContext{context, context}, changed},
		{"missing changed component", []settings.CommitMessageContext{context}, nil},
		{"mismatched component", []settings.CommitMessageContext{context}, []PlanChange{{Field: "repository.squash_merge_commit_title", Before: "PR_TITLE", After: "COMMIT_OR_PR_TITLE"}}},
		{"forged unchanged component", []settings.CommitMessageContext{context}, append(append([]PlanChange{}, changed...), PlanChange{Field: "repository.squash_merge_commit_message", Before: "COMMIT_MESSAGES", After: "COMMIT_MESSAGES"})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCommitMessageContexts(tc.contexts, tc.changes); err == nil {
				t.Fatal("accepted inconsistent context")
			}
		})
	}
}

func TestActionsExecutionContextKeepsDisabledLatentValuePrivate(t *testing.T) {
	context := &settings.ActionsExecutionContext{Before: uimodel.DisableActions, After: uimodel.AllowOwnerActionsAndReusableWorkflows, ObservedAllowed: "all"}
	changes := []PlanChange{{Field: "actions.enabled", Before: false, After: true}, {Field: "actions.allowed_actions", Before: "all", After: "local_only"}}
	if err := validateActionsExecutionContext(context, changes, nil); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		context *settings.ActionsExecutionContext
		changes []PlanChange
	}{
		{"missing context", nil, changes},
		{"missing enabled", context, changes[1:]},
		{"missing changed allowed companion", context, changes[:1]},
		{"forged desired", context, []PlanChange{{Field: "actions.enabled", Before: false, After: true}, {Field: "actions.allowed_actions", Before: "all", After: "selected"}}},
		{"unreferenced", context, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateActionsExecutionContext(tc.context, tc.changes, nil); err == nil {
				t.Fatal("accepted inconsistent context")
			}
		})
	}
}

func TestFailureResourceUsesOnlyFixedPaths(t *testing.T) {
	if validFailureResource("read-failure", "repository.settings.pages secret") || validFailureResource("changed", "repository.settings.pages") || !validFailureResource("read-failure", "repository.settings.pages") {
		t.Fatal("failure resource boundary not enforced")
	}
}

func TestFailedReportShowsOnlyFixedResourceInDetailedSummary(t *testing.T) {
	report := PlanReport{Outcome: "read-failure", Target: "owner/current", Managed: 0, Changes: []PlanChange{}, Diagnostic: readFailureDiagnostic, Resource: "repository.settings.pages"}
	valid, err := validatePlanReport(report, "owner/current")
	if err != nil {
		t.Fatal(err)
	}
	detailed := renderSummary(valid, RunMetadata{Target: "owner/current"}, []string{"owner", "repo", ".github/workflows/test.yml"})
	if !strings.Contains(detailed, "repository.settings.pages: repository read failed") {
		t.Fatal(detailed)
	}
	path, summary := filepath.Join(t.TempDir(), "report.json"), filepath.Join(t.TempDir(), "summary.md")
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	if err := writeSummaryMode(path, summary, "status-only", nil); err == nil {
		t.Fatal("failed plan must retain failure status")
	}
	output, err := os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(output), "repository.settings.pages") || strings.Contains(string(output), readFailureDiagnostic) {
		t.Fatal(string(output))
	}
}

func TestPlanRejectsUnrepresentableSelectedRuleset(t *testing.T) {
	const beforeJSON = `{"id":7,"name":"policy","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[{"actor_type":"OrganizationAdmin","bypass_mode":"always"}],"rules":[]}`
	const afterJSON = `{"id":7,"name":"policy","target":"branch","enforcement":"disabled","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[]}`
	var before, after settings.RulesetDeclaration
	if err := json.Unmarshal([]byte(beforeJSON), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(afterJSON), &after); err != nil {
		t.Fatal(err)
	}
	change, err := settings.NewRulesetChange("update", 7, &before, &after)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "report.json")
	err = writePlanReport(path, func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Rulesets: []settings.RulesetChange{change}}}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "repository.settings.rulesets") {
		t.Fatal(err)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "read-failure" || report.Resource != "repository.settings.rulesets" || len(report.Rulesets) != 0 || strings.Contains(string(data), "OrganizationAdmin") {
		t.Fatal(string(data))
	}
}
