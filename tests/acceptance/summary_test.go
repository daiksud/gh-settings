//go:build acceptance

package acceptance

import (
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActionChangedSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	fixture, err := startAPIFixture(planResponse, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	workflowSHA, configSHA := strings.Repeat("a", 40), strings.Repeat("b", 40)
	actionRef, checksum := strings.Repeat("d", 40), strings.Repeat("c", 64)
	workflowRef := "workflow-owner/workflows/.github/workflows/settings-main.yml@refs/heads/main"
	env := append(fixtureEnv(fixture.cfg),
		"SETTINGS_ACTOR=original-user", "SETTINGS_TRIGGERING_ACTOR=rerun-user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=2",
		"SETTINGS_WORKFLOW_REF="+workflowRef, "SETTINGS_WORKFLOW_SHA="+workflowSHA, "SETTINGS_CONFIG_SHA="+configSHA, "SETTINGS_TARGET=config-owner/config-repo",
		"SETTINGS_ACTION_REPOSITORY=action-owner/action-repo", "SETTINGS_ACTION_REF="+actionRef, "SETTINGS_CLI_VERSION=v0.6.1", "SETTINGS_CLI_SHA256="+checksum,
		"SETTINGS_TOKEN=unrelated-secret-sentinel", "UNRELATED_ENV=unrelated-secret-sentinel")
	reportPath := filepath.Join(fixture.root, "plan.json")
	report := `{"outcome":"changed","target":"config-owner/config-repo","managed":3,"changes":[{"field":"repository.description","before":"old-description","after":"new-description"},{"field":"repository.has_issues","before":true,"after":false},{"field":"topics.names","before":["old"],"after":[]}]}`
	if err := os.WriteFile(reportPath, []byte(report), 0600); err != nil {
		t.Fatal(err)
	}
	summaryPath := filepath.Join(fixture.root, "review Summary.md")
	if err := os.WriteFile(summaryPath, []byte("existing Summary"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "summary", reportPath, summaryPath}, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
		t.Fatalf("summary: %v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
	}
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	written := string(data)
	visible := html.UnescapeString(written)
	if !strings.HasPrefix(written, "existing Summary\n\n## Settings review\n") || strings.Contains(written, "unrelated-secret-sentinel") {
		t.Fatalf("Summary lost prior content or exposed unrelated data: %s", written)
	}
	for _, expected := range []string{
		"Original actor", "original-user", "Initiating / rerun actor", "rerun-user", "Event", "push", "Run / attempt", "123", "2",
		"Workflow path / ref", workflowRef, "Workflow revision", workflowSHA, "Target repository", "config-owner/config-repo", "Configuration revision", configSHA,
		"Action repository", "action-owner/action-repo", "Action ref", actionRef, "CLI version", "v0.6.1", "CLI SHA-256", checksum,
		"description", `"old-description"`, `"new-description"`, "repository.settings.general.features.issues", "true", "false", "Observed before", "desired after",
		"https://github.com/workflow-owner/workflows/actions/runs/123",
		"https://github.com/workflow-owner/workflows/blob/" + workflowSHA + "/.github/workflows/settings-main.yml",
		"https://github.com/config-owner/config-repo/blob/" + configSHA + "/.github/settings.yml",
		`repository.about.topics: ["old"] → []`, "can overwrite intervening remote changes", "does not prove publication",
	} {
		if !strings.Contains(visible, expected) {
			t.Errorf("Summary missing %q: %s", expected, visible)
		}
	}
	if err := fixture.checkRequests(true); err != nil || fixture.reads.Load() != 0 {
		t.Fatalf("requests: %v reads=%d", err, fixture.reads.Load())
	}
}
