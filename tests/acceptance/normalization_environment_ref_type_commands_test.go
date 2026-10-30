//go:build acceptance

package acceptance

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestEnvironmentRefTypeNormalizationCommands(t *testing.T) {
	var mu sync.Mutex
	patterns := map[int]map[string]any{7: {"id": 7, "type": "future", "name": "main"}, 8: {"id": 8, "type": "tag", "name": "main"}}
	var history []string
	finalRead, retainRaw := false, false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		base := "/repos/owner/current/environments/production"
		if r.Method == "PUT" && r.URL.Path == base {
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			want := map[string]any{"wait_timer": float64(0), "prevent_self_review": false, "can_admins_bypass": false, "reviewers": []any{}, "deployment_branch_policy": map[string]any{"protected_branches": false, "custom_branch_policies": true}}
			if !reflect.DeepEqual(payload, want) {
				t.Error("literal declared PUT differs", payload)
			}
			history = append(history, "PUT")
			fmt.Fprint(w, `{"id":1,"name":"production","protection_rules":[],"can_admins_bypass":false,"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}`)
			return
		}
		if r.Method == "POST" && r.URL.Path == base+"/deployment-branch-policies" {
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(payload, map[string]any{"type": "branch", "name": "main"}) {
				t.Error("literal canonical POST differs", payload)
			}
			history = append(history, "POST branch/main")
			patterns[9] = map[string]any{"id": 9, "type": "branch", "name": "main"}
			json.NewEncoder(w).Encode(patterns[9])
			return
		}
		if r.Method == "DELETE" && r.URL.Path == base+"/deployment-branch-policies/7" {
			history = append(history, "DELETE 7")
			delete(patterns, 7)
			if retainRaw {
				patterns[9]["type"] = "future"
			}
			w.WriteHeader(204)
			return
		}
		if r.Method != "GET" {
			t.Error("plan or Summary wrote", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		if r.URL.Path == base+"/deployment-branch-policies/7" {
			history = append(history, "GET future/main/7")
			fmt.Fprint(w, `{"id":7,"type":"future","name":"main"}`)
			return
		}
		if r.URL.Path == base+"/deployment-branch-policies" {
			ids := []int{}
			for id := range patterns {
				ids = append(ids, id)
			}
			sort.Ints(ids)
			entries := []any{}
			for _, id := range ids {
				entries = append(entries, patterns[id])
			}
			if patterns[7] == nil {
				finalRead = true
			}
			json.NewEncoder(w).Encode(map[string]any{"total_count": len(entries), "branch_policies": entries})
			return
		}
		responses := map[string]string{
			"/repos/owner/current":                `{"id":71,"full_name":"owner/current","archived":false}`,
			"/repos/owner/current/environments":   `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`,
			base:                                  `{"id":1,"name":"production","protection_rules":[],"can_admins_bypass":false,"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}`,
			base + "/deployment_protection_rules": `{"total_count":0,"custom_deployment_protection_rules":[]}`,
		}
		body, ok := responses[r.URL.Path]
		if !ok {
			t.Error("unexpected request", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		fmt.Fprint(w, body)
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    environments:
      - name: production
        deployment_protection_rules:
          wait_timer: 0
          prevent_self_review: false
          allow_administrators_to_bypass_configured_protection_rules: false
          required_reviewers: []
        deployment_branches_and_tags: selected_branches_and_tags
        deployment_branch_or_tag_rules:
          - {ref_type: branch, name_pattern: main}
          - {ref_type: tag, name_pattern: main}
`
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) string {
		t.Helper()
		result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
			t.Fatal(args, err, result.runErr, result.stderr.String())
		}
		if err := checkSettingsUnchanged(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		return result.stdout.String()
	}
	path := "repository.settings.environments.deployment_branch_or_tag_rules.ref_type"
	warning := `Warning: repository.settings.environments.deployment_branch_or_tag_rules.ref_type: observed "future" is outside UI choices; using "branch" (projection only). Environment "production", pattern 7 name_pattern "main".`
	decision := `repository.settings.environments.deployment_branch_or_tag_rules.ref_type (Environment "production"): normalized pattern observation → request [{"ref_type":"branch","name_pattern":"main"},{"ref_type":"tag","name_pattern":"main"}] (write required; raw identities differ)`
	output := run("plan")
	if strings.Count(output, warning) != 1 || strings.Count(output, "write required") != 1 || !strings.Contains(output, decision) || !strings.Contains(output, "effective before policy") || strings.Contains(output, "No differences") {
		t.Fatal("fallback equality hid write intent", output)
	}
	report := filepath.Join(fixture.root, "report.json")
	if output := run("action", "plan", report); output != "" {
		t.Fatal(output)
	}
	data, err := os.ReadFile(report)
	var saved struct {
		Outcome      string
		Managed      int
		Environments []map[string]any
	}
	if err != nil || json.Unmarshal(data, &saved) != nil || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Environments) != 1 {
		t.Fatal("private report outcome", string(data), err)
	}
	entry := saved.Environments[0]
	wantPatterns := []any{
		map[string]any{"id": float64(7), "pattern": map[string]any{"type": "branch", "name": "main"}, "ref_type_normalization": map[string]any{"path": path, "raw_value": "future", "fallback": "branch"}},
		map[string]any{"id": float64(8), "pattern": map[string]any{"type": "tag", "name": "main"}},
	}
	if entry["operation"] != "update" || !reflect.DeepEqual(entry["before"], entry["after"]) || !reflect.DeepEqual(entry["pattern_observations"], wantPatterns) || entry["deployment_mode_normalization"] != nil {
		t.Fatal("raw ownership or effective policy lost", string(data))
	}
	summary := filepath.Join(fixture.root, "summary.md")
	beforeSummary := fixture.reads.Load()
	if output := run("action", "summary", report, summary); output != "" {
		t.Fatal(output)
	}
	data, err = os.ReadFile(summary)
	decoded := html.UnescapeString(string(data))
	if err != nil || strings.Count(decoded, warning) != 1 || !strings.Contains(decoded, decision) || !strings.Contains(decoded, "effective before policy") {
		t.Fatal("detailed Summary parity", string(data), err)
	}
	private := filepath.Join(fixture.root, "status.md")
	if output := run("action", "summary", report, private, "status-only"); output != "plan-outcome=changed\n" {
		t.Fatal(output)
	}
	data, err = os.ReadFile(private)
	if err != nil || strings.Contains(string(data), path) || strings.Contains(string(data), "production") || strings.Contains(string(data), "future") || strings.Contains(string(data), "name_pattern") || fixture.reads.Load() != beforeSummary {
		t.Fatal("status privacy or Summary requested API", string(data), err)
	}
	mu.Lock()
	beforeApply := len(history)
	mu.Unlock()
	if beforeApply != 0 {
		t.Fatal("read-only commands wrote", beforeApply)
	}
	result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || result.runErr != nil || result.stdout.String() != "Applied repository settings.\n" || result.stderr.String() != warning+"\n"+warning+"\n" {
		t.Fatal("apply outcome or fresh diagnostics", err, result.runErr, result.stdout.String(), result.stderr.String())
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	wantHistory := []string{"PUT", "POST branch/main", "GET future/main/7", "DELETE 7"}
	wantPatternsAfter := map[int]map[string]any{8: {"id": 8, "type": "tag", "name": "main"}, 9: {"id": 9, "type": "branch", "name": "main"}}
	if !reflect.DeepEqual(history, wantHistory) || !reflect.DeepEqual(patterns, wantPatternsAfter) || !finalRead {
		t.Error("raw replacement or fresh confirmation", history, patterns, finalRead)
	}
	mu.Unlock()
	if output := run("plan"); output != "No differences for managed repository settings.\n" {
		t.Fatal(output)
	}
	if output := run("apply"); output != "Applied repository settings.\n" {
		t.Fatal(output)
	}
	mu.Lock()
	if !reflect.DeepEqual(history, wantHistory) || !reflect.DeepEqual(patterns, wantPatternsAfter) {
		t.Error("canonical repeat wrote", history, patterns)
	}
	mu.Unlock()
	mu.Lock()
	patterns = map[int]map[string]any{7: {"id": 7, "type": "future", "name": "main"}, 8: {"id": 8, "type": "tag", "name": "main"}}
	history = nil
	finalRead, retainRaw = false, true
	mu.Unlock()
	result, err = executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
	var failed *exec.ExitError
	finalWarning := `Warning: repository.settings.environments.deployment_branch_or_tag_rules.ref_type: observed "future" is outside UI choices; using "branch" (projection only). Environment "production", pattern 9 name_pattern "main".`
	if err != nil || !errors.As(result.runErr, &failed) || failed.ExitCode() != 1 || result.stdout.Len() != 0 || !strings.HasPrefix(result.stderr.String(), warning+"\n"+warning+"\n"+finalWarning+"\n") || strings.Count(result.stderr.String(), "Warning:") != 3 || !strings.Contains(result.stderr.String(), "verify environment readback (write may have applied)") {
		t.Fatal("unsupported final state confirmed success", err, result.runErr, result.stdout.String(), result.stderr.String())
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	wantFailedPatterns := map[int]map[string]any{8: {"id": 8, "type": "tag", "name": "main"}, 9: {"id": 9, "type": "future", "name": "main"}}
	if !reflect.DeepEqual(history, wantHistory) || !reflect.DeepEqual(patterns, wantFailedPatterns) || !finalRead {
		t.Error("failure retried or hid prior mutations", history, patterns, finalRead)
	}
	mu.Unlock()
	if err := fixture.checkRequests(false); err != nil {
		t.Fatal(err)
	}
}
