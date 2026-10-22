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
	if !reflect.DeepEqual(result.Needs, []string{"code-start", "draft", "commit-stage", "ready", "acceptance-stage", "composite-smoke"}) || result.If != "always() && ("+codeEventCondition+")" || !reflect.DeepEqual(result.Permissions, map[string]string{"statuses": "write", "pull-requests": "read"}) || len(result.Steps) != 1 {
		t.Fatal("code result must aggregate every prerequisite only for code/base events")
	}
	if result.Steps[0].Uses != "" || result.Steps[0].Continue || result.Steps[0].Env["RESULTS"] != "${{ toJSON(needs) }}" || result.Steps[0].Env["WAS_CANCELLED"] != "${{ job.status == 'cancelled' }}" {
		t.Fatal("API-only reporter must consume actual prerequisite results and a supported cancellation context")
	}
}

func TestCodeResultPublication(t *testing.T) {
	jobs := readCodeGate(t)
	var result struct{ Steps []step }
	node := jobs["code-result"]
	if err := node.Decode(&result); err != nil {
		t.Fatal(err)
	}
	head, base := strings.Repeat("a", 40), strings.Repeat("b", 40)
	pr := `{"head":{"sha":"` + head + `"},"base":{"sha":"` + base + `","ref":"main"}}`
	rules := `[[{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"pass","integration_id":15368},{"context":"gh-settings","integration_id":15368},{"context":"pr-message","integration_id":15368}]}}]]`
	for _, test := range []struct {
		name, outcome, cancelled, pr, rules, failAPI, wantState string

		wantSuccess bool
	}{
		{name: "all successful and enforced", wantState: "success", wantSuccess: true},
		{name: "failed code", outcome: "failure", wantState: "failure"},
		{name: "skipped code", outcome: "skipped", wantState: "failure"},
		{name: "cancelled code", outcome: "cancelled", wantState: "failure"},
		{name: "cancelled workflow", cancelled: "true", wantState: "failure"},
		{name: "independent check not required", rules: "[[]]", wantState: "failure"},
		{name: "wrong required provider", rules: strings.ReplaceAll(rules, "15368", "42"), wantState: "failure"},
		{name: "head changed", pr: strings.ReplaceAll(pr, head, strings.Repeat("c", 40))},
		{name: "base changed", pr: strings.ReplaceAll(pr, base, strings.Repeat("c", 40))},
		{name: "base branch changed", pr: strings.ReplaceAll(pr, "main", "other")},
		{name: "incomplete PR observation", pr: "{}"},
		{name: "PR read fails", failAPI: "pulls"},
		{name: "rules read fails", failAPI: "rules"},
		{name: "status write fails", failAPI: "statuses", wantState: "success"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			log := filepath.Join(dir, "calls")
			stub := `#!/usr/bin/env bash
set -euo pipefail
endpoint=''
for arg in "$@"; do
  case "$arg" in repos/*) endpoint="$arg";; esac
done
case "$endpoint" in
  repos/example/gh-settings/pulls/1)
    [[ "$FAIL_API" != pulls ]] || exit 42
    printf '%s\n' "$FAKE_PR"
    ;;
  repos/example/gh-settings/rules/branches/main)
    [[ "$FAIL_API" != rules ]] || exit 42
    printf '%s\n' "$FAKE_RULES"
    ;;
  repos/example/gh-settings/statuses/*)
    printf '%s\n' "$@" >> "$CALL_LOG"
    [[ "$FAIL_API" != statuses ]] || exit 42
    ;;
  *) echo "unexpected API request: $endpoint" >&2; exit 43;;
esac
`
			if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(stub), 0700); err != nil {
				t.Fatal(err)
			}
			outcomes := map[string]map[string]string{}
			for _, name := range []string{"code-start", "draft", "commit-stage", "ready", "acceptance-stage", "composite-smoke"} {
				outcomes[name] = map[string]string{"result": "success"}
			}
			if test.outcome != "" {
				outcomes["acceptance-stage"]["result"] = test.outcome
			}
			encoded, err := json.Marshal(outcomes)
			if err != nil {
				t.Fatal(err)
			}
			actualPR, actualRules, cancelled := pr, rules, "false"
			if test.pr != "" {
				actualPR = test.pr
			}
			if test.rules != "" {
				actualRules = test.rules
			}
			if test.cancelled != "" {
				cancelled = test.cancelled
			}
			cmd := exec.Command("bash", "-euo", "pipefail", "-c", result.Steps[0].Run)
			cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GH_TOKEN=fixture", "GITHUB_REPOSITORY=example/gh-settings", "GITHUB_SERVER_URL=https://github.com", "GITHUB_RUN_ID=123",
				"HEAD_SHA="+head, "BASE_SHA="+base, "BASE_REF=main", "PR_NUMBER=1", "RESULTS="+string(encoded),
				"WAS_CANCELLED="+cancelled, "FAKE_PR="+actualPR, "FAKE_RULES="+actualRules, "FAIL_API="+test.failAPI, "CALL_LOG="+log)
			output, runErr := cmd.CombinedOutput()
			if (runErr == nil) != test.wantSuccess {
				t.Fatalf("success=%v, want %v: %s", runErr == nil, test.wantSuccess, output)
			}
			calls, readErr := os.ReadFile(log)
			if readErr != nil && !os.IsNotExist(readErr) {
				t.Fatal(readErr)
			}
			if test.wantState == "" {
				if len(calls) != 0 {
					t.Fatalf("stale or unreadable metadata wrote a code status: %s", calls)
				}
			} else if !strings.Contains(string(calls), "state="+test.wantState+"\n") || !strings.Contains(string(calls), "context=pass\n") || !strings.Contains(string(calls), "/statuses/"+head+"\n") {
				t.Fatalf("unexpected required status request: %s", calls)
			}
		})
	}
}
