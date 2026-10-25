package githubactions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestCITargetMatrices(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/actions/ci-metadata/action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Runs struct {
			Steps []step `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Runs.Steps) != 1 {
		t.Fatal("expected a single matrix producer")
	}
	output := filepath.Join(t.TempDir(), "outputs")
	cmd := exec.Command("bash", "-euo", "pipefail", "-c", metadata.Runs.Steps[0].Run)
	cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+output)
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("matrix production failed: %v: %s", err, result)
	}
	lines, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	matrices := map[string]struct {
		Include []struct {
			Target string `json:"target"`
			Runner string `json:"runner"`
			Kind   string `json:"kind"`
		} `json:"include"`
	}{}
	for _, line := range strings.Split(strings.TrimSpace(string(lines)), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found {
			t.Fatalf("invalid matrix output: %q", line)
		}
		var matrix struct {
			Include []struct {
				Target string `json:"target"`
				Runner string `json:"runner"`
				Kind   string `json:"kind"`
			} `json:"include"`
		}
		if err := json.Unmarshal([]byte(value), &matrix); err != nil {
			t.Fatal(err)
		}
		matrices[key] = matrix
	}
	build := matrices["build-matrix"].Include
	accept := matrices["acceptance-matrix"].Include
	if len(build) != 1 || build[0].Target != "linux-amd64" || build[0].Runner != "ubuntu-slim" ||
		len(accept) != 2 || accept[0].Target != build[0].Target ||
		accept[0].Kind != "process" || accept[0].Runner != "ubuntu-slim" ||
		accept[1].Target != build[0].Target || accept[1].Kind != "public-action" ||
		accept[1].Runner != "ubuntu-24.04" {
		t.Fatalf("public Action or process target coverage changed: build=%+v accept=%+v", build, accept)
	}
}

func TestCICompositesPreserveSameCandidate(t *testing.T) {
	read := func(name string) []step {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root(), ".github/actions", name, "action.yml"))
		if err != nil {
			t.Fatal(err)
		}
		var action struct {
			Runs struct {
				Steps []step `yaml:"steps"`
			} `yaml:"runs"`
		}
		if err := yaml.Unmarshal(data, &action); err != nil {
			t.Fatal(err)
		}
		return action.Runs.Steps
	}
	commit := read("ci-commit")
	acceptance := read("ci-acceptance")
	if len(commit) != 3 || commit[1].Uses != "./.github/actions/commit-stage" ||
		!strings.Contains(commit[2].Run, "manifest.json") ||
		!strings.Contains(commit[2].Run, "binary_sha256") ||
		!strings.Contains(commit[2].Run, "product_version") ||
		!strings.Contains(commit[2].Run, "source_sha") ||
		!strings.Contains(commit[2].Run, "target") {
		t.Fatalf("commit stage must produce a manifested binary: %+v", commit)
	}
	if len(acceptance) != 4 ||
		acceptance[1].Uses != "./.action/.github/actions/acceptance-stage" ||
		acceptance[3].Uses != "./.action/.github/actions/candidate-smoke" ||
		acceptance[1].If != "inputs.kind == 'process'" ||
		acceptance[3].If != "inputs.kind == 'public-action'" {
		t.Fatalf("acceptance must run process and public Action checks on transferred bytes: %+v", acceptance)
	}
	for _, s := range acceptance {
		if strings.Contains(s.Run, "go build") || strings.Contains(s.Run, "gh-settings-linux-amd64") {
			t.Fatal("acceptance must not rebuild or download the product from a release")
		}
	}
}

func TestCIManifestRejectsTampering(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/actions/ci-acceptance/action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var action struct {
		Runs struct {
			Steps []step `yaml:"steps"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(data, &action); err != nil {
		t.Fatal(err)
	}
	if len(action.Runs.Steps) == 0 {
		t.Fatal("missing manifest verification")
	}
	verify := action.Runs.Steps[0].Run
	for _, tc := range []struct {
		name      string
		mutate    string
		wantValid bool
	}{
		{name: "valid artifact", wantValid: true},
		{name: "wrong source", mutate: "source"},
		{name: "wrong target", mutate: "target"},
		{name: "wrong binary name", mutate: "name"},
		{name: "wrong digest", mutate: "digest"},
		{name: "invalid version", mutate: "version"},
		{name: "corrupted binary", mutate: "binary"},
		{name: "missing manifest", mutate: "missing"},
		{name: "extra file", mutate: "extra"},
		{name: "wrong checkout", mutate: "checkout"},
		{name: "wrong acceptance kind", mutate: "kind"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			checkout := filepath.Join(dir, ".action")
			if err := os.MkdirAll(checkout, 0755); err != nil {
				t.Fatal(err)
			}
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = checkout
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %s: %v", args, output, err)
				}
				return strings.TrimSpace(string(output))
			}
			git("init", "-q")
			git("-c", "user.name=CI", "-c", "user.email=ci@example.invalid", "commit", "-q", "--allow-empty", "-m", "fixture")
			sourceSHA := git("rev-parse", "HEAD")
			artifact := filepath.Join(dir, "product")
			if err := os.Mkdir(artifact, 0755); err != nil {
				t.Fatal(err)
			}
			binaryBytes := []byte("candidate binary\n")
			digest := sha256.Sum256(binaryBytes)
			binary := filepath.Join(artifact, "gh-settings")
			if err := os.WriteFile(binary, binaryBytes, 0600); err != nil {
				t.Fatal(err)
			}
			manifest := map[string]string{
				"source_sha": sourceSHA,
				"target": "linux-amd64",
				"binary_name": "gh-settings",
				"binary_sha256": hex.EncodeToString(digest[:]),
				"product_version": "v0.18.1",
			}
			switch tc.mutate {
			case "source":
				manifest["source_sha"] = strings.Repeat("b", 40)
			case "target":
				manifest["target"] = "linux-arm64"
			case "name":
				manifest["binary_name"] = "other"
			case "digest":
				manifest["binary_sha256"] = strings.Repeat("0", 64)
			case "version":
				manifest["product_version"] = "latest"
			case "binary":
				if err := os.WriteFile(binary, []byte("tampered"), 0600); err != nil {
					t.Fatal(err)
				}
			case "extra":
				if err := os.WriteFile(filepath.Join(artifact, "extra"), []byte("extra"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			payload, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if tc.mutate != "missing" {
				if err := os.WriteFile(filepath.Join(artifact, "manifest.json"), payload, 0600); err != nil {
					t.Fatal(err)
				}
			}
			givenSHA, kind := sourceSHA, "process"
			if tc.mutate == "checkout" {
				givenSHA = strings.Repeat("c", 40)
			}
			if tc.mutate == "kind" {
				kind = "invalid"
			}
			outputFile := filepath.Join(dir, "outputs")
			cmd := exec.Command("bash", "-euo", "pipefail", "-c", verify)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "TARGET=linux-amd64", "KIND="+kind,
				"SOURCE_SHA="+givenSHA, "ARTIFACT_DIRECTORY="+artifact,
				"GITHUB_OUTPUT="+outputFile)
			output, err := cmd.CombinedOutput()
			if (err == nil) != tc.wantValid {
				t.Fatalf("valid=%v, want %v: %s", err == nil, tc.wantValid, output)
			}
			if tc.wantValid {
				actual, err := os.ReadFile(outputFile)
				if err != nil {
					t.Fatal(err)
				}
				for _, expected := range []string{"binary-path="+binary, "binary-sha256="+manifest["binary_sha256"], "product-version=v0.18.1"} {
					if !strings.Contains(string(actual), expected) {
						t.Fatalf("missing verified output %q: %s", expected, actual)
					}
				}
			}
		})
	}
}
