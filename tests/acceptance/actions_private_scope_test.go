//go:build acceptance

package acceptance

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPrivateActionsControlsFailBeforeRemoteOrFileWork(t *testing.T) {
	binary := acceptanceBinary(t)
	var calls atomic.Int32
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	for _, control := range []string{"access: not_accessible", "access: accessible_from_repositories_owned_by_user", "access: accessible_from_repositories_in_organization", "fork_pull_request_workflows: {}", "fork_pull_request_workflows: {run_workflows_from_fork_pull_requests: false}", "fork_pull_request_workflows: {send_write_tokens_to_workflows_from_fork_pull_requests: false}"} {
		content := "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {general: {" + control + "}}}}\n"
		if err := writeSettingsFile(fixture.work, content); err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"validate", "plan", "apply"} {
			out, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || out.runErr == nil || !strings.Contains(withoutExportInfo(out.stderr.String()), "unknown repository.settings.actions.general key") {
				t.Fatal(command, err, out.stderr.String())
			}
		}
		saved, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
		if err != nil || string(saved) != content {
			t.Fatal("changed unsupported declaration", err)
		}
	}
	for _, flag := range []string{"--actions-access", "--private-fork-workflows"} {
		out, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export", flag}, false)
		if err != nil || out.runErr == nil || !strings.Contains(withoutExportInfo(out.stderr.String()), "unsupported command") {
			t.Fatal(flag, err, out.stderr.String())
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unsupported control reached API", calls.Load())
	}
}
