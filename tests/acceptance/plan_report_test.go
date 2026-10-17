//go:build acceptance

package acceptance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionPlanChangedReport(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startAPIFixture(planResponse, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := repositoryInput("description: after\nhas_issues: false\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(fixture.root, "plan.json")
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", reportPath}, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
		t.Fatalf("plan report: %v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Outcome string `json:"outcome"`
		Target  string `json:"target"`
		Managed int    `json:"managed"`
		Changes []struct {
			Field  string `json:"field"`
			Before any    `json:"before"`
			After  any    `json:"after"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "changed" || report.Target != "owner/current" || report.Managed != 2 || len(report.Changes) != 2 {
		t.Fatalf("report=%+v", report)
	}
	if got := report.Changes[0]; got.Field != "repository.description" || got.Before != "old" || got.After != "after" {
		t.Fatalf("string change=%+v", got)
	}
	if got := report.Changes[1]; got.Field != "repository.has_issues" || got.Before != true || got.After != false {
		t.Fatalf("boolean change=%+v", got)
	}
	if err := fixture.checkRequests(false); err != nil || fixture.reads.Load() != 1 {
		t.Fatalf("requests: %v reads=%d", err, fixture.reads.Load())
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(reportPath); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("report permissions: info=%v error=%v", info, err)
	}
}

func TestActionPlanReadFailureReport(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startAPIFixture(`{"message":"Authorization: Bearer secret-sentinel"}`, 403)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "PRIVATE_ENV=secret-sentinel")
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := repositoryInput("has_issues: false\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(fixture.root, "plan.json")
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", reportPath}, false)
	if err != nil {
		t.Fatal(err)
	}
	if exit, ok := result.runErr.(*exec.ExitError); !ok || exit.ExitCode() != 1 || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "403") {
		t.Fatalf("plan report: %v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report["outcome"] != "read-failure" || report["target"] != "owner/current" || report["diagnostic"] != "repository read failed" {
		t.Fatalf("report=%v", report)
	}
	for _, sensitive := range []string{"secret-sentinel", "fixture-only-token", "Authorization", "PRIVATE_ENV"} {
		if strings.Contains(string(data), sensitive) {
			t.Fatalf("report contains %q: %s", sensitive, data)
		}
	}
	if err := fixture.checkRequests(false); err != nil || fixture.reads.Load() != 1 {
		t.Fatalf("requests: %v reads=%d", err, fixture.reads.Load())
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
}

func TestActionPlanInvalidConfigReport(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startAPIFixture(planResponse, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := fixtureEnv(fixture.cfg)
	if err := initGitCheckout(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := repositoryInput("description: [secret-sentinel\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(fixture.root, "plan.json")
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", reportPath}, false)
	if err != nil {
		t.Fatal(err)
	}
	if exit, ok := result.runErr.(*exec.ExitError); !ok || exit.ExitCode() != 1 || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "validate .github/settings.yml") {
		t.Fatalf("plan report: %v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report["outcome"] != "invalid-config" || report["diagnostic"] != "configuration validation failed" || strings.Contains(string(data), "secret-sentinel") {
		t.Fatalf("report=%v", report)
	}
	if err := fixture.checkRequests(true); err != nil || fixture.reads.Load() != 0 {
		t.Fatalf("requests: %v reads=%d", err, fixture.reads.Load())
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
}

func TestActionPlanUnmanagedReport(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startAPIFixture(planResponse, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := fixtureEnv(fixture.cfg)
	if err := initGitCheckout(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := "schemaVersion: 1\n"
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(fixture.root, "plan.json")
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", reportPath}, false)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	changes, ok := report["changes"].([]any)
	if !ok {
		t.Fatalf("changes is not an array: %v", report)
	}
	if report["outcome"] != "unmanaged" || report["managed"] != float64(0) || len(changes) != 0 {
		t.Fatalf("report=%v", report)
	}
	if _, present := report["target"]; present {
		t.Fatalf("unexpected target in local no-op: %v", report)
	}
	if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
		t.Fatalf("plan report: %v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
	}
	if err := fixture.checkRequests(true); err != nil || fixture.reads.Load() != 0 {
		t.Fatalf("requests: %v reads=%d", err, fixture.reads.Load())
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
}
