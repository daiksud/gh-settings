package githubactions

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestReleaseWorkflowDelegatesThreeStages(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var release struct{ Jobs map[string]yaml.Node }
	if err := yaml.Unmarshal(data, &release); err != nil {
		t.Fatal(err)
	}
	wantJobs := []string{"resolve", "commit-stage", "acceptance-stage", "release", "verify-release"}
	if len(release.Jobs) != len(wantJobs) {
		t.Fatalf("release job inventory changed: %v", release.Jobs)
	}
	for _, name := range wantJobs {
		if _, ok := release.Jobs[name]; !ok {
			t.Fatalf("release is missing %s", name)
		}
	}
	type releaseJob struct {
		Needs       yaml.Node         `yaml:"needs"`
		Runner      string            `yaml:"runs-on"`
		Permissions map[string]string `yaml:"permissions"`
		If          string            `yaml:"if"`
		Continue    bool              `yaml:"continue-on-error"`
		Strategy    struct {
			FailFast bool   `yaml:"fail-fast"`
			Matrix   string `yaml:"matrix"`
		} `yaml:"strategy"`
		Steps []step `yaml:"steps"`
	}
	needs := func(t *testing.T, node yaml.Node) []string {
		t.Helper()
		if node.Kind == yaml.ScalarNode {
			return []string{node.Value}
		}
		var result []string
		if err := node.Decode(&result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	var build releaseJob
	buildNode := release.Jobs["commit-stage"]
	if err := buildNode.Decode(&build); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(needs(t, build.Needs), []string{"resolve"}) || build.Runner != "${{ matrix.runner }}" || !reflect.DeepEqual(build.Permissions, map[string]string{"contents": "read"}) || build.If != "" || build.Continue || build.Strategy.FailFast || build.Strategy.Matrix != "${{ fromJSON(needs.resolve.outputs.build-matrix) }}" {
		t.Fatalf("matrix build gate changed: %+v", build)
	}
	buildAction := build.Steps[len(build.Steps)-1]
	if buildAction.Uses != "./.github/actions/release-build" || !reflect.DeepEqual(buildAction.With, map[string]string{
		"target": "${{ matrix.target }}", "version": "${{ needs.resolve.outputs.version }}", "source-sha": "${{ github.sha }}",
	}) {
		t.Fatalf("build does not delegate the frozen target identity: %+v", buildAction)
	}
	var acceptance releaseJob
	acceptanceNode := release.Jobs["acceptance-stage"]
	if err := acceptanceNode.Decode(&acceptance); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(needs(t, acceptance.Needs), []string{"resolve", "commit-stage"}) || acceptance.Runner != "${{ matrix.runner }}" || acceptance.Strategy.FailFast || acceptance.Strategy.Matrix != "${{ fromJSON(needs.resolve.outputs.acceptance-matrix) }}" || !reflect.DeepEqual(acceptance.Permissions, map[string]string{"contents": "read"}) || acceptance.If != "" || acceptance.Continue {
		t.Fatalf("matrix acceptance gate changed: %+v", acceptance)
	}
	acceptAction := acceptance.Steps[len(acceptance.Steps)-1]
	if acceptAction.Uses != "./.action/.github/actions/release-acceptance" || !reflect.DeepEqual(acceptAction.With, map[string]string{
		"target": "${{ matrix.target }}", "version": "${{ needs.resolve.outputs.version }}", "source-sha": "${{ github.sha }}",
	}) {
		t.Fatalf("acceptance does not delegate the frozen target identity: %+v", acceptAction)
	}
	var publish releaseJob
	publishNode := release.Jobs["release"]
	if err := publishNode.Decode(&publish); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(needs(t, publish.Needs), []string{"resolve", "commit-stage", "acceptance-stage"}) || !reflect.DeepEqual(publish.Permissions, map[string]string{"contents": "write"}) || publish.If != "" || publish.Continue {
		t.Fatalf("publication overrides the ordinary accepted-product gate: %+v", publish)
	}
	publishAction := publish.Steps[len(publish.Steps)-1]
	if publishAction.Uses != "./.github/actions/release-publish" || publishAction.Env["GH_TOKEN"] != "${{ github.token }}" || !reflect.DeepEqual(publishAction.With, map[string]string{
		"source-sha": "${{ github.sha }}", "version": "${{ needs.resolve.outputs.version }}",
		"previous-tag": "${{ needs.resolve.outputs.previous-tag }}", "previous-sha": "${{ needs.resolve.outputs.previous-sha }}",
	}) {
		t.Fatalf("publication does not delegate the accepted immutable identity: %+v", publishAction)
	}
	var verify releaseJob
	verifyNode := release.Jobs["verify-release"]
	if err := verifyNode.Decode(&verify); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(needs(t, verify.Needs), []string{"resolve", "release"}) || !reflect.DeepEqual(verify.Permissions, map[string]string{"contents": "read"}) || verify.If != "" || verify.Continue {
		t.Fatalf("published verification gate changed: %+v", verify)
	}
	verifyAction := verify.Steps[len(verify.Steps)-1]
	if verifyAction.Uses != "./.action/.github/actions/release-verify" || len(verifyAction.Env) != 0 || !reflect.DeepEqual(verifyAction.With, map[string]string{
		"source-sha": "${{ github.sha }}", "version": "${{ needs.resolve.outputs.version }}",
	}) {
		t.Fatalf("published verification does not remain token-free and source-bound: %+v", verifyAction)
	}
}

func TestCandidateDownloadFixtureIdentity(t *testing.T) {
	for _, mismatch := range []string{"", "source", "version", "checksum"} {
		t.Run("mismatch="+mismatch, func(t *testing.T) {
			f := newCompositeFixture(t)
			directory := filepath.Join(f.packagePath, "tests/github-actions")
			if err := os.MkdirAll(directory, 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"prepare-candidate-download.sh", "candidate-curl.sh"} {
				data, err := os.ReadFile(filepath.Join(root(), "tests/github-actions", name))
				if err != nil {
					t.Fatal(err)
				}
				f.write(filepath.Join(directory, name), string(data))
			}
			f.git("-C", f.packagePath, "init", "--quiet")
			f.git("-C", f.packagePath, "add", "action", "tests")
			f.git("-C", f.packagePath, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "test: candidate source")
			source, digest, version := f.git("-C", f.packagePath, "rev-parse", "HEAD"), f.digest, committedCLIVersion(t)
			switch mismatch {
			case "source":
				source = strings.Repeat("a", 40)
			case "version":
				version = "v999.0.0"
			case "checksum":
				digest = strings.Repeat("0", 64)
			}
			pathFile := filepath.Join(f.base["RUNNER_TEMP"], "candidate-path")
			f.write(pathFile, "")
			f.base["GITHUB_PATH"] = pathFile
			cmd := exec.Command("bash", filepath.Join(directory, "prepare-candidate-download.sh"), f.base["MOCK_ASSET"], digest, source, version)
			cmd.Dir, cmd.Env = f.caller, fixtureEnvironment(f.base)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			published, readErr := os.ReadFile(pathFile)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if _, err := os.Stat(f.base["MOCK_DOWNLOAD"]); !os.IsNotExist(err) {
				t.Fatal("candidate preparation invoked a download")
			}
			if mismatch != "" {
				if err == nil || stdout.Len() != 0 || len(published) != 0 || !strings.Contains(stderr.String(), mismatch+" mismatch") {
					t.Fatalf("mismatched candidate reached PATH publication: err=%v stdout=%q stderr=%q path=%q", err, stdout.String(), stderr.String(), published)
				}
				return
			}
			if err != nil || stderr.Len() != 0 {
				t.Fatalf("matching candidate rejected: %v\n%s", err, stderr.String())
			}
			fixture := strings.TrimSpace(string(published))
			if filepath.Dir(fixture) != f.base["RUNNER_TEMP"] || strings.Contains(fixture, "\n") {
				t.Fatalf("candidate PATH is not one owned directory: %q", fixture)
			}
			asset, err := os.ReadFile(f.base["MOCK_ASSET"])
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string][]byte{
				"gh-settings-linux-amd64": asset, "SHA256SUMS": []byte(digest + "  gh-settings-linux-amd64\n"), "cli-version": []byte(version + "\n"),
			} {
				actual, err := os.ReadFile(filepath.Join(fixture, name))
				if err != nil || !bytes.Equal(actual, want) {
					t.Fatalf("candidate %s differs from the accepted input: %v", name, err)
				}
			}
		})
	}
}

func TestReleaseOptionalSourcePin(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		On   map[string]yaml.Node `yaml:"on"`
		Jobs map[string]yaml.Node `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	dispatchNode := workflow.On["workflow_dispatch"]
	var dispatch struct {
		Inputs map[string]struct {
			Required bool   `yaml:"required"`
			Type     string `yaml:"type"`
		} `yaml:"inputs"`
	}
	if err := dispatchNode.Decode(&dispatch); err != nil {
		t.Fatal(err)
	}
	if len(dispatch.Inputs) != 1 || dispatch.Inputs["source_sha"].Required || dispatch.Inputs["source_sha"].Type != "string" {
		t.Fatal("release must calculate its version and retain only an optional full source SHA")
	}
	resolveNode := workflow.Jobs["resolve"]
	var resolve struct {
		Outputs map[string]string `yaml:"outputs"`
		Steps   []step            `yaml:"steps"`
		If      string            `yaml:"if"`
	}
	if err := resolveNode.Decode(&resolve); err != nil {
		t.Fatal(err)
	}
	var branch, current, pin step
	for _, candidate := range resolve.Steps {
		switch candidate.Name {
		case "Require a main-branch dispatch":
			branch = candidate
		case "Require the dispatched main commit to remain current":
			current = candidate
		case "Validate the optional reviewed source pin":
			pin = candidate
		}
	}
	if resolve.If != "" || resolve.Outputs["source-sha"] != "${{ github.sha }}" ||
		branch.Env["SOURCE_REF"] != "${{ github.ref }}" || len(branch.Env) != 1 ||
		current.Env["SOURCE_SHA"] != "${{ github.sha }}" ||
		current.Env["GH_TOKEN"] != "${{ github.token }}" ||
		pin.Env["SOURCE_SHA"] != "${{ github.sha }}" ||
		pin.Env["EXPECTED_SOURCE_SHA"] != "${{ inputs.source_sha }}" ||
		len(pin.Env) != 2 || strings.Contains(current.Run, "GITHUB_OUTPUT") || strings.Contains(pin.Run, "GITHUB_OUTPUT") {
		t.Fatal("release resolve must report non-main dispatches, publish trusted github.sha, and keep the reviewed pin separate from the token and shell outputs")
	}

	// The hosted Ubuntu workflow uses modern Bash. macOS's system Bash 3.2
	// does not apply errexit to a failed [[ ... ]] compound command.
	version, err := exec.Command("bash", "-c", `printf '%s' "${BASH_VERSINFO[0]}"`).Output()
	if err != nil {
		t.Fatal(err)
	}
	major, err := strconv.Atoi(string(version))
	if err != nil {
		t.Fatalf("invalid Bash major version %q: %v", version, err)
	}
	if major < 5 {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("hosted resolve-step coverage requires Bash 5 or newer")
		}
		t.Skip("resolve-step execution requires Bash 5 or newer, matching the hosted Ubuntu runner; structural bindings were checked")
	}

	for _, test := range []struct {
		name, ref string
		success   bool
	}{
		{name: "main", ref: "refs/heads/main", success: true},
		{name: "non-main", ref: "refs/heads/release-test"},
	} {
		t.Run(test.name+" dispatch", func(t *testing.T) {
			cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", branch.Run)
			cmd.Env = fixtureEnvironment(map[string]string{"SOURCE_REF": test.ref})
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if test.success != (err == nil) || stdout.Len() != 0 || (test.success && stderr.Len() != 0) {
				t.Fatalf("branch check: err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
			}
		})
	}

	sha := strings.Repeat("a", 40)
	for _, test := range []struct {
		name, response, apiStatus string
		success                   bool
	}{
		{name: "current main", response: sha, apiStatus: "0", success: true},
		{name: "main moved", response: strings.Repeat("b", 40), apiStatus: "0"},
		{name: "invalid API response", response: "not-a-sha", apiStatus: "0"},
		{name: "API failure with valid stdout", response: sha, apiStatus: "7"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			if err := os.Mkdir(bin, 0700); err != nil {
				t.Fatal(err)
			}
			stub := `#!/bin/bash
set -euo pipefail
[[ "$1" == api ]] || exit 90
shift
endpoint='' query=''
while [[ $# -gt 0 ]]; do
  case "$1" in
    --jq|-q) query="$2"; shift 2 ;;
    *) [[ -z "$endpoint" ]] || exit 90; endpoint="$1"; shift ;;
  esac
done
[[ "$endpoint" == "repos/$GITHUB_REPOSITORY/git/ref/heads/main" ]] || exit 91
printf '%s\n' "$endpoint" >> "$GH_CALLS"
[[ "$query" == .object.sha ]] || exit 92
printf '%s\n' "$GH_RESPONSE"
exit "$GH_STATUS"
`
			if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(stub), 0700); err != nil {
				t.Fatal(err)
			}
			calls := filepath.Join(dir, "calls")
			if err := os.WriteFile(calls, nil, 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", current.Run)
			cmd.Dir = dir
			cmd.Env = fixtureEnvironment(map[string]string{
				"PATH": bin + string(os.PathListSeparator) + os.Getenv("PATH"), "SOURCE_SHA": sha,
				"GITHUB_REPOSITORY": "fixture-owner/fixture-repo", "GH_CALLS": calls,
				"GH_RESPONSE": test.response, "GH_STATUS": test.apiStatus,
			})
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if test.success != (err == nil) || stdout.Len() != 0 || (test.success && stderr.Len() != 0) {
				t.Fatalf("current-main check: err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
			}
			requested, err := os.ReadFile(calls)
			if err != nil || string(requested) != "repos/fixture-owner/fixture-repo/git/ref/heads/main\n" {
				t.Fatalf("main API request=%q err=%v", requested, err)
			}
		})
	}

	for _, test := range []struct {
		name, requested string
		success         bool
	}{
		{name: "omitted pin", success: true},
		{name: "matching pin", requested: sha, success: true},
		{name: "mismatched pin", requested: strings.Repeat("b", 40)},
		{name: "invalid requested pin", requested: "main"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", pin.Run)
			cmd.Dir = dir
			cmd.Env = fixtureEnvironment(map[string]string{
				"EXPECTED_SOURCE_SHA": test.requested,
				"SOURCE_SHA":          sha,
			})
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if test.success != (err == nil) || stdout.Len() != 0 || (test.success && stderr.Len() != 0) {
				t.Fatalf("source resolution: err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
			}
		})
	}
}
