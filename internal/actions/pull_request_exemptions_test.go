package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPullRequestExemptionsReportValidation(t *testing.T) {
	good := PullRequestExemptionsPlanChange{Before: []string{"alice"}, After: []string{}}
	if got, err := validatePullRequestExemptionsReport([]PullRequestExemptionsPlanChange{good}); err != nil || len(got) != 1 {
		t.Fatal(got, err)
	}
	for _, raw := range [][]PullRequestExemptionsPlanChange{{good, good}, {{Before: nil, After: []string{}}}, {{Before: []string{}, After: nil}}, {{Before: []string{}, After: []string{"alice", "Alice"}}}, {{Before: []string{"alice"}, After: []string{"ALICE"}}}, {{Before: []string{}, After: []string{"<script>"}}}} {
		if _, err := validatePullRequestExemptionsReport(raw); err == nil {
			t.Fatal(raw)
		}
	}
	for _, raw := range []string{`{"before":[],"after":null}`, `{"before":[],"after":[] ,"unknown":true}`, `{"after":[]}`, `{"before":[],"after":[1]}`} {
		var change PullRequestExemptionsPlanChange
		if err := json.Unmarshal([]byte(raw), &change); err == nil {
			t.Fatal(raw)
		}
	}
	report := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, PullRequestExemptions: []PullRequestExemptionsPlanChange{good}}
	if _, err := validatePlanReport(report, "owner/current"); err != nil {
		t.Fatal(err)
	}
	report.Managed = 0
	if _, err := validatePlanReport(report, "owner/current"); err == nil {
		t.Fatal("missing managed count accepted")
	}
}
func TestPullRequestExemptionsReportPrivacy(t *testing.T) {
	change, _ := settings.NewPullRequestExemptionsChange([]string{}, []string{"selected-user"})
	path := filepath.Join(t.TempDir(), "report.json")
	if err := writePlanReport(path, func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, PullRequestExemptions: []settings.PullRequestExemptionsChange{change}}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	valid, err := validatePlanReport(report, "owner/current")
	if err != nil {
		t.Fatal(err)
	}
	summary := renderSummary(valid, RunMetadata{Target: "owner/current"}, []string{"owner", "current", ".github/workflows/settings.yml"})
	if !strings.Contains(summary, settings.PullRequestExemptionsPath) || !strings.Contains(summary, "&#34;selected-user&#34;") {
		t.Fatal(summary)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	status := filepath.Join(t.TempDir(), "status.md")
	var output bytes.Buffer
	if err := writeSummaryMode(path, status, "status-only", &output); err != nil {
		t.Fatal(err)
	}
	private, err := os.ReadFile(status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(private)+output.String(), "selected-user") {
		t.Fatal("status-only leaked login")
	}
	failure := &app.PlanError{Stage: "read-failure", Path: settings.PullRequestExemptionsPath, Target: repository.Repository{Owner: "owner", Name: "current"}, Err: errors.New("selected-user secret")}
	if err := writePlanReport(path, func() (app.PlanResult, error) { return app.PlanResult{}, failure }); err == nil {
		t.Fatal("failure lost")
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "selected-user") || strings.Contains(string(data), "secret") {
		t.Fatal(string(data))
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Resource != settings.PullRequestExemptionsPath {
		t.Fatal(report.Resource)
	}
}
