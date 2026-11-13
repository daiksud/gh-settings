//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestEnvironmentModePreservationAndExplicitClear(t *testing.T) {
	for _, choice := range []string{"no_restriction", "protected_branches_only"} {
		t.Run(choice, func(t *testing.T) {
			var mu sync.Mutex
			var mode any = map[string]bool{"protected_branches": false, "custom_branch_policies": true}
			patterns := []map[string]any{{"id": 7, "type": "branch", "name": "PATTERN_CANARY"}}
			writes := []string{}
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				base := "/repos/owner/current/environments/production"
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, `{"id":71,"full_name":"owner/current","archived":false}`)
				case "/repos/owner/current/environments":
					fmt.Fprint(w, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`)
				case base:
					if r.Method == "PUT" {
						var sent map[string]any
						if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
							t.Error(err)
						}
						if len(patterns) != 0 {
							t.Error("policy PUT discarded omitted patterns")
						}
						mode = sent["deployment_branch_policy"]
						writes = append(writes, "PUT environment")
					} else if r.Method != "GET" {
						t.Error("unexpected method", r.Method)
					}
					json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "production", "protection_rules": []any{}, "can_admins_bypass": false, "deployment_branch_policy": mode})
				case base + "/deployment_protection_rules":
					fmt.Fprint(w, `{"total_count":0,"custom_deployment_protection_rules":[]}`)
				case base + "/deployment-branch-policies":
					if r.Method != "GET" {
						t.Error("unexpected pattern method", r.Method)
					}
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(patterns), "branch_policies": patterns})
				case base + "/deployment-branch-policies/7":
					if r.Method == "DELETE" {
						patterns = []map[string]any{}
						writes = append(writes, "DELETE pattern")
						w.WriteHeader(204)
					} else {
						json.NewEncoder(w).Encode(patterns[0])
					}
				default:
					t.Error("unexpected later operation", r.Method, r.URL.Path)
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
			input := func(mode, extra string) string {
				return repositoryInput("settings:\n  environments:\n    - name: production\n      deployment_branches_and_tags: " + mode + "\n" + extra)
			}
			blocked := input(choice, "    - name: zzz-later\n")
			if err := writeSettingsFile(fixture.work, blocked); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"plan"}, {"action", "plan", filepath.Join(fixture.root, "blocked.json")}, {"apply"}} {
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
				if err != nil || result.runErr == nil || !strings.Contains(result.stderr.String(), "omitted deployment patterns") || strings.Contains(result.stderr.String(), "PATTERN_CANARY") {
					t.Fatalf("%v: error=%v run=%v stdout=%q stderr=%q", args, err, result.runErr, result.stdout.String(), result.stderr.String())
				}
			}
			mu.Lock()
			noWrites := len(writes) == 0 && len(patterns) == 1
			mu.Unlock()
			if !noWrites {
				t.Fatal("rejected transition changed remote state")
			}
			if err := checkSettingsUnchanged(fixture.work, blocked); err != nil {
				t.Fatal(err)
			}
			for _, declaration := range []string{input("selected_branches_and_tags", "      deployment_branch_or_tag_rules: []\n"), input(choice, "")} {
				if err := writeSettingsFile(fixture.work, declaration); err != nil {
					t.Fatal(err)
				}
				for _, command := range []string{"plan", "apply", "plan", "apply"} {
					result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{command}, false)
					if err != nil || result.runErr != nil {
						t.Fatalf("%s: %v %v %q", command, err, result.runErr, result.stderr.String())
					}
				}
				if err := checkSettingsUnchanged(fixture.work, declaration); err != nil {
					t.Fatal(err)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if strings.Join(writes, ",") != "DELETE pattern,PUT environment" || len(patterns) != 0 {
				t.Fatal("explicit clear/mode order or repeat writes differ", writes, patterns)
			}
		})
	}
}
