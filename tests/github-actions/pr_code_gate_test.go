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

func TestPRWorkflowNamedAggregateCheck(t *testing.T) {
    source, err := os.ReadFile(filepath.Join(root(), ".github/workflows/ci-pr.yml"))
    if err != nil { t.Fatal(err) }
    var w struct {
        Jobs map[string]yaml.Node `yaml:"jobs"`
        Concurrency struct {
            Group string `yaml:"group"`
            Cancel bool `yaml:"cancel-in-progress"`
        } `yaml:"concurrency"`
    }
    if err := yaml.Unmarshal(source, &w); err != nil { t.Fatal(err) }
    if w.Concurrency.Group != "ci-pr-${{ github.event.pull_request.number }}" || !w.Concurrency.Cancel {
        t.Fatal("every PR event must validate one current candidate")
    }
    for _, old := range []string{"pass", "code-start", "code-result"} {
        if _, exists := w.Jobs[old]; exists { t.Fatalf("legacy job remains: %s", old) }
    }
    if strings.Contains(string(source), "/statuses/") || strings.Contains(string(source), "context=pass") {
        t.Fatal("required checks must be native jobs, not manually published statuses")
    }
    var draft struct { Needs string `yaml:"needs"`; If string `yaml:"if"` }
    if err := w.Jobs["draft"].Decode(&draft); err != nil { t.Fatal(err) }
    if draft.Needs != "" || draft.If != "" { t.Fatal("Draft must start every PR run") }
    var check struct {
        Needs []string `yaml:"needs"`
        If string `yaml:"if"`
        Permissions map[string]string `yaml:"permissions"`
        Steps []step `yaml:"steps"`
    }
    if err := w.Jobs["ci-pr"].Decode(&check); err != nil { t.Fatal(err) }
    if !reflect.DeepEqual(check.Needs, []string{"draft", "resolve", "commit-stage", "ready", "acceptance-stage"}) ||
        check.If != "always() && !cancelled()" || len(check.Permissions) != 0 || len(check.Steps) != 1 {
        t.Fatal("ci-pr must aggregate all stages with no write permissions")
    }
    if check.Steps[0].Uses != "" || check.Steps[0].Continue ||
        check.Steps[0].Env["RESULTS"] != "${{ toJSON(needs) }}" ||
        !strings.Contains(check.Steps[0].Run, "length == 5 and all(.[]; .result == \"success\")") {
        t.Fatal("ci-pr must fail when any prerequisite does not succeed")
    }
}

func TestPRWorkflowAggregateRejectsUnsuccessfulStages(t *testing.T) {
    source, err := os.ReadFile(filepath.Join(root(), ".github/workflows/ci-pr.yml"))
    if err != nil { t.Fatal(err) }
    var w struct { Jobs map[string]yaml.Node `yaml:"jobs"` }
    if err := yaml.Unmarshal(source, &w); err != nil { t.Fatal(err) }
    var check struct { Steps []step `yaml:"steps"` }
    if err := w.Jobs["ci-pr"].Decode(&check); err != nil { t.Fatal(err) }
    names := []string{"draft", "resolve", "commit-stage", "ready", "acceptance-stage"}
    for _, tc := range []struct { name, failed, outcome string; wantOK bool }{
        {"all-success", "", "", true},
        {"failed-commit", "commit-stage", "failure", false},
        {"skipped-ready", "ready", "skipped", false},
        {"cancelled-acceptance", "acceptance-stage", "cancelled", false},
    } {
        t.Run(tc.name, func(t *testing.T) {
            results := make(map[string]map[string]string)
            for _, name := range names { results[name] = map[string]string{"result":"success"} }
            if tc.failed != "" { results[tc.failed]["result"] = tc.outcome }
            value, err := json.Marshal(results)
            if err != nil { t.Fatal(err) }
            cmd := exec.Command("bash", "-euo", "pipefail", "-c", check.Steps[0].Run)
            cmd.Env = append(os.Environ(), "RESULTS="+string(value))
            output, err := cmd.CombinedOutput()
            if (err == nil) != tc.wantOK { t.Fatalf("success=%t, want=%t: %s", err==nil, tc.wantOK, output) }
        })
    }
}
