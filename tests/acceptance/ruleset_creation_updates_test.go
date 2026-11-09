//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// Each mutation acknowledges the request independently of persisted GET state.
// A successful acknowledgment cannot hide a retained restriction or sync flag.
func TestRulesetCreationUpdatesExecutableLifecycle(t *testing.T) {
	for _, target := range []string{"branch", "tag"} {
		t.Run(target, func(t *testing.T) {
			var mu sync.Mutex
			var state map[string]any
			json.Unmarshal([]byte(acceptanceRuleset), &state)
			state["target"] = target
			state["conditions"] = map[string]any{"ref_name": map[string]any{"include": []any{"~ALL"}, "exclude": []any{}}}
			state["rules"] = []any{map[string]any{"type": "creation"}, map[string]any{"type": "deletion"}, map[string]any{"type": "update", "parameters": map[string]any{"update_allows_fetch_and_merge": true}}}
			writes, reads := 0, 0
			ignore := false
			expectedRules := []any{map[string]any{"type": "creation"}, map[string]any{"type": "deletion"}, map[string]any{"type": "update", "parameters": map[string]any{"update_allows_fetch_and_merge": false}}}
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/current/rulesets":
					if r.Method != "GET" {
						t.Error("unexpected write", r.Method)
						w.WriteHeader(400)
						return
					}
					fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
				case "/repos/owner/current/rulesets/7":
					if r.Method == "GET" {
						reads++
						json.NewEncoder(w).Encode(state)
						return
					}
					if r.Method != "PUT" {
						t.Error("unexpected write", r.Method)
						w.WriteHeader(400)
						return
					}
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					want := map[string]any{"name": "basic", "target": target, "enforcement": "active", "bypass_actors": []any{}, "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~ALL"}, "exclude": []any{}}}, "rules": expectedRules}
					if !reflect.DeepEqual(payload, want) {
						t.Errorf("payload=%v want=%v", payload, want)
					}
					writes++
					response := map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}
					for k, v := range payload {
						response[k] = v
						if !ignore {
							state[k] = v
						}
					}
					json.NewEncoder(w).Encode(response)
				default:
					t.Error("unexpected request", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			run := func(command string, success bool, args ...string) string {
				t.Helper()
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, append([]string{command}, args...), false)
				if err != nil || (result.runErr == nil) != success {
					t.Fatalf("%s err=%v run=%v out=%s stderr=%s", command, err, result.runErr, result.stdout.String(), result.stderr.String())
				}
				return result.stdout.String()
			}
			run("export", true, "--rulesets")
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "restrict_creations: true") || !strings.Contains(string(data), "allow_fetch_and_merge: true") {
				t.Fatal(string(data))
			}
			run("validate", true)
			if out := run("plan", true); !strings.Contains(out, "No differences") {
				t.Fatal(out)
			}
			targetYAML := "target_branches: {include_all_branches: true}"
			if target == "tag" {
				targetYAML = "target_tags: {include_all_tags: true}"
			}
			intent := func(rules string) {
				t.Helper()
				if err := writeSettingsFile(fixture.work, repositoryInput("settings: {rulesets: [{id: 7, ruleset_name: basic, enforcement_status: active, "+targetYAML+", rules: "+rules+"}]}\n")); err != nil {
					t.Fatal(err)
				}
			}
			intent("{restrict_creations: true, restrict_deletions: true, restrict_updates: {}}")
			run("validate", true)
			if out := run("plan", true); !strings.Contains(out, "update") {
				t.Fatal(out)
			}
			mu.Lock()
			before := reads
			mu.Unlock()
			run("apply", true)
			mu.Lock()
			if reads-before != 3 || writes != 1 {
				t.Error("preflight plus independent GET required", reads-before, writes)
			}
			mu.Unlock()
			if out := run("plan", true); !strings.Contains(out, "No differences") {
				t.Fatal(out)
			}
			run("apply", true)
			mu.Lock()
			if writes != 1 {
				t.Error("repeat apply wrote", writes)
			}
			ignore = true
			expectedRules = []any{map[string]any{"type": "deletion"}}
			mu.Unlock()
			intent("{restrict_deletions: true}")
			run("apply", false) // ack says removed; independent GET retains both protections
			mu.Lock()
			ignore = false
			mu.Unlock()
			run("apply", true)
			if out := run("plan", true); !strings.Contains(out, "No differences") {
				t.Fatal(out)
			}
			run("apply", true)
			mu.Lock()
			if writes != 3 {
				t.Error("expected successful update, rejected readback, removal only", writes)
			}
			mu.Unlock()
		})
	}
}
