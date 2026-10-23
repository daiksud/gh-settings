package githubactions

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func releaseRequestFixture(t *testing.T) workflow {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release-request.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestReleaseRequestWorkflowTrustBoundary(t *testing.T) {
	w := releaseRequestFixture(t)
	var events struct {
		Types []string `yaml:"types"`
	}
	trigger := w.On["issue_comment"]
	if len(w.On) != 1 || trigger.Decode(&events) != nil || !reflect.DeepEqual(events.Types, []string{"created"}) ||
		!reflect.DeepEqual(w.Permissions, map[string]string{}) || len(w.Jobs) != 2 || w.Concurrency.Cancel ||
		w.Concurrency.Group != "release-request-${{ github.repository }}-${{ github.event.comment.id }}" {
		t.Fatal("unexpected release event, concurrency, or default permission scope")
	}
	authorize, dispatch := w.Jobs["authorize"], w.Jobs["dispatch"]
	gate := strings.Join(strings.Fields(authorize.If), " ")
	want := "github.repository == 'daiksud/gh-settings' && " +
		"github.ref == 'refs/heads/main' && " +
		"github.event.issue.number == 410 && " +
		"!github.event.issue.pull_request && " +
		"github.event.issue.state == 'open' && " +
		"github.event.comment.user.id == github.event.repository.owner.id && " +
		"github.actor == github.repository_owner && " +
		"github.triggering_actor == github.repository_owner"
	if gate != want || authorize.Needs != "" || authorize.Timeout != 3 || authorize.Runner != "ubuntu-slim" ||
		!reflect.DeepEqual(authorize.Permissions, map[string]string{"contents": "read"}) ||
		!reflect.DeepEqual(authorize.Outputs, map[string]string{"source-sha": "${{ steps.command.outputs.source-sha }}"}) ||
		len(authorize.Steps) != 1 || authorize.Steps[0].ID != "command" || authorize.Steps[0].Shell != "bash" ||
		!reflect.DeepEqual(authorize.Steps[0].Env, map[string]string{"RELEASE_COMMENT": "${{ github.event.comment.body }}"}) {
		t.Fatal("owner/Issue authorization or unprivileged parser boundary changed")
	}
	if dispatch.Needs != "authorize" || dispatch.If != "" || dispatch.Continue || dispatch.Timeout != 3 || dispatch.Runner != "ubuntu-slim" ||
		!reflect.DeepEqual(dispatch.Permissions, map[string]string{"contents": "read", "actions": "write"}) ||
		len(dispatch.Steps) != 1 || dispatch.Steps[0].Shell != "bash" ||
		!reflect.DeepEqual(dispatch.Steps[0].Env, map[string]string{
			"GH_TOKEN": "${{ github.token }}", "SOURCE_SHA": "${{ needs.authorize.outputs.source-sha }}",
		}) {
		t.Fatal("privileged dispatch must be gated and bound to one validated SHA")
	}
	if strings.Contains(authorize.Steps[0].Run, "gh ") || strings.Contains(dispatch.Steps[0].Run, "git ") ||
		strings.Contains(dispatch.Steps[0].Run, "uses:") {
		t.Fatal("release request must never execute candidate source or acquire credentials before authorization")
	}
	// Keep the existing release pipeline as the sole publication authority.
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var release struct {
		On map[string]yaml.Node `yaml:"on"`
	}
	if err := yaml.Unmarshal(data, &release); err != nil {
		t.Fatal(err)
	}
	if len(release.On) != 1 || release.On["workflow_dispatch"].Kind != yaml.MappingNode {
		t.Fatal("release pipeline must retain its manual-only dispatch trigger")
	}
}

func runReleaseRequestStep(t *testing.T, script string, env map[string]string) (string, string, error) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	mock := `#!/bin/bash
set -euo pipefail
printf '%s\n' "$*" >> "$CALLS"
if [[ "$1" == api ]]; then
  [[ "$2" == 'repos/daiksud/gh-settings/git/ref/heads/main' && "$3" == --jq && "$4" == .object.sha ]] || exit 79
  [[ "$API_STATUS" == 0 ]] || exit "$API_STATUS"
  printf '%s\n' "$CURRENT_MAIN"
elif [[ "$1" == workflow ]]; then
  [[ "$2" == run && "$3" == release.yml && "$4" == --repo && "$5" == daiksud/gh-settings && "$6" == --ref && "$7" == main && "$8" == -f && "$9" == source_sha=* ]] || exit 78
  exit "$DISPATCH_STATUS"
else
  exit 77
fi
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(mock), 0700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "output")
	calls := filepath.Join(dir, "calls")
	for _, p := range []string{output, calls} {
		if err := os.WriteFile(p, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("bash", "-e", "-o", "pipefail", "-c", script)
	command.Dir = dir
	command.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_OUTPUT="+output, "CALLS="+calls, "GITHUB_REPOSITORY=daiksud/gh-settings",
		"API_STATUS=0", "DISPATCH_STATUS=0",
	)
	for k, v := range env {
		command.Env = append(command.Env, k+"="+v)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if stdout.Len() != 0 {
		t.Fatalf("unexpected public stdout: %q", stdout.String())
	}
	data, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	logs, readErr := os.ReadFile(calls)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(data), string(logs), err
}

func TestReleaseRequestCommandRejectsUntrustedText(t *testing.T) {
	step := releaseRequestFixture(t).Jobs["authorize"].Steps[0]
	sha := strings.Repeat("a", 40)
	for _, tt := range []struct {
		name, comment string
		valid         bool
	}{
		{"valid", "/release " + sha, true},
		{"missing", "", false},
		{"prefix", "/release", false},
		{"bare SHA", sha, false},
		{"short", "/release " + sha[:39], false},
		{"nonhex", "/release " + strings.Repeat("g", 40), false},
		{"uppercase", "/release " + strings.Repeat("A", 40), false},
		{"trailing newline", "/release " + sha + "\n", false},
		{"leading space", " /release " + sha, false},
		{"extra argument", "/release " + sha + " --ref attacker", false},
		{"shell text", "/release " + sha + "$(touch unexpected)", false},
		{"second line", "/release " + sha + "\n/release " + sha, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output, calls, err := runReleaseRequestStep(t, step.Run, map[string]string{"RELEASE_COMMENT": tt.comment})
			if tt.valid {
				if err != nil || output != "source-sha="+sha+"\n" || calls != "" {
					t.Fatalf("valid request rejected: output=%q calls=%q err=%v", output, calls, err)
				}
			} else if err == nil || output != "" || calls != "" {
				t.Fatalf("unauthorized input reached output/API: output=%q calls=%q err=%v", output, calls, err)
			}
		})
	}
}

func TestReleaseRequestDispatchPinsMainAndExistingWorkflow(t *testing.T) {
	step := releaseRequestFixture(t).Jobs["dispatch"].Steps[0]
	sha := strings.Repeat("a", 40)
	for _, tt := range []struct {
		name, source, current, apiStatus, dispatchStatus string
		valid, callsAPI, dispatches                      bool
	}{
		{"exact SHA", sha, sha, "0", "0", true, true, true},
		{"main moved", sha, strings.Repeat("b", 40), "0", "0", false, true, false},
		{"malformed main", sha, "main", "0", "0", false, true, false},
		{"no source", "", sha, "0", "0", false, false, false},
		{"nonhex source", strings.Repeat("z", 40), sha, "0", "0", false, false, false},
		{"API error", sha, sha, "7", "0", false, true, false},
		{"dispatch error", sha, sha, "0", "8", false, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, calls, err := runReleaseRequestStep(t, step.Run, map[string]string{
				"SOURCE_SHA": tt.source, "CURRENT_MAIN": tt.current, "API_STATUS": tt.apiStatus, "DISPATCH_STATUS": tt.dispatchStatus,
			})
			if (err == nil) != tt.valid {
				t.Fatalf("unexpected request status: %v (%s)", err, calls)
			}
			want := ""
			if tt.callsAPI {
				want = "api repos/daiksud/gh-settings/git/ref/heads/main --jq .object.sha\n"
			}
			if tt.dispatches {
				want += "workflow run release.yml --repo daiksud/gh-settings --ref main -f source_sha=" + sha + "\n"
			}
			if calls != want {
				t.Fatalf("unexpected dispatched target or arguments: got %q, want %q", calls, want)
			}
		})
	}
}
