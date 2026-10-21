package githubactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/daiksud/gh-settings/internal/actions"
)

// The mock CLI delegates selected metadata/Summary checks to the real adapter without a product build.
func TestMain(m *testing.M) {
	if os.Getenv("GH_SETTINGS_METADATA_HELPER") == "1" {
		os.Exit(actions.Run([]string{"metadata"}, os.Stdout, os.Stderr, os.Getenv("MOCK_CLI_VERSION"), nil))
	}
	if os.Getenv("GH_SETTINGS_SUMMARY_HELPER") == "1" {
		os.Exit(actions.Run(append([]string{"summary"}, os.Args[1:]...), os.Stdout, os.Stderr, "", nil))
	}
	os.Exit(m.Run())
}

type actionField struct {
	Description string  `yaml:"description"`
	Required    bool    `yaml:"required"`
	Default     *string `yaml:"default"`
	Value       string  `yaml:"value"`
}
type compositeStep struct {
	Name  string            `yaml:"name"`
	ID    string            `yaml:"id"`
	Shell string            `yaml:"shell"`
	If    string            `yaml:"if"`
	Env   map[string]string `yaml:"env"`
	Run   string            `yaml:"run"`
}
type composite struct {
	Name        string                 `yaml:"name"`
	Author      string                 `yaml:"author"`
	Description string                 `yaml:"description"`
	Inputs      map[string]actionField `yaml:"inputs"`
	Outputs     map[string]actionField `yaml:"outputs"`
	Runs        struct {
		Using string          `yaml:"using"`
		Steps []compositeStep `yaml:"steps"`
	} `yaml:"runs"`
}

func committedCLIVersion(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), "action/cli-version"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}

func TestRootCompositeValidate(t *testing.T) {
	result := runComposite(t, "validate", "unused-fixture-token", map[string]string{"GH_TOKEN": "ambient-fixture", "GITHUB_TOKEN": "ambient-fixture"})
	if result.status != 0 {
		t.Fatalf("status=%d\n%s", result.status, result.log)
	}
	want := map[string]string{"config-sha": result.sha, "target": "owner/config", "cli-version": committedCLIVersion(t), "cli-sha256": result.digest}
	if !reflect.DeepEqual(result.outputs, want) {
		t.Fatalf("outputs=%v, want %v", result.outputs, want)
	}
	if result.calls != "validate\n" || result.metadataCalls != "action metadata\n" {
		t.Fatalf("settings calls=%q metadata calls=%q", result.calls, result.metadataCalls)
	}
	if result.summary != "" {
		t.Fatalf("unexpected Summary: %s", result.summary)
	}
}

func TestRootCompositeRejectsGlobalOnlyOrigin(t *testing.T) {
	f := newCompositeFixture(t)
	f.git("remote", "remove", "origin")
	f.write(f.base["GIT_CONFIG_GLOBAL"], "[remote \"origin\"]\n\turl = https://github.com/unrelated/wrong.git\n")
	f.git("-C", f.packagePath, "init", "--quiet")
	f.git("-C", f.packagePath, "remote", "add", "origin", "https://github.com/action-owner/source.git")
	f.git("-C", f.packagePath, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "--quiet", "-m", "test: Action source identity")

	result := f.run(t, "validate", "", map[string]string{"MOCK_REAL_METADATA": "1", "GH_REPO": "other/repository"})
	if result.status != 1 || len(result.outputs) != 0 || result.calls != "" || result.api != "" || result.injected {
		t.Fatalf("global-only origin escaped the scope boundary: %+v", result)
	}
	if !result.downloaded || result.metadataCalls != "action metadata\n" || !strings.Contains(result.log, "checkout-scoped GitHub origin remote is required") {
		t.Fatalf("scope rejection did not use real Go metadata: %+v", result)
	}
}

func TestRootCompositeRejectsMismatchedReleaseTag(t *testing.T) {
	ref := "v0.0.0"
	if ref == committedCLIVersion(t) {
		ref = "v0.0.1"
	}
	result := runComposite(t, "validate", "", map[string]string{"github.action_ref": ref})
	if result.status == 0 || len(result.outputs) != 0 || result.calls != "" || result.downloaded || !strings.Contains(result.log, "Action tag differs from committed CLI version") {
		t.Fatalf("mismatched release tag reached installation: %+v", result)
	}
}

func TestRootCompositePlan(t *testing.T) {
	result := runComposite(t, "plan", "fixture-token", map[string]string{"GH_TOKEN": "ambient-fixture", "GITHUB_TOKEN": "ambient-fixture"})
	if result.status != 0 || len(result.outputs) != 4 {
		t.Fatalf("status=%d outputs=%v\n%s", result.status, result.outputs, result.log)
	}
	if result.presentationCalls != "action plan\naction summary\n" || result.calls != "plan\n" {
		t.Fatalf("presentation calls=%q settings calls=%q", result.presentationCalls, result.calls)
	}
	for _, text := range []string{"Settings review", "settings.repository.description", `"before"`, `"desired"`, "settings.repository.has_issues: true → false", result.sha, "owner/config", "workflow-owner/workflow", "action-owner/action", "approved-action-ref", strings.Repeat("b", 40), "original-actor", "rerun-actor"} {
		if !strings.Contains(html.UnescapeString(result.summary), text) {
			t.Errorf("Summary missing %q: %s", text, result.summary)
		}
	}
	if strings.Contains(result.summary+result.log, "fixture-token") {
		t.Fatal("token escaped execution environment")
	}
}

func TestRootCompositeUnmanagedPlan(t *testing.T) {
	result := runComposite(t, "plan", "fixture-token", map[string]string{"MOCK_PLAN_OUTCOME": "unmanaged"})
	want := map[string]string{"config-sha": result.sha, "target": "owner/config", "cli-version": committedCLIVersion(t), "cli-sha256": result.digest}
	if result.status != 0 || !reflect.DeepEqual(result.outputs, want) || result.calls != "plan\n" || result.presentationCalls != "action plan\naction summary\n" {
		t.Fatalf("unmanaged plan: %+v", result)
	}
	if !strings.Contains(result.summary, "No managed settings.") || strings.Contains(result.summary, "Observed before") {
		t.Fatalf("incorrect unmanaged review: %s", result.summary)
	}
	if strings.Contains(result.summary+result.log+fmt.Sprint(result.outputs), "fixture-token") {
		t.Fatal("token escaped unmanaged planning")
	}
}

func TestRootCompositeQuotesPlanData(t *testing.T) {
	result := runComposite(t, "plan", "fixture-token", map[string]string{"MOCK_DESCRIPTION_BEFORE": "</pre><script>setting</script>\n\n::error::value", "github.actor": "<script>actor</script>\n\n## forged"})
	want := map[string]string{"config-sha": result.sha, "target": "owner/config", "cli-version": committedCLIVersion(t), "cli-sha256": result.digest}
	if result.status != 0 || !reflect.DeepEqual(result.outputs, want) {
		t.Fatalf("quoted plan: %+v", result)
	}
	for _, text := range []string{`&lt;/pre&gt;&lt;script&gt;setting&lt;/script&gt;\n\n::error::value`, "&lt;script&gt;actor&lt;/script&gt;&#10;&#10;## forged", "settings.repository.has_issues: true → false"} {
		if !strings.Contains(result.summary, text) {
			t.Fatalf("Summary lost quoted data %q: %s", text, result.summary)
		}
	}
	if strings.Contains(result.summary, "<script>") || strings.Contains(result.summary, "\n\n## forged") || strings.Contains(result.summary+result.log+fmt.Sprint(result.outputs), "fixture-token") {
		t.Fatalf("data escaped its presentation boundary: %+v", result)
	}
}

func TestRootCompositeApply(t *testing.T) {
	result := runComposite(t, "apply", "fixture-token", nil)
	if result.status != 0 || len(result.outputs) != 4 || result.calls != "apply\n" || result.summary != "" {
		t.Fatalf("status=%d outputs=%v calls=%q Summary=%s\n%s", result.status, result.outputs, result.calls, result.summary, result.log)
	}
}

func TestRootCompositeRejectsUnknownCommand(t *testing.T) {
	for _, command := range []string{"", "export", "validate --flag", `$(touch "$GITHUB_WORKSPACE/injected")`} {
		t.Run(command, func(t *testing.T) {
			result := runComposite(t, command, "", nil)
			if result.status == 0 || result.downloaded || result.calls != "" || len(result.outputs) != 0 || result.injected {
				t.Fatalf("status=%d downloaded=%t calls=%q outputs=%v injected=%t\n%s", result.status, result.downloaded, result.calls, result.outputs, result.injected, result.log)
			}
		})
	}
}

func TestRootCompositeRequiresExplicitToken(t *testing.T) {
	ambient := map[string]string{"GH_TOKEN": "ambient-fixture", "GITHUB_TOKEN": "ambient-fixture"}
	for _, command := range []string{"plan", "apply"} {
		t.Run(command, func(t *testing.T) {
			result := runComposite(t, command, "", ambient)
			if result.status == 0 || result.downloaded || result.calls != "" || len(result.outputs) != 0 {
				t.Fatalf("status=%d downloaded=%t calls=%q outputs=%v\n%s", result.status, result.downloaded, result.calls, result.outputs, result.log)
			}
		})
	}
	result := runComposite(t, "validate", "unused-fixture-token", ambient)
	if result.status != 0 || len(result.outputs) != 4 || result.calls != "validate\n" {
		t.Fatalf("validate did not ignore credentials: %+v", result)
	}
}

func TestRootCompositePreservesCLIStatus(t *testing.T) {
	for _, test := range []struct{ name, command, failure, calls string }{
		{"validate", "validate", "MOCK_VALIDATE_STATUS", "validate\n"},
		{"shared plan validation", "plan", "MOCK_VALIDATE_STATUS", "plan\n"},
		{"plan", "plan", "MOCK_REMOTE_STATUS", "plan\n"},
		{"apply", "apply", "MOCK_REMOTE_STATUS", "apply\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runComposite(t, test.command, "fixture-token", map[string]string{test.failure: "37"})
			if result.status != 37 || len(result.outputs) != 0 || result.calls != test.calls {
				t.Fatalf("status=%d outputs=%v calls=%q\n%s", result.status, result.outputs, result.calls, result.log)
			}
			if test.command == "plan" {
				diagnostic := "repository read failed"
				if test.failure == "MOCK_VALIDATE_STATUS" {
					diagnostic = "configuration validation failed"
				}
				if !strings.Contains(result.summary, "Error — plan failed.") || !strings.Contains(result.summary, diagnostic) {
					t.Fatalf("missing failed-plan review: %s", result.summary)
				}
			}
		})
	}
}

func TestRootCompositeUnusableReport(t *testing.T) {
	for _, mode := range []string{"missing", "malformed"} {
		for _, planStatus := range []string{"0", "37"} {
			t.Run(mode+planStatus, func(t *testing.T) {
				result := runComposite(t, "plan", "fixture-token", map[string]string{"MOCK_REPORT_MODE": mode, "MOCK_REMOTE_STATUS": planStatus})
				want := 1
				if planStatus == "37" {
					want = 37
				}
				diagnostic := "read plan report:"
				if mode == "malformed" {
					diagnostic = "decode plan report:"
				}
				if result.status != want || len(result.outputs) != 0 || result.summary != "" || result.presentationCalls != "action plan\naction summary\n" {
					t.Fatalf("unusable report: %+v", result)
				}
				if !strings.Contains(result.log, diagnostic) || strings.Contains(result.log+fmt.Sprint(result.outputs), "fixture-token") {
					t.Fatalf("report diagnostic/token boundary: %s", result.log)
				}
			})
		}
	}
}

func TestRootCompositeSummaryFailure(t *testing.T) {
	for _, failure := range []string{"render", "write"} {
		for _, status := range []string{"0", "37"} {
			t.Run(failure+status, func(t *testing.T) {
				extra := map[string]string{"MOCK_REMOTE_STATUS": status}
				if failure == "render" {
					extra["github.run_id"] = ""
				} else {
					extra["SUMMARY_DIRECTORY"] = "1"
				}
				result := runComposite(t, "plan", "fixture-token", extra)
				want := 1
				if status == "37" {
					want = 37
				}
				diagnostic := "missing SETTINGS_RUN_ID"
				if failure == "write" {
					diagnostic = "open Summary:"
				}
				if result.status != want || len(result.outputs) != 0 || !strings.Contains(result.log, diagnostic) || strings.Contains(result.log, "fixture-token") {
					t.Fatalf("status=%d want%d outputs=%v\n%s", result.status, want, result.outputs, result.log)
				}
			})
		}
	}
}

func TestRootCompositeFailureGates(t *testing.T) {
	for _, test := range []struct {
		name  string
		extra map[string]string
		calls string
	}{
		{"installation", map[string]string{"RUNNER_OS": "Windows"}, ""},
		{"metadata", map[string]string{"MOCK_METADATA_STATUS": "37"}, ""},
		{"output", map[string]string{"OUTPUT_DIRECTORY": "1"}, "validate\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := runComposite(t, "validate", "", test.extra)
			if result.status == 0 || len(result.outputs) != 0 || result.calls != test.calls || result.injected {
				t.Fatalf("failure gate did not hold: %+v", result)
			}
			if test.name == "metadata" && (!result.downloaded || result.metadataCalls != "action metadata\n") {
				t.Fatalf("metadata failure occurred before the verified entry: %+v", result)
			}
		})
	}
}

func TestRootCompositeStatusHandoffFailure(t *testing.T) {
	for _, status := range []string{"0", "37"} {
		t.Run(status, func(t *testing.T) {
			result := runComposite(t, "plan", "fixture-token", map[string]string{"MOCK_REMOTE_STATUS": status, "STATUS_DIRECTORY": "1"})
			want := 1
			if status == "37" {
				want = 37
			}
			if result.status != want || len(result.outputs) != 0 || result.calls != "plan\n" {
				t.Fatalf("handoff status=%d want%d outputs=%v calls=%q\n%s", result.status, want, result.outputs, result.calls, result.log)
			}
		})
	}
}

type compositeResult struct {
	status                                                                  int
	downloaded, injected                                                    bool
	outputs                                                                 map[string]string
	remainingReports                                                        []string
	log, calls, metadataCalls, presentationCalls, summary, sha, digest, api string
}

type compositeFixture struct {
	action                           composite
	caller, packagePath, sha, digest string
	base                             map[string]string
	git                              func(...string) string
	write                            func(string, string)
}

func runComposite(t *testing.T, command, token string, extra map[string]string) compositeResult {
	f := newCompositeFixture(t)
	return f.run(t, command, token, extra)
}

func fixtureEnvironment(values map[string]string) []string {
	var env []string
	for key, value := range values {
		env = append(env, key+"="+value)
	}
	return env
}

func newCompositeFixture(t *testing.T) *compositeFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var action composite
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&action); err != nil {
		t.Fatal(err)
	}
	if action.Name == "" || action.Description == "" || action.Runs.Using != "composite" {
		t.Fatal("missing composite metadata")
	}
	if len(action.Inputs) != 3 || action.Inputs["plan-reporting"].Default == nil || *action.Inputs["plan-reporting"].Default != "detailed" || !action.Inputs["command"].Required || action.Inputs["token"].Required || action.Inputs["command"].Description == "" || action.Inputs["token"].Description == "" || action.Inputs["command"].Default != nil || action.Inputs["token"].Default != nil {
		t.Fatal("input contract changed")
	}
	if len(action.Outputs) != 5 {
		t.Fatal("public output contract changed")
	}
	fixture := t.TempDir()
	caller, packagePath, bin := filepath.Join(fixture, "caller"), filepath.Join(fixture, "action package"), filepath.Join(fixture, "bin")
	for _, dir := range []string{caller, packagePath, bin, filepath.Join(fixture, "runner temp")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{filepath.Join(caller, "action"), filepath.Join(caller, ".github")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	poison := "#!/usr/bin/env bash\ntouch \"$GITHUB_WORKSPACE/injected\"\nexit 99\n"
	write(filepath.Join(caller, "action", "install-cli.sh"), poison)
	write(filepath.Join(bin, "gh-settings"), poison)
	write(filepath.Join(caller, ".github/settings.yml"), "schemaVersion: 1\n")
	asset := `#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  --version) [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]] || exit 98; echo "gh-settings $MOCK_CLI_VERSION"; exit ;;
  action)
    case "${2:-}" in
      private-reporting)
        [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]] || exit 98
        [[ ${MOCK_OLD_CLI:-0} != 1 ]] || exit 1
        echo status-only-v1; exit ;;
      metadata)
        [[ $# == 2 && -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]] || exit 98
        printf 'action metadata\n' >> "$MOCK_METADATA_CALLS"
        if [[ ${MOCK_REAL_METADATA:-} == 1 ]]; then
          GH_SETTINGS_METADATA_HELPER=1 exec "$MOCK_TEST_BINARY"
        fi
        printf 'config-sha=%s\ntarget=owner/config\ncli-version=%s\ncli-sha256=%s\n' "$MOCK_CONFIG_SHA" "$MOCK_CLI_VERSION" "$MOCK_CLI_SHA256"
        exit "${MOCK_METADATA_STATUS:-0}" ;;
      plan)
        [[ ($# == 3 || $# == 4) && (${GH_TOKEN:-} == fixture-token || ${GH_TOKEN:-} == read-fixture) && -z ${GITHUB_TOKEN:-} ]] || exit 98
        if [[ ${MOCK_DEBUG_NOISE:-0} == 1 ]]; then
          echo 'CANARY-unexpected-stdout'
          echo 'CANARY-unexpected-stderr' >&2
        fi
        printf 'action plan\n' >> "$MOCK_PRESENTATION_CALLS"
        printf 'plan\n' >> "$MOCK_CALLS"
        case "${MOCK_REPORT_MODE:-}" in
          missing) ;;
          malformed) printf '{' > "$3" ;;
          *) printf '%s\n' "$MOCK_PLAN_REPORT" > "$3" ;;
        esac
        status=${MOCK_VALIDATE_STATUS:-0}
        [[ $status != 0 ]] || status=${MOCK_REMOTE_STATUS:-0}
        if [[ $status != 0 ]]; then
          if [[ ${4:-} == status-only ]]; then echo 'private plan failed' >&2; else echo "${MOCK_ERROR:-fixture command error}" >&2; fi
        fi
        exit "$status" ;;
      summary)
        [[ ($# == 4 || $# == 5) && -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]] || exit 98
        printf 'action summary\n' >> "$MOCK_PRESENTATION_CALLS"
        GH_SETTINGS_SUMMARY_HELPER=1 exec "$MOCK_TEST_BINARY" "${@:3}" ;;
      *) exit 99 ;;
    esac ;;
  validate) [[ -z ${GH_TOKEN:-} && -z ${GITHUB_TOKEN:-} ]] || exit 98; status=${MOCK_VALIDATE_STATUS:-0} ;;
  apply)
    [[ (${GH_TOKEN:-} == fixture-token || ${GH_TOKEN:-} == app-fixture) && -z ${GITHUB_TOKEN:-} ]] || exit 98
    status=${MOCK_VALIDATE_STATUS:-0}
    if [[ $status == 0 ]]; then
      printf 'PATCH\n' >> "$MOCK_API"
      status=${MOCK_REMOTE_STATUS:-0}
    fi ;;
  *) exit 99 ;;
esac
echo "$1" >> "$MOCK_CALLS"
if [[ $status != 0 ]]; then
  echo 'fixture command error' >&2
fi
exit "$status"
`

	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(asset)))
	write(filepath.Join(fixture, "asset"), asset)
	write(filepath.Join(fixture, "checksums"), digest+"  gh-settings-linux-amd64\n")
	write(filepath.Join(bin, "curl"), `#!/usr/bin/env bash
set -euo pipefail
touch "$MOCK_DOWNLOAD"
asset="$MOCK_ASSET"
while [[ $# -gt 0 ]]; do
  [[ "$1" != */SHA256SUMS ]] || asset="$MOCK_CHECKSUMS"
  if [[ "$1" == --output ]]; then cp "$asset" "$2"; exit; fi
  shift
done
exit 1
`)
	write(filepath.Join(bin, "sha256sum"), "#!/usr/bin/env bash\nexec shasum -a 256 \"$@\"\n")
	files, err := os.ReadDir(filepath.Join(root(), "action"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(root(), "action", file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(packagePath, "action"), 0700); err != nil {
			t.Fatal(err)
		}
		write(filepath.Join(packagePath, "action", file.Name()), string(content))
	}
	base := map[string]string{"PATH": bin + string(os.PathListSeparator) + os.Getenv("PATH"), "HOME": fixture, "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": filepath.Join(fixture, "gitconfig"), "GITHUB_WORKSPACE": caller, "RUNNER_TEMP": filepath.Join(fixture, "runner temp"), "RUNNER_OS": "Linux", "RUNNER_ARCH": "X64", "GITHUB_STEP_SUMMARY": filepath.Join(fixture, "summary"), "MOCK_ASSET": filepath.Join(fixture, "asset"), "MOCK_CALLS": filepath.Join(fixture, "calls")}
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	base["MOCK_TEST_BINARY"] = testBinary
	base["MOCK_PRESENTATION_CALLS"] = filepath.Join(fixture, "presentation-calls")
	base["MOCK_CLI_VERSION"] = committedCLIVersion(t)
	base["MOCK_METADATA_CALLS"] = filepath.Join(fixture, "metadata-calls")
	base["MOCK_CHECKSUMS"] = filepath.Join(fixture, "checksums")
	base["MOCK_DOWNLOAD"] = filepath.Join(fixture, "download")
	base["MOCK_API"] = filepath.Join(fixture, "api")
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = caller
		cmd.Env = fixtureEnvironment(base)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	git("init", "--quiet")
	git("add", ".github/settings.yml")
	git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "--quiet", "-m", "fixture")
	git("remote", "add", "origin", "https://github.com/owner/config.git")
	sha := git("rev-parse", "HEAD")
	return &compositeFixture{action: action, caller: caller, packagePath: packagePath, sha: sha, digest: digest, base: base, git: git, write: write}
}

func (f *compositeFixture) run(t *testing.T, command, token string, extra map[string]string) compositeResult {
	t.Helper()
	caller, packagePath, sha, digest, action := f.caller, f.packagePath, f.sha, f.digest, f.action
	base := map[string]string{}
	for key, value := range f.base {
		base[key] = value
	}
	for _, key := range []string{"MOCK_CALLS", "MOCK_METADATA_CALLS", "MOCK_PRESENTATION_CALLS", "MOCK_DOWNLOAD", "MOCK_API", "GITHUB_STEP_SUMMARY"} {
		if err := os.Remove(base[key]); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	base["MOCK_CONFIG_SHA"], base["MOCK_CLI_SHA256"] = sha, digest
	outputDirectory := t.TempDir()
	values := map[string]string{"inputs.command": command, "inputs.plan-reporting": "detailed", "inputs.token": token, "inputs.token != ''": fmt.Sprint(token != ""), "github.action_path": packagePath,
		"github.actor": "original-actor", "github.triggering_actor": "rerun-actor", "github.event_name": "push", "github.run_id": "123", "github.run_attempt": "2", "github.workflow_ref": "workflow-owner/workflow/.github/workflows/plan.yml@refs/heads/main", "github.workflow_sha": strings.Repeat("b", 40), "github.action_repository": "action-owner/action", "github.action_ref": "approved-action-ref"}
	for key, value := range extra {
		if _, known := values[key]; known && (strings.HasPrefix(key, "github.") || key == "inputs.plan-reporting") {
			values[key] = value
		} else {
			base[key] = value
		}
	}
	outcome := base["MOCK_PLAN_OUTCOME"]
	if value := base["MOCK_VALIDATE_STATUS"]; value != "" && value != "0" {
		outcome = "invalid-config"
	} else if value := base["MOCK_REMOTE_STATUS"]; value != "" && value != "0" {
		outcome = "read-failure"
	}
	report := actions.PlanReport{Outcome: "changed", Target: "owner/config", Managed: 2, Changes: []actions.PlanChange{{Field: "repository.description", Before: "before", After: "desired"}, {Field: "repository.has_issues", Before: true, After: false}}}
	if before, supplied := base["MOCK_DESCRIPTION_BEFORE"]; supplied {
		report.Changes[0].Before = before
	}
	switch outcome {
	case "", "changed":
	case "no-difference":
		report.Outcome, report.Changes = outcome, []actions.PlanChange{}
	case "unmanaged":
		report.Outcome, report.Target, report.Managed, report.Changes = outcome, "", 0, []actions.PlanChange{}
	case "invalid-config", "read-failure":
		report.Outcome, report.Managed, report.Changes = outcome, 0, []actions.PlanChange{}
		if outcome == "invalid-config" {
			report.Target, report.Diagnostic = "", "configuration validation failed"
		} else {
			report.Diagnostic = "repository read failed"
		}
	default:
		t.Fatalf("unsupported plan fixture outcome %q", outcome)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	base["MOCK_PLAN_REPORT"] = string(data)
	if extra["SUMMARY_DIRECTORY"] == "1" {
		base["GITHUB_STEP_SUMMARY"] = caller
	}
	resolve := func(value string) string { return fixtureValue(t, value, values) }
	for _, step := range action.Runs.Steps {
		for _, name := range []string{"binary-path", "config-sha", "target", "cli-version", "cli-sha256", "exit-status", "plan-outcome"} {
			values["steps."+step.ID+".outputs."+name] = ""
		}
	}

	result := compositeResult{outputs: map[string]string{}, sha: sha, digest: digest}
	for i, step := range action.Runs.Steps {
		if result.status != 0 && !strings.HasPrefix(step.If, "always()") {
			continue
		}
		if step.Shell != "bash" || step.Run == "" || strings.Contains(step.Run, "${{") {
			t.Fatal("composite command must use explicit Bash and env-bound contexts")
		}
		switch step.If {
		case "":
		case "always() && inputs.command == 'plan'":
			if command != "plan" {
				continue
			}
		case "inputs.command == 'validate'":
			if command != "validate" {
				continue
			}
		case "inputs.command == 'plan'":
			if command != "plan" {
				continue
			}
		case "inputs.command != 'validate'":
			if command == "validate" {
				continue
			}
		default:
			t.Fatalf("unmodeled selected-path condition %q", step.If)
		}
		env := map[string]string{}
		for key, value := range base {
			env[key] = value
		}
		for key, value := range step.Env {
			if strings.Contains(value, "inputs.token") && value != "${{ inputs.token != '' }}" && (key != "GH_TOKEN" || step.If != "inputs.command != 'validate'") {
				t.Fatal("token input outside authenticated execution boundary")
			}
			env[key] = resolve(value)
		}
		outputPath := filepath.Join(outputDirectory, fmt.Sprintf("step-%d", i))
		if extra["STATUS_DIRECTORY"] == "1" && step.Env["GH_TOKEN"] == "${{ inputs.token }}" {
			outputPath = caller
		}
		if extra["OUTPUT_DIRECTORY"] == "1" {
			for _, field := range action.Outputs {
				if strings.HasPrefix(field.Value, "${{ steps."+step.ID+".outputs.") {
					outputPath = caller
				}
			}
		}
		env["GITHUB_OUTPUT"] = outputPath
		cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", step.Run)
		cmd.Dir = caller
		cmd.Env = fixtureEnvironment(env)
		output, err := cmd.CombinedOutput()
		result.log += string(output)
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok && result.status == 0 {
				result.status = exit.ExitCode()
			} else if !ok {
				t.Fatal(err)
			}
			continue
		}
		fields, _ := os.ReadFile(outputPath)
		for _, line := range strings.Split(strings.TrimSpace(string(fields)), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if ok {
				values["steps."+step.ID+".outputs."+key] = value
			}
		}
	}
	for name, field := range action.Outputs {
		if field.Description == "" {
			t.Fatal("missing output description")
		}
		key := strings.TrimSuffix(strings.TrimPrefix(field.Value, "${{ "), " }}")
		if value := values[key]; value != "" {
			result.outputs[name] = value
		}
	}
	calls, _ := os.ReadFile(base["MOCK_CALLS"])
	result.calls = string(calls)
	metadataCalls, _ := os.ReadFile(base["MOCK_METADATA_CALLS"])
	result.metadataCalls = string(metadataCalls)
	presentationCalls, _ := os.ReadFile(base["MOCK_PRESENTATION_CALLS"])
	_ = filepath.WalkDir(base["RUNNER_TEMP"], func(path string, entry os.DirEntry, err error) error {
		if err == nil && entry.Name() == "plan.json" {
			result.remainingReports = append(result.remainingReports, path)
		}
		return err
	})
	result.presentationCalls = string(presentationCalls)
	summary, _ := os.ReadFile(base["GITHUB_STEP_SUMMARY"])
	result.summary = string(summary)
	api, _ := os.ReadFile(base["MOCK_API"])
	result.api = string(api)
	_, err = os.Stat(base["MOCK_DOWNLOAD"])
	result.downloaded = err == nil
	_, err = os.Stat(filepath.Join(caller, "injected"))
	result.injected = err == nil
	return result
}

// fixtureValue resolves only exact contexts already present in the known examples/composite.
func fixtureValue(t *testing.T, value string, values map[string]string) string {
	t.Helper()
	if strings.HasPrefix(value, "${{ ") && strings.HasSuffix(value, " }}") {
		key := strings.TrimSuffix(strings.TrimPrefix(value, "${{ "), " }}")
		resolved, ok := values[key]
		if !ok {
			t.Fatalf("unbound fixture context %s", key)
		}
		return resolved
	}
	return value
}
