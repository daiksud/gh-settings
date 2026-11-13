package githubactions

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootCompositeExport(t *testing.T) {
	f := newCompositeFixture(t)
	if err := os.RemoveAll(filepath.Join(f.caller, ".github")); err != nil {
		t.Fatal(err)
	}
	result := f.run(t, "export", "fixture-token", map[string]string{"GH_TOKEN": "ambient-fixture", "GITHUB_TOKEN": "ambient-fixture", "GH_DEBUG": "api"})
	want := map[string]string{"config-sha": result.sha, "target": "owner/config", "cli-version": committedCLIVersion(t), "cli-sha256": result.digest, "config-path": ".github/settings.yml"}
	if result.status != 0 || !reflect.DeepEqual(result.outputs, want) || result.calls != "export\n" || result.metadataCalls != "action metadata\n" {
		t.Fatalf("export result: %+v", result)
	}
	if result.presentationCalls != "" || result.summary != "" || len(result.remainingReports) != 0 || result.api != "GET\n" {
		t.Fatalf("export crossed its read-only local-file boundary: %+v", result)
	}
	data, err := os.ReadFile(filepath.Join(f.caller, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "CANARY-export-variable") {
		t.Fatal("export did not save the local fixture value")
	}
	for _, canary := range []string{"CANARY-export-variable", "CANARY-export-stdout", "fixture-token", "ambient-fixture"} {
		if strings.Contains(result.log+result.summary+fmt.Sprint(result.outputs), canary) {
			t.Fatalf("export exposed %s outside the local file", canary)
		}
	}
	if f.git("rev-parse", "HEAD") != f.sha || f.git("diff", "--cached", "--name-only") != "" {
		t.Fatal("export changed consumer commit or index")
	}
}

func TestRootCompositeExportRejectsUnsupportedOptionsBeforeInstall(t *testing.T) {
	for _, extra := range []map[string]string{
		{"inputs.prune": "labels"},
		{"inputs.prune": `$(touch "$GITHUB_WORKSPACE/injected")`},
		{"inputs.plan-reporting": "status-only"},
	} {
		result := runComposite(t, "export", "fixture-token", extra)
		if result.status == 0 || result.downloaded || result.calls != "" || result.injected || len(result.outputs) != 0 {
			t.Fatalf("export option reached installation: %+v", result)
		}
	}
}

func TestRootCompositeExportFailureGates(t *testing.T) {
	for _, test := range []struct {
		name, calls string
		extra       map[string]string
	}{
		{"installation", "", map[string]string{"RUNNER_OS": "Windows"}},
		{"metadata", "", map[string]string{"MOCK_METADATA_STATUS": "37"}},
		{"command", "export\n", map[string]string{"MOCK_REMOTE_STATUS": "37"}},
		{"outputs after save", "export\n", map[string]string{"OUTPUT_DIRECTORY": "1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runComposite(t, "export", "fixture-token", test.extra)
			if result.status == 0 || len(result.outputs) != 0 || result.calls != test.calls || result.summary != "" || result.presentationCalls != "" {
				t.Fatalf("export failure gate: %+v", result)
			}
			if test.name == "command" && (result.status != 37 || !strings.Contains(result.log, "fixture command error")) {
				t.Fatalf("export lost original status or stderr: %+v", result)
			}
		})
	}
}
