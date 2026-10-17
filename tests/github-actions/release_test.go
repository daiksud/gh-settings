package githubactions

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestReleaseCandidateGate(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var release struct{ Jobs map[string]yaml.Node }
	if err := yaml.Unmarshal(data, &release); err != nil {
		t.Fatal(err)
	}
	node, ok := release.Jobs["candidate-action-smoke"]
	if !ok {
		t.Fatal("release has no required pre-publication candidate Action gate")
	}
	type releaseJob struct {
		Needs       []string          `yaml:"needs"`
		Runner      string            `yaml:"runs-on"`
		Permissions map[string]string `yaml:"permissions"`
		If          string            `yaml:"if"`
		Continue    bool              `yaml:"continue-on-error"`
		Steps       []step            `yaml:"steps"`
	}
	var candidate releaseJob
	if err := node.Decode(&candidate); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(candidate.Needs, []string{"resolve", "commit-stage"}) || candidate.Runner != "ubuntu-24.04" || !reflect.DeepEqual(candidate.Permissions, map[string]string{"contents": "read"}) || candidate.If != "" || candidate.Continue {
		t.Fatalf("candidate source/build prerequisites, supported runner or read-only success gate changed: %+v", candidate)
	}
	var checkout, download step
	for _, s := range candidate.Steps {
		if s.Continue || s.Env["GH_TOKEN"] != "" || s.Env["GITHUB_TOKEN"] != "" {
			t.Fatalf("candidate step bypasses failure or supplies credentials: %+v", s)
		}
		if strings.HasPrefix(s.Uses, "actions/checkout@") {
			checkout = s
		}
		if strings.HasPrefix(s.Uses, "actions/download-artifact@") {
			download = s
		}
	}
	if checkout.With["ref"] != "${{ needs.resolve.outputs.source-sha }}" || checkout.With["path"] != ".action" || checkout.With["persist-credentials"] != "false" {
		t.Fatalf("candidate does not use the frozen isolated Action source: %+v", checkout)
	}
	if download.With["name"] != "release-linux-amd64" || download.With["path"] != "${{ runner.temp }}/product" || download.With["digest-mismatch"] != "error" {
		t.Fatalf("candidate does not consume the existing Linux artifact: %+v", download)
	}
	fixture := findStep(t, job{Steps: candidate.Steps}, "download-fixture")
	if !strings.Contains(fixture.Run, "bash .action/tests/github-actions/prepare-candidate-download.sh") || !reflect.DeepEqual(fixture.Env, map[string]string{
		"BINARY_PATH": "${{ runner.temp }}/product/gh-settings-linux-amd64", "EXPECTED_SHA256": "${{ needs.commit-stage.outputs.binary-sha256 }}",
		"SOURCE_SHA": "${{ needs.resolve.outputs.source-sha }}", "CLI_VERSION": "${{ inputs.version }}",
	}) {
		t.Fatalf("candidate fixture source/version/binary identity mapping changed: %+v", fixture)
	}
	validate := findStep(t, job{Steps: candidate.Steps}, "validate")
	if validate.Uses != "./.action" || !reflect.DeepEqual(validate.With, map[string]string{"command": "validate"}) {
		t.Fatalf("candidate must invoke actual token-free composite validate: %+v", validate)
	}
	identity := findStep(t, job{Steps: candidate.Steps}, "identity")
	if identity.If != "" || !reflect.DeepEqual(identity.Env, map[string]string{
		"ACTION_SHA": "${{ needs.resolve.outputs.source-sha }}", "CONFIG_SHA": "${{ steps.validate.outputs.config-sha }}", "TARGET": "${{ steps.validate.outputs.target }}",
		"CLI_VERSION": "${{ steps.validate.outputs.cli-version }}", "CLI_SHA256": "${{ steps.validate.outputs.cli-sha256 }}",
		"EXPECTED_CLI_VERSION": "${{ inputs.version }}", "EXPECTED_CLI_SHA256": "${{ needs.commit-stage.outputs.binary-sha256 }}",
	}) {
		t.Fatalf("mandatory candidate public-output identity bindings changed: %+v", identity)
	}
	var publish releaseJob
	publishNode := release.Jobs["publish"]
	if err := publishNode.Decode(&publish); err != nil {
		t.Fatal(err)
	}
	for _, prerequisite := range []string{"resolve", "commit-stage", "build-macos", "acceptance-linux", "acceptance-macos", "candidate-action-smoke"} {
		if !slices.Contains(publish.Needs, prerequisite) {
			t.Fatalf("publish is missing successful prerequisite %q", prerequisite)
		}
	}
	// needs and the default success condition block failed, skipped or cancelled validation.
	if publish.If != "" || publish.Continue {
		t.Fatal("publish overrides the ordinary prerequisite-success gate")
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

func TestReleaseDispatchSourcePin(t *testing.T) {
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
	if !dispatch.Inputs["version"].Required || dispatch.Inputs["source_sha"].Required || dispatch.Inputs["source_sha"].Type != "string" {
		t.Fatal("release must retain mandatory version and optional full source SHA")
	}
	resolveNode := workflow.Jobs["resolve"]
	var resolve struct {
		Steps []step `yaml:"steps"`
	}
	if err := resolveNode.Decode(&resolve); err != nil {
		t.Fatal(err)
	}
	source := findStep(t, job{Steps: resolve.Steps}, "source")
	guard := `[[ "$sha" == "$EXPECTED_SOURCE_SHA" ]]`
	if source.Env["EXPECTED_SOURCE_SHA"] != "${{ inputs.source_sha }}" ||
		!strings.Contains(source.Run, `[[ "$EXPECTED_SOURCE_SHA" =~ ^[0-9a-f]{40}$ ]]`) ||
		!strings.Contains(source.Run, guard) ||
		strings.Index(source.Run, guard) > strings.Index(source.Run, `printf 'source-sha=%s`) {
		t.Fatal("release resolve must validate and pin the requested source before output")
	}

	oneShot, _ := readWorkflow(t, "../../.github/workflows/dispatch-v0.7.1.yml")
	job := oneShot.Jobs["dispatch"]
	if !strings.Contains(job.Steps[0].Run, `"$current_main" == "$GITHUB_SHA"`) ||
		!strings.Contains(job.Steps[0].Run, `-f "source_sha=$GITHUB_SHA"`) ||
		!strings.Contains(job.Steps[0].Run, `-f "version=$RELEASE_VERSION"`) ||
		strings.Contains(job.Steps[0].Run, `git/ref/tags/`) {
		t.Fatal("one-shot dispatcher must pin approved main commit and delegate tag recovery to release")
	}
}
