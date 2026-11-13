package actions_test

import (
	"bytes"
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassicProtectionReportSummaryAndPrivacy(t *testing.T) {
	id, yes, no := int64(7), true, false
	before := settings.ClassicRule{ID: &id, BranchNamePattern: "<private>/*", RequireConversationResolutionBeforeMerging: &no, RequireSignedCommits: &no, RequireLinearHistory: &no, AllowDeletions: &no}
	after := settings.ClassicRule{ID: &id, BranchNamePattern: before.BranchNamePattern, RequireSignedCommits: &yes}
	plan := func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, BranchProtectionRules: []settings.ClassicChange{{Before: &before, After: after}}}}, nil
	}
	path := filepath.Join(t.TempDir(), "report.json")
	var out, errout bytes.Buffer
	if status := actions.Run([]string{"plan", path}, &out, &errout, "dev", plan); status != 0 {
		t.Fatal(status, errout.String())
	}
	data, _ := os.ReadFile(path)
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	summary, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || strings.Contains(summary, "<private>") || !strings.Contains(summary, "require_signed_commits: false") || strings.Contains(summary, "allow_deletions") {
		t.Fatal(summary, err)
	}
	for _, bad := range []string{strings.Replace(string(data), `"require_signed_commits":true`, `"require_signed_commits":null`, 1), strings.Replace(string(data), `"branch_name_pattern":`, `"Branch_name_pattern":`, 1), strings.Replace(string(data), `"require_signed_commits":true`, `"unknown":true`, 1)} {
		var r actions.PlanReport
		if json.Unmarshal([]byte(bad), &r) == nil {
			t.Fatal("invalid report decoded", bad)
		}
	}
	for _, mutate := range []func(*actions.PlanReport){func(r *actions.PlanReport) { r.Managed = 0 }, func(r *actions.PlanReport) {
		r.BranchProtectionRules = append(r.BranchProtectionRules, r.BranchProtectionRules[0])
	}, func(r *actions.PlanReport) { r.BranchProtectionRules[0].After.RequireSignedCommits = &no }, func(r *actions.PlanReport) { r.BranchProtectionRules[0].Before.ID = nil }} {
		var r actions.PlanReport
		_ = json.Unmarshal(data, &r)
		mutate(&r)
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatal("invalid report accepted", r)
		}
	}
	if status := actions.Run([]string{"plan", path, "status-only"}, &out, &errout, "dev", plan); status != 0 {
		t.Fatal(status, errout.String())
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	t.Setenv("SETTINGS_ACTOR", "private-actor")
	privatePath := filepath.Join(t.TempDir(), "summary")
	if status := actions.Run([]string{"summary", path, privatePath, "status-only"}, &out, &errout, "dev", nil); status != 0 {
		t.Fatal(status, errout.String())
	}
	private, _ := os.ReadFile(privatePath)
	if strings.Contains(string(private), "private") || strings.Contains(string(private), "signed_commits") {
		t.Fatal(string(private))
	}
}
