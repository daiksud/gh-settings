package actions_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func TestPrivatePlanHidesErrors(t *testing.T) {
	const secret = "CANARY-credential-policy-value"
	for _, stage := range []string{"", "invalid-config", "read-failure"} {
		t.Run(stage, func(t *testing.T) {
			for _, storageFailure := range []bool{false, true} {
				path := filepath.Join(t.TempDir(), "plan.json")
				if storageFailure {
					path = filepath.Join(t.TempDir(), secret, "plan.json")
				}
				var output, diagnostic bytes.Buffer
				status := actions.Run([]string{"plan", path, "status-only"}, &output, &diagnostic, "dev", func() (app.PlanResult, error) {
					err := errors.New(secret)
					if stage != "" {
						return app.PlanResult{}, &app.PlanError{Stage: stage, Err: err}
					}
					return app.PlanResult{}, err
				})
				if status != 1 || output.Len() != 0 || diagnostic.String() != "gh settings action: private plan reporting failed\n" {
					t.Fatalf("private failure escaped: %d %q %q", status, output.String(), diagnostic.String())
				}
			}
		})
	}
}

func TestDetailedPathNamedStatusOnlyKeepsDiagnostics(t *testing.T) {
	t.Chdir(t.TempDir())
	var output, diagnostic bytes.Buffer
	status := actions.Run([]string{"plan", "status-only"}, &output, &diagnostic, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{}, errors.New("detailed diagnostic retained")
	})
	if status != 1 || !strings.Contains(diagnostic.String(), "detailed diagnostic retained") {
		t.Fatal(status, diagnostic.String())
	}
	diagnostic.Reset()
	status = actions.Run([]string{"summary", "missing.json", "status-only"}, &output, &diagnostic, "dev", nil)
	if status != 1 || !strings.Contains(diagnostic.String(), "read plan report") {
		t.Fatal(status, diagnostic.String())
	}
}

func TestPrivateSummaryHidesCompleteRulesetPolicy(t *testing.T) {
	d, err := workspace.DecodeConfig([]byte(`schemaVersion: 1
settings: {rulesets: [{name: 'CANARY-hidden-policy', target: branch, enforcement: active, conditions: {ref_name: {include: ['~ALL'], exclude: []}}, bypass_actors: [{actor_type: Integration, actor_id: 987654321, bypass_mode: always}], rules: [{type: deletion}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	before := d.Rulesets.Values()[0]
	change, err := settings.NewRulesetChange("delete", 7, &before, nil)
	if err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(t.TempDir(), "plan.json")
	var output, diagnostic bytes.Buffer
	status := actions.Run([]string{"plan", report, "status-only"}, &output, &diagnostic, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Rulesets: []settings.RulesetChange{change}}}, nil
	})
	if status != 0 {
		t.Fatal(diagnostic.String())
	}
	info, err := os.Stat(report)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("report permissions", err)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	t.Setenv("SETTINGS_ACTOR", "CANARY-private-actor")
	summary := filepath.Join(t.TempDir(), "summary")
	status = actions.Run([]string{"summary", report, summary, "status-only"}, &output, &diagnostic, "dev", nil)
	data, err := os.ReadFile(summary)
	if err != nil || status != 0 || output.String() != "plan-outcome=changed\n" || diagnostic.Len() != 0 {
		t.Fatal(status, err, output.String(), diagnostic.String())
	}
	if strings.Contains(string(data), "CANARY") || strings.Contains(string(data), "987654321") || !strings.Contains(string(data), "Outcome: changed") {
		t.Fatal("private summary exposed policy", string(data))
	}
}

func TestPrivateSummaryRejectsUnusableReportsWithoutDisclosure(t *testing.T) {
	for _, body := range []string{"", `{"CANARY":`, `{"outcome":"CANARY","managed":0,"changes":[]}`, `{"outcome":"unmanaged","changes":[]}`} {
		var output, diagnostic bytes.Buffer
		report := filepath.Join(t.TempDir(), "CANARY.json")
		if body != "" {
			if err := os.WriteFile(report, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
		}
		status := actions.Run([]string{"summary", report, filepath.Join(t.TempDir(), "summary"), "status-only"}, &output, &diagnostic, "dev", nil)
		if status != 1 || output.Len() != 0 || strings.Contains(diagnostic.String(), "CANARY") {
			t.Fatalf("leaked invalid report: %d %q %q", status, output.String(), diagnostic.String())
		}
	}
	for _, failure := range []string{"summary destination", "outcome output"} {
		report := filepath.Join(t.TempDir(), "plan.json")
		data, _ := json.Marshal(actions.PlanReport{Outcome: "unmanaged", Changes: []actions.PlanChange{}})
		if err := os.WriteFile(report, data, 0600); err != nil {
			t.Fatal(err)
		}
		var diagnostic bytes.Buffer
		summary := t.TempDir()
		var output bytes.Buffer
		if failure == "outcome output" {
			summary = filepath.Join(t.TempDir(), "summary")
		}
		writer := interface{ Write([]byte) (int, error) }(&output)
		if failure == "outcome output" {
			writer = failingWriter{}
		}
		status := actions.Run([]string{"summary", report, summary, "status-only"}, writer, &diagnostic, "dev", nil)
		if status != 1 || strings.Contains(diagnostic.String(), "output unavailable") {
			t.Fatal(failure, status, diagnostic.String())
		}
	}
}
