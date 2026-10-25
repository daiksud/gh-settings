package githubactions

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

const codeEventCondition = "github.event.action != 'edited' || github.event.changes.base != null"

func readCodeGate(t *testing.T) map[string]yaml.Node {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/ci-pr.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs        map[string]yaml.Node `yaml:"jobs"`
		Concurrency struct {
			Group  string `yaml:"group"`
			Cancel bool   `yaml:"cancel-in-progress"`
		} `yaml:"concurrency"`
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}
	if w.Concurrency.Group != "ci-pr-${{ github.event.pull_request.number }}-${{ github.event.action == 'edited' && github.event.changes.base == null && 'metadata' || 'code' }}" || !w.Concurrency.Cancel {
		t.Fatal("metadata edits must not cancel the code validation group")
	}
	return w.Jobs
}

func TestCodeGatePreservesResultsAcrossMetadataEdits(t *testing.T) {
	jobs := readCodeGate(t)
	if _, exists := jobs["pass"]; exists {
		t.Fatal("a skipped native pass job could replace a failed required code result")
	}
	if _, exists := jobs["commitlint"]; exists {
		t.Fatal("message validation belongs to the independent required workflow")
	}
	var start, draft job
	startNode, draftNode := jobs["code-start"], jobs["draft"]
	if err := startNode.Decode(&start); err != nil {
		t.Fatal(err)
	}
	if err := draftNode.Decode(&draft); err != nil {
		t.Fatal(err)
	}
	if start.If != codeEventCondition || draft.Needs != "code-start" || len(start.Steps) != 1 || !reflect.DeepEqual(start.Permissions, map[string]string{"statuses": "write"}) {
		t.Fatal("code-event guard must precede status publication and Draft/Ready transitions")
	}
	if start.Steps[0].Uses != "" || !strings.Contains(start.Steps[0].Run, "-f state=pending -f context=pass") {
		t.Fatal("start must publish pending without checking out candidate code")
	}
	var result struct {
		Needs       []string          `yaml:"needs"`
		If          string            `yaml:"if"`
		Permissions map[string]string `yaml:"permissions"`
		Steps       []step            `yaml:"steps"`
	}
	node := jobs["code-result"]
	if err := node.Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Needs, []string{"code-start", "draft", "resolve", "commit-stage", "ready", "acceptance-stage"}) || result.If != "always() && !cancelled() && ("+codeEventCondition+")" || !reflect.DeepEqual(result.Permissions, map[string]string{"statuses": "write"}) || len(result.Steps) != 1 {
		t.Fatal("code result must aggregate every prerequisite only for non-cancelled code/base events")
	}
	if result.Steps[0].Uses != "" || result.Steps[0].Continue || result.Steps[0].Env["RESULTS"] != "${{ toJSON(needs) }}" || result.Steps[0].Env["WAS_CANCELLED"] != "" {
		t.Fatal("API-only reporter must consume actual prerequisite results and a supported cancellation context")
	}
}

func TestCodeResultPublication(t *testing.T) {
	jobs := readCodeGate(t)
	var result struct{ Steps []step }
	resultNode := jobs["code-result"]
	if err := resultNode.Decode(&result); err != nil {
		t.Fatal(err)
	}
	head := strings.Repeat("a", 40)
	for _, tc := range []struct {
		name        string
		failedJob   string
		outcome     string
		failAPI     bool
		wantState   string
		wantSuccess bool
	}{
		{name: "all prerequisites pass", wantState: "success", wantSuccess: true},
		{name: "commit fails", failedJob: "commit-stage", outcome: "failure", wantState: "failure"},
		{name: "acceptance fails", failedJob: "acceptance-stage", outcome: "failure", wantState: "failure"},
		{name: "ready skipped", failedJob: "ready", outcome: "skipped", wantState: "failure"},
		{name: "resolve cancelled", failedJob: "resolve", outcome: "cancelled", wantState: "failure"},
		{name: "status API fails", failAPI: true, wantState: "success"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			log := filepath.Join(dir, "calls")
			stub := strings.Join([]string{
				"#!/usr/bin/env bash",
				"set -euo pipefail",
				"case \"$*\" in *repos/example/gh-settings/statuses/*) ;; *) exit 44;; esac",
				"printf '%s\\n' \"$@\" >> \"$CALL_LOG\"",
				"[[ \"$FAIL_API\" != true ]] || exit 42",
			}, "\n")
			if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(stub), 0700); err != nil {
				t.Fatal(err)
			}
			outcomes := map[string]map[string]string{}
			for _, name := range []string{"code-start", "draft", "resolve", "commit-stage", "ready", "acceptance-stage"} {
				outcomes[name] = map[string]string{"result": "success"}
			}
			if tc.failedJob != "" {
				outcomes[tc.failedJob]["result"] = tc.outcome
			}
			encoded, err := json.Marshal(outcomes)
			if err != nil {
				t.Fatal(err)
			}
			apiFailure := "false"
			if tc.failAPI {
				apiFailure = "true"
			}
			cmd := exec.Command("bash", "-euo", "pipefail", "-c", result.Steps[0].Run)
			cmd.Env = append(os.Environ(),
				"PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GH_TOKEN=fixture", "GITHUB_REPOSITORY=example/gh-settings",
				"GITHUB_SERVER_URL=https://github.com", "GITHUB_RUN_ID=123",
				"HEAD_SHA="+head, "RESULTS="+string(encoded),
				"FAIL_API="+apiFailure, "CALL_LOG="+log)
			output, runErr := cmd.CombinedOutput()
			if (runErr == nil) != tc.wantSuccess {
				t.Fatalf("success=%v, want %v: %s", runErr == nil, tc.wantSuccess, output)
			}
			calls, readErr := os.ReadFile(log)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !strings.Contains(string(calls), "state="+tc.wantState+"\n") ||
				!strings.Contains(string(calls), "context=pass\n") ||
				!strings.Contains(string(calls), "/statuses/"+head+"\n") {
				t.Fatalf("unexpected required status request: %s", calls)
			}
		})
	}
}
