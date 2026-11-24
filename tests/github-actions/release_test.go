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

func TestReleaseWorkflowDelegatesReusableStages(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]struct {
			Needs       yaml.Node         `yaml:"needs"`
			If          string            `yaml:"if"`
			Uses        string            `yaml:"uses"`
			Permissions map[string]string `yaml:"permissions"`
			With        map[string]string `yaml:"with"`
			Steps       []step            `yaml:"steps"`
		}
	}
	if err := yaml.Unmarshal(data, &w); err != nil {
		t.Fatal(err)
	}

	candidate := w.Jobs["candidate"]
	if candidate.Uses != "./.github/workflows/candidate.yml" || candidate.Needs.Value != "resolve" ||
		candidate.If != "needs.resolve.outputs.release-needed == 'true' && needs.resolve.outputs.reused == 'false'" ||
		candidate.With["source-sha"] != "${{ github.sha }}" || candidate.With["version"] != "${{ needs.resolve.outputs.version }}" {
		t.Fatal("fresh release candidates must come from the shared workflow at the frozen source and version")
	}
	resolve := w.Jobs["resolve"]
	stepID := func(id string) step { return findStep(t, job{Steps: resolve.Steps}, id) }
	if stepID("ci").If != "steps.metadata.outputs.release-needed == 'true'" ||
		stepID("reuse").If != stepID("ci").If || stepID("reuse").With["ci-run-attempt"] != "${{ steps.ci.outputs.ci-run-attempt }}" {
		t.Fatal("reuse must follow version resolution and select the checked attempt")
	}
	checkNeeds := func(name string, expected ...string) {
		t.Helper()
		var actual []string
		n := w.Jobs[name].Needs
		if err := n.Decode(&actual); err != nil || !reflect.DeepEqual(actual, expected) {
			t.Fatalf("%s dependencies: got %v, want %v", name, actual, expected)
		}
	}
	checkNeeds("select", "resolve", "candidate")
	checkNeeds("release", "resolve", "select")
	checkNeeds("verify-release", "resolve", "select", "release")
	if w.Jobs["select"].If != "${{ !cancelled() && needs.resolve.result == 'success' && needs.resolve.outputs.release-needed == 'true' }}" ||
		w.Jobs["release"].If != "${{ !cancelled() && needs.resolve.result == 'success' && needs.select.result == 'success' }}" ||
		w.Jobs["verify-release"].If != "${{ !cancelled() && needs.release.result == 'success' }}" {
		t.Fatal("release and post-publication checks must require their successful, uncancelled prerequisites")
	}
	for name, j := range w.Jobs {
		if (j.Permissions["contents"] == "write") != (name == "release") {
			t.Fatalf("unexpected contents permission in %s", name)
		}
	}
	for name, action := range map[string]string{"release": "./.github/actions/release-publish", "verify-release": "./.action/.github/actions/release-verify"} {
		j := w.Jobs[name]
		last := j.Steps[len(j.Steps)-1]
		if last.Uses != action || j.Permissions["actions"] != "read" ||
			last.With["run-id"] != "${{ needs.select.outputs.run-id }}" ||
			last.With["linux-artifact-id"] != "${{ needs.select.outputs.linux-id }}" ||
			last.With["macos-artifact-id"] != "${{ needs.select.outputs.macos-id }}" ||
			last.With["source-sha"] != "${{ github.sha }}" || last.With["version"] != "${{ needs.resolve.outputs.version }}" {
			t.Fatalf("%s must use the selected producer, accepted IDs and frozen version", name)
		}
		if name == "verify-release" && len(last.Env) > 0 {
			t.Fatal("public release verification must run without ambient credentials")
		}
	}
}

func TestCandidateDownloadFixtureIdentity(t *testing.T) {
	for _, mismatch := range []string{"", "source", "version", "checksum"} {
		t.Run("mismatch="+mismatch, func(t *testing.T) {
			f := newCompositeFixture(t)
			// Artifact downloads retain the bytes but strip their execute bit.
			if err := os.Chmod(f.base["MOCK_ASSET"], 0600); err != nil {
				t.Fatal(err)
			}
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
			source, digest, version := f.git("-C", f.packagePath, "rev-parse", "HEAD"), f.digest, fixtureCLIVersion
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
			envFile := filepath.Join(f.base["RUNNER_TEMP"], "candidate-env")
			f.write(envFile, "")
			f.base["GITHUB_ENV"] = envFile
			cmd := exec.Command("bash", filepath.Join(directory, "prepare-candidate-download.sh"), f.base["MOCK_ASSET"], digest, source, version)
			cmd.Dir, cmd.Env = f.caller, fixtureEnvironment(f.base)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			published, readErr := os.ReadFile(pathFile)
			envPublished, envReadErr := os.ReadFile(envFile)
			if envReadErr != nil {
				t.Fatal(envReadErr)
			}
			if readErr != nil {
				t.Fatal(readErr)
			}
			if _, err := os.Stat(f.base["MOCK_DOWNLOAD"]); !os.IsNotExist(err) {
				t.Fatal("candidate preparation invoked a download")
			}
			if mismatch != "" {
				if err == nil || stdout.Len() != 0 || len(published) != 0 || len(envPublished) != 0 || !strings.Contains(stderr.String(), mismatch+" mismatch") {
					t.Fatalf("mismatched candidate reached PATH publication: err=%v stdout=%q stderr=%q path=%q", err, stdout.String(), stderr.String(), published)
				}
				return
			}
			if err != nil || stderr.Len() != 0 {
				t.Fatalf("matching candidate rejected: %v\n%s", err, stderr.String())
			}
			if string(envPublished) != "SETTINGS_CANDIDATE_VERSION="+version+"\n" {
				t.Fatalf("candidate version was not selected: %q", envPublished)
			}
			fixture := strings.TrimSpace(string(published))
			if filepath.Dir(fixture) != f.base["RUNNER_TEMP"] || strings.Contains(fixture, "\n") {
				t.Fatalf("candidate PATH is not one owned directory: %q", fixture)
			}
			stagedInfo, err := os.Stat(filepath.Join(fixture, "gh-settings-linux-amd64"))
			if err != nil || stagedInfo.Mode().Perm() != 0700 {
				t.Fatalf("verified candidate is not executable: %v", err)
			}
			sourceInfo, err := os.Stat(f.base["MOCK_ASSET"])
			if err != nil || sourceInfo.Mode().Perm() != 0600 {
				t.Fatalf("input artifact executable permissions changed: %v", err)
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
			Default  bool   `yaml:"default"`
		} `yaml:"inputs"`
	}
	if err := dispatchNode.Decode(&dispatch); err != nil {
		t.Fatal(err)
	}
	if len(dispatch.Inputs) != 2 || dispatch.Inputs["source_sha"].Required || dispatch.Inputs["source_sha"].Type != "string" ||
		dispatch.Inputs["promote_to_v1"].Required || dispatch.Inputs["promote_to_v1"].Type != "boolean" || dispatch.Inputs["promote_to_v1"].Default {
		t.Fatal("release must retain the optional source SHA and default-off promotion boolean")
	}
	resolveNode := workflow.Jobs["resolve"]
	var resolve struct {
		Steps []step `yaml:"steps"`
		If    string `yaml:"if"`
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
	if resolve.If != "" ||
		branch.Env["SOURCE_REF"] != "${{ github.ref }}" || len(branch.Env) != 1 ||
		current.Env["SOURCE_SHA"] != "${{ github.sha }}" ||
		current.Env["GH_TOKEN"] != "${{ github.token }}" ||
		pin.Env["SOURCE_SHA"] != "${{ github.sha }}" ||
		pin.Env["EXPECTED_SOURCE_SHA"] != "${{ inputs.source_sha }}" ||
		len(pin.Env) != 2 || strings.Contains(current.Run, "GITHUB_OUTPUT") || strings.Contains(pin.Run, "GITHUB_OUTPUT") {
		t.Fatal("release resolve must require a main dispatch and keep the reviewed pin separate from the token and shell outputs")
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
