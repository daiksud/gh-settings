//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestActionsPoliciesPruneExecutableLifecycle(t *testing.T) {
	var mu sync.Mutex
	exists, deletes := true, 0
	policy := map[string]any{"id": 7, "name": "old-policy", "source_type": "Repository", "source": "owner/current", "target": "actions", "enforcement": "active", "rules": []any{map[string]any{"type": "restrict_action_events", "parameters": map[string]any{"allowed_events": []string{"push"}}}}}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == actionsEventPoliciesEndpoint:
			values := []any{}
			if exists {
				values = append(values, policy)
			}
			json.NewEncoder(w).Encode(map[string]any{"total_count": len(values), "policies": values})
		case r.Method == "GET" && r.URL.Path == actionsEventPoliciesEndpoint+"/7" && exists:
			json.NewEncoder(w).Encode(policy)
		case r.Method == "DELETE" && r.URL.Path == actionsEventPoliciesEndpoint+"/7" && exists:
			exists = false
			deletes++
			w.WriteHeader(204)
		default:
			t.Error("unexpected request", r.Method, r.URL)
			w.WriteHeader(500)
		}
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_PRUNE=actions-policies", "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    actions:\n      policies: []\n"
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	binary := acceptanceBinary(t)
	run := func(args ...string) string {
		t.Helper()
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr != nil {
			t.Fatal(args, err, result.stderr.String())
		}
		return result.stdout.String()
	}
	if out := run("plan"); !strings.Contains(strings.ToLower(out), "no managed") {
		t.Fatal(out)
	}
	if out := run("plan", "--prune", "actions-policies"); !strings.Contains(out, "delete") || !strings.Contains(out, "absent") {
		t.Fatal(out)
	}
	report := filepath.Join(fixture.root, "policies-report.json")
	run("action", "plan", report)
	for _, mode := range []string{"detailed", "status-only"} {
		out := filepath.Join(fixture.root, mode+".md")
		args := []string{"action", "summary", report, out}
		if mode == "status-only" {
			args = append(args, mode)
		}
		run(args...)
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if mode == "detailed" && !strings.Contains(string(data), "absent") {
			t.Fatal(string(data))
		}
		if mode == "status-only" && strings.Contains(string(data), "old-policy") {
			t.Fatal(string(data))
		}
	}
	run("apply", "--prune", "actions-policies")
	if out := run("plan", "--prune", "actions-policies"); !strings.Contains(out, "No differences") {
		t.Fatal(out)
	}
	run("apply", "--prune", "actions-policies")
	mu.Lock()
	defer mu.Unlock()
	if exists || deletes != 1 {
		t.Fatal(fmt.Sprint(exists, deletes))
	}
}
