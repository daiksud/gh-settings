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

func TestEnvironmentPatternBinaryTypedReconciliationAndRepeat(t *testing.T) {
	var mu sync.Mutex
	patterns := map[string]map[string]any{"7": {"id": 7, "name": "old", "type": "branch"}, "8": {"id": 8, "name": "same", "type": "tag"}}
	writes := []string{}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		base := "/repos/owner/current/environments/production"
		if r.Method == "GET" {
			switch r.URL.Path {
			case "/repos/owner/current":
				fmt.Fprint(w, `{"id":71,"full_name":"owner/current","archived":false}`)
				return
			case "/repos/owner/current/environments":
				fmt.Fprint(w, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`)
				return
			case base:
				fmt.Fprint(w, `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true},"can_admins_bypass":false}`)
				return
			case base + "/deployment_protection_rules":
				fmt.Fprint(w, `{"total_count":0,"custom_deployment_protection_rules":[]}`)
				return
			case base + "/deployment-branch-policies":
				entries := []any{}
				for _, entry := range patterns {
					entries = append(entries, entry)
				}
				json.NewEncoder(w).Encode(map[string]any{"total_count": len(entries), "branch_policies": entries})
				return
			case base + "/deployment-branch-policies/7":
				json.NewEncoder(w).Encode(patterns["7"])
				return
			}
		}
		writes = append(writes, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == "PUT" && r.URL.Path == base:
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if len(payload) != 5 || payload["wait_timer"] != float64(0) || payload["can_admins_bypass"] != false || payload["deployment_branch_patterns"] != nil {
				t.Error("incorrect environment PUT", payload)
			}
			fmt.Fprint(w, `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true},"can_admins_bypass":false}`)
		case r.Method == "POST" && r.URL.Path == base+"/deployment-branch-policies":
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if len(payload) != 2 || payload["type"] != "branch" || payload["name"] != "same" {
				t.Error(payload)
			}
			patterns["9"] = map[string]any{"id": 9, "type": "branch", "name": "same"}
			json.NewEncoder(w).Encode(patterns["9"])
		case r.Method == "DELETE" && r.URL.Path == base+"/deployment-branch-policies/7":
			delete(patterns, "7")
			w.WriteHeader(204)
		default:
			t.Error("unexpected request", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := labelIDFixtureEnv(fixture.cfg, false)
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
          - {ref_type: branch, name_pattern: same}
          - {ref_type: tag, name_pattern: same}
`
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"validate", "plan", "apply", "plan", "apply"} {
		result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{command}, false)
		if err != nil || result.runErr != nil {
			t.Fatal(command, result, err)
		}
		mu.Lock()
		writeCount := len(writes)
		mu.Unlock()
		if command == "plan" && writeCount > 0 && !strings.Contains(result.stdout.String(), "No differences") {
			t.Fatal(result)
		}
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || string(data) != input {
		t.Fatal("unexpected ID writeback or YAML change", err, string(data))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(writes) != 3 || len(patterns) != 2 || patterns["8"]["type"] != "tag" || patterns["9"]["type"] != "branch" {
		t.Fatal(writes, patterns)
	}
}
