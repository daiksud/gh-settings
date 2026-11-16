package githubactions

import (
	"encoding/json"
	"go.yaml.in/yaml/v4"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Execute the actual aggregate scripts: missing, failed, cancelled and skipped
// prerequisites must never turn into a successful delivery gate.
func TestJSONAggregateGates(t *testing.T) {
	for _, name := range []string{"ci-push", "release"} {
		data, err := os.ReadFile(filepath.Join(root(), ".github/workflows/"+name+".yml"))
		if err != nil {
			t.Fatal(err)
		}
		var w struct {
			Jobs map[string]struct {
				Steps []step `yaml:"steps"`
			} `yaml:"jobs"`
		}
		if err := yaml.Unmarshal(data, &w); err != nil {
			t.Fatal(err)
		}
		job := "ci-push"
		if name == "release" {
			job = "delivery-gate"
		}
		script := w.Jobs[job].Steps[0].Run
		run := func(values map[string]any, success bool) {
			t.Helper()
			data, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-euo", "pipefail", "-c", script)
			cmd.Env = append(os.Environ(), "RESULTS="+string(data))
			if output, err := cmd.CombinedOutput(); (err == nil) != success {
				t.Fatalf("%s gate success=%v expected=%v: %s", name, err == nil, success, output)
			}
		}
		valid := func(reused bool) map[string]any {
			values := map[string]any{"resolve": map[string]any{"result": "success"}, "commit-stage": map[string]any{"result": "success"}, "acceptance-stage": map[string]any{"result": "success"}}
			if name == "release" {
				flag := "false"
				if reused {
					flag = "true"
					values["commit-stage"] = map[string]any{"result": "skipped"}
					values["acceptance-stage"] = map[string]any{"result": "skipped"}
				}
				values["reuse"] = map[string]any{"result": "success", "outputs": map[string]any{"reused": flag}}
			}
			return values
		}
		run(valid(false), true)
		for _, key := range []string{"resolve", "commit-stage", "acceptance-stage"} {
			for _, status := range []string{"failure", "cancelled", "skipped"} {
				v := valid(false)
				v[key] = map[string]any{"result": status}
				run(v, false)
			}
			v := valid(false)
			delete(v, key)
			run(v, false)
		}
		run(map[string]any{}, false)
		if name == "release" {
			run(valid(true), true)
			v := valid(true)
			v["reuse"].(map[string]any)["result"] = "failure"
			run(v, false)
			v = valid(true)
			v["commit-stage"] = map[string]any{"result": "success"}
			run(v, false)
			v = valid(false)
			v["reuse"].(map[string]any)["outputs"] = map[string]any{}
			run(v, false)
		}
	}
}
