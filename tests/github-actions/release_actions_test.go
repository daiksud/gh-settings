package githubactions

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

type localReleaseAction struct {
	Inputs  map[string]yaml.Node `yaml:"inputs"`
	Outputs map[string]yaml.Node `yaml:"outputs"`
	Runs    struct {
		Using string `yaml:"using"`
		Steps []step `yaml:"steps"`
	} `yaml:"runs"`
}

func readReleaseAction(t *testing.T, name string) (localReleaseAction, string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), ".github/actions", name, "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var action localReleaseAction
	if err := yaml.Unmarshal(data, &action); err != nil {
		t.Fatal(err)
	}
	if action.Runs.Using != "composite" {
		t.Fatalf("%s is not a composite action", name)
	}
	return action, string(data)
}

func TestReleaseLocalActionContracts(t *testing.T) {
	wantInputs := map[string][]string{
		"release-metadata":   {"source-sha"},
		"release-acceptance": {"target", "version", "source-sha"},
		"release-publish":    {"source-sha", "version", "previous-tag", "previous-sha"},
		"release-verify":     {"source-sha", "version"},
	}
	for name, keys := range wantInputs {
		t.Run(name, func(t *testing.T) {
			action, _ := readReleaseAction(t, name)
			if len(action.Inputs) != len(keys) {
				t.Fatalf("input inventory changed: %v", action.Inputs)
			}
			for _, key := range keys {
				if _, ok := action.Inputs[key]; !ok {
					t.Fatalf("missing input %s", key)
				}
			}
		})
	}
	metadata, metadataSource := readReleaseAction(t, "release-metadata")
	for _, key := range []string{"version", "previous-tag", "previous-sha", "build-matrix", "acceptance-matrix"} {
		if _, ok := metadata.Outputs[key]; !ok {
			t.Fatalf("metadata is missing output %s", key)
		}
	}
	for _, matrix := range []string{
		`build_matrix='{"include":[{"target":"linux-amd64","runner":"ubuntu-slim"},{"target":"darwin-arm64","runner":"macos-15"}]}'`,
		`acceptance_matrix='{"include":[{"target":"linux-amd64","runner":"ubuntu-slim"},{"target":"darwin-arm64","runner":"macos-15"},{"target":"candidate-action","runner":"ubuntu-24.04"}]}'`,
	} {
		if !strings.Contains(metadataSource, matrix) {
			t.Fatalf("metadata action lost allowlisted matrix %s", matrix)
		}
	}
}

func TestReleaseStagesOwnArtifactIdentityAndTokenBoundaries(t *testing.T) {
	build, _ := readReleaseAction(t, "build")
	acceptance, acceptanceSource := readReleaseAction(t, "release-acceptance")
	_, publishSource := readReleaseAction(t, "release-publish")
	verify, verifySource := readReleaseAction(t, "release-verify")

	for _, workflow := range []string{"ci-push.yml", "release.yml"} {
		data, err := os.ReadFile(filepath.Join(root(), ".github/workflows", workflow))
		if err != nil {
			t.Fatal(err)
		}
		source := string(data)
		for _, required := range []string{
			"uses: ./.github/actions/lint",
			"uses: ./.github/actions/unit-test",
			"uses: ./.github/actions/build",
			"release-artifact.sh write",
			"actions/upload-artifact@cf430e030ddbb5b0abf93d22962f4752f3646cd9",
			"compression-level: 0",
			"overwrite: true",
		} {
			if !strings.Contains(source, required) {
				t.Fatalf("%s: missing separate release stage %q", workflow, required)
			}
		}
		if strings.Contains(source, "uses: ./.github/actions/release-build") {
			t.Fatalf("%s: release-build wrapper must not run", workflow)
		}
	}
	if _, err := os.Stat(filepath.Join(root(), ".github/actions/release-build")); !os.IsNotExist(err) {
		t.Fatal("release-build wrapper must be removed")
	}
	if _, err := os.Stat(filepath.Join(root(), ".github/actions/setup-tools")); !os.IsNotExist(err) {
		t.Fatal("single-step setup-tools wrapper must be removed")
	}
	for name, source := range map[string]string{
		"acceptance": acceptanceSource,
		"publish":    publishSource,
		"verify":     verifySource,
	} {
		if !strings.Contains(source, "actions/download-artifact@9000827ccba6bdab643e8b6fd33ac0654aef8333") ||
			!strings.Contains(source, "digest-mismatch: error") ||
			!strings.Contains(source, "release-artifact.sh verify") {
			t.Fatalf("%s wrapper no longer verifies the manifest and artifact digest", name)
		}
	}
	if !strings.Contains(acceptanceSource, "./.action/.github/actions/e2e-test") ||
		!strings.Contains(acceptanceSource, "./.action/.github/actions/integration-test") {
		t.Fatal("acceptance wrapper lost native or public Action acceptance")
	}
	if !strings.Contains(publishSource, "bash scripts/publish-release.sh") ||
		strings.Contains(publishSource, "go build -trimpath") {
		t.Fatal("publisher must retain immutable recovery without rebuilding products")
	}
	for name, action := range map[string]localReleaseAction{
		"build": build, "acceptance": acceptance, "verify": verify,
	} {
		for _, s := range action.Runs.Steps {
			if _, ok := s.Env["GH_TOKEN"]; ok {
				t.Fatalf("%s supplies GH_TOKEN to %s", name, s.Name)
			}
			if _, ok := s.Env["GITHUB_TOKEN"]; ok {
				t.Fatalf("%s supplies GITHUB_TOKEN to %s", name, s.Name)
			}
		}
	}
}

func TestReleaseArtifactManifestBehavior(t *testing.T) {
	script := filepath.Join(root(), ".github/scripts/release-artifact.sh")
	version, source := "v1.2.3", strings.Repeat("a", 40)
	for _, target := range []string{"linux-amd64", "darwin-arm64"} {
		t.Run(target, func(t *testing.T) {
			directory := t.TempDir()
			product := filepath.Join(directory, "product")
			if err := os.WriteFile(product, []byte("product-"+target), 0700); err != nil {
				t.Fatal(err)
			}
			artifact := filepath.Join(directory, "artifact")
			command := exec.Command("bash", script, "write", target, version, source, product, artifact)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("write: %v\n%s", err, output)
			}
			result := filepath.Join(directory, "output")
			if err := os.WriteFile(result, nil, 0600); err != nil {
				t.Fatal(err)
			}
			command = exec.Command("bash", script, "verify", target, version, source, artifact, result)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("verify: %v\n%s", err, output)
			}
			outputs, err := os.ReadFile(result)
			if err != nil || !bytes.Contains(outputs, []byte("sha256=")) || !bytes.Contains(outputs, []byte("asset-path=")) {
				t.Fatalf("missing verified outputs: %q %v", outputs, err)
			}
			hash := sha256.Sum256([]byte("product-" + target))
			if !bytes.Contains(outputs, []byte(fmt.Sprintf("sha256=%x", hash))) {
				t.Fatalf("verified digest differs: %q", outputs)
			}
			command = exec.Command("bash", script, "verify", target, version, strings.Repeat("b", 40), artifact, result)
			if output, err := command.CombinedOutput(); err == nil {
				t.Fatalf("wrong source identity accepted: %s", output)
			}
			// A catalog change can resolve a different version for the same source.
			command = exec.Command("bash", script, "verify", target, "v1.2.4", source, artifact, result)
			if output, err := command.CombinedOutput(); err == nil {
				t.Fatalf("wrong version identity accepted for the same source: %s", output)
			}
			if err := os.WriteFile(filepath.Join(artifact, "unexpected"), []byte("extra"), 0600); err != nil {
				t.Fatal(err)
			}
			command = exec.Command("bash", script, "verify", target, version, source, artifact, result)
			if output, err := command.CombinedOutput(); err == nil || !strings.Contains(string(output), "inventory differs") {
				t.Fatalf("unexpected inventory accepted: err=%v output=%s", err, output)
			}
			if err := os.Remove(filepath.Join(artifact, "unexpected")); err != nil {
				t.Fatal(err)
			}
			asset := "gh-settings-" + strings.ReplaceAll(target, "-", "-")
			if err := os.WriteFile(filepath.Join(artifact, asset), []byte("tampered"), 0700); err != nil {
				t.Fatal(err)
			}
			command = exec.Command("bash", script, "verify", target, version, source, artifact, result)
			if output, err := command.CombinedOutput(); err == nil || !strings.Contains(string(output), "bytes differ") {
				t.Fatalf("tampered product accepted: err=%v output=%s", err, output)
			}
		})
	}
}

func TestReleaseCandidateFixtureToolchain(t *testing.T) {
	// The public Action itself consumes a binary, but its bounded API fixture is
	// compiled from the frozen checkout. A runner's ambient Go may be too old.
	action, _ := readReleaseAction(t, "release-acceptance")
	ci, _ := readReleaseAction(t, "ci-acceptance")
	var ciSetup step
	for _, s := range ci.Runs.Steps {
		if strings.HasPrefix(s.Uses, "jdx/mise-action@") {
			ciSetup = s
		}
	}
	setupIndex, candidateIndex := -1, -1
	for i, s := range action.Runs.Steps {
		if strings.HasPrefix(s.Uses, "jdx/mise-action@") {
			setupIndex = i
			if s.Uses != ciSetup.Uses || !reflect.DeepEqual(s.With, ciSetup.With) ||
				s.With["working_directory"] != ".action" || s.With["install_args"] != "go" ||
				s.If != "inputs.target == 'candidate-action'" {
				t.Fatalf("release fixture must use the same pinned source toolchain as CI: %+v", s)
			}
		}
		if s.Uses == "./.action/.github/actions/e2e-test" {
			candidateIndex = i
		}
	}
	if setupIndex < 0 || candidateIndex <= setupIndex {
		t.Fatal("release candidate must install its fixture toolchain before public Action acceptance")
	}

	// Build and execute the actual fixture with automatic toolchain downloads
	// disabled, just as e2e-test does. This does not rebuild the product
	// or claim that a hosted mise installation has been exercised locally.
	binary := filepath.Join(t.TempDir(), "plan-fixture")
	command := exec.Command("go", "build", "-o", binary, "./tests/github-actions/plan-fixture")
	command.Dir = root()
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build candidate API fixture with local toolchain: %v\n%s", err, output)
	}
	output, err := exec.Command(binary).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "usage: plan-fixture STATE_DIRECTORY") {
		t.Fatalf("built fixture did not execute its real entry point: err=%v output=%s", err, output)
	}
}

func TestReleaseWorkflowProtectsPublication(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "uses: ./.github/actions/release-build") {
		t.Fatal("release must contain separate build and packaging steps")
	}
	if strings.Contains(string(data), "${{ needs.resolve.outputs.source-sha }}") {
		t.Fatal("downstream jobs must consume the immutable dispatch SHA directly")
	}
	if !reflect.DeepEqual(strings.Count(string(data), "permissions: {contents: write, actions: read}"), 1) {
		t.Fatal("only publication may request contents write")
	}
}
