package githubactions

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"
)

func workflowJobSteps(t *testing.T, filename, jobName string) []step {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root(), ".github/workflows", filename))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]struct{ Steps []step }
	}
	if err := yaml.Unmarshal(b, &w); err != nil {
		t.Fatal(err)
	}
	return w.Jobs[jobName].Steps
}

func TestDeploymentRequiresCompletedValidation(t *testing.T) {
	script := workflowJobSteps(t, "push.yml", "deployment")[0].Run
	for _, result := range []string{"success", "failure", "cancelled", "skipped", ""} {
		for _, resolution := range []string{"success", "failure", "skipped"} {
			cmd := exec.Command("bash", "-euo", "pipefail", "-c", script)
			cmd.Env = []string{"RESOLUTION=" + resolution, "CANDIDATE=" + result}
			if success := cmd.Run() == nil; success != (result == "success" && resolution == "success") {
				t.Errorf("resolution %q, candidate %q: success = %v", resolution, result, success)
			}
		}
	}
}

func TestSelectAcceptedArtifacts(t *testing.T) {
	script := workflowJobSteps(t, "release.yml", "select")[0].Run
	for _, tc := range []struct{ reused, candidate, linux, macos, want string }{
		{"true", "skipped", "21", "22", "run-id=1\nlinux-id=21\nmacos-id=22\n"},
		{"false", "success", "21", "22", "run-id=2\nlinux-id=31\nmacos-id=32\n"},
		{"false", "skipped", "", "", ""},
		{"false", "cancelled", "", "", ""},
		{"false", "failure", "", "", ""},
		{"true", "success", "21", "22", ""},
		{"true", "skipped", "", "22", ""},
		{"true", "skipped", "21", "21", ""},
		{"", "success", "21", "22", ""},
	} {
		output := filepath.Join(t.TempDir(), "output")
		cmd := exec.Command("bash", "-euo", "pipefail", "-c", script)
		cmd.Env = []string{"REUSED=" + tc.reused, "CANDIDATE=" + tc.candidate, "PUSH_RUN=1",
			"GITHUB_RUN_ID=2", "PUSH_LINUX=" + tc.linux, "PUSH_MACOS=" + tc.macos,
			"FRESH_LINUX=31", "FRESH_MACOS=32", "GITHUB_OUTPUT=" + output}
		err := cmd.Run()
		if (err == nil) != (tc.want != "") {
			t.Errorf("unexpected selection for %+v: %v", tc, err)
		}
		data, readErr := os.ReadFile(output)
		if !os.IsNotExist(readErr) && readErr != nil {
			t.Fatal(readErr)
		}
		if string(data) != tc.want {
			t.Errorf("selected %q, want %q", data, tc.want)
		}
	}
}
