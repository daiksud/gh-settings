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

func TestEnvironmentPruneBinary(t *testing.T) {
	for _, mode := range []string{"retained", "empty", "all", "omitted", "custom-denied", "late-custom", "recreated", "variables-prune"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			events := []string{}
			deleted := false
			customReads := 0
			variableDeleted := false
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				base := "/repos/owner/current/environments/"
				if r.Method == "DELETE" {
					events = append(events, r.URL.Path)
					switch r.URL.Path {
					case base + "old":
						deleted = true
					case base + "kept-vars/variables/REMOVE":
						variableDeleted = true
					default:
						t.Error("deleted retained parent", r.URL)
						w.WriteHeader(400)
						return
					}
					w.WriteHeader(204)
					return
				}
				if r.Method != "GET" {
					t.Error("unexpected mutation", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, `{"id":71,"full_name":"owner/current","archived":false}`)
				case "/repos/owner/current/environments":
					entries := []map[string]any{}
					if mode != "empty" {
						entries = append(entries, map[string]any{"id": 7, "name": "kept-policy"}, map[string]any{"id": 8, "name": "kept-vars"})
					}
					if !deleted || mode == "recreated" {
						id := 9
						if deleted {
							id = 10
						}
						entries = append(entries, map[string]any{"id": id, "name": "old"})
					}
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(entries), "environments": entries})
				case base + "old", base + "kept-policy", base + "kept-vars":
					name := strings.TrimPrefix(r.URL.Path, base)
					id := 7
					if name == "old" {
						id = 9
					}
					if name == "kept-vars" {
						id = 8
					}
					json.NewEncoder(w).Encode(map[string]any{"id": id, "name": name, "protection_rules": []any{}, "deployment_branch_policy": nil, "can_admins_bypass": false})
				case base + "old/deployment_protection_rules", base + "kept-policy/deployment_protection_rules":
					if strings.Contains(r.URL.Path, "/old/") {
						customReads++
						if mode == "custom-denied" || mode == "late-custom" && customReads >= 5 {
							w.WriteHeader(403)
							fmt.Fprint(w, `{}`)
							return
						}
					}
					fmt.Fprint(w, `{"total_count":0,"custom_deployment_protection_rules":[]}`)
				case base + "old/variables":
					fmt.Fprint(w, `{"total_count":1,"variables":[{"name":"PRIVATE_CHILD","value":"PRIVATE_VARIABLE_CANARY"}]}`)
				case base + "old/secrets":
					fmt.Fprint(w, `{"total_count":1,"secrets":[{"name":"SECRET_METADATA_CANARY","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]}`)
				case base + "kept-vars/variables":
					if mode == "variables-prune" && !variableDeleted {
						fmt.Fprint(w, `{"total_count":1,"variables":[{"name":"REMOVE","value":"old"}]}`)
					} else {
						fmt.Fprint(w, `{"total_count":0,"variables":[]}`)
					}
				case base + "kept-vars/variables/REMOVE":
					if variableDeleted {
						w.WriteHeader(404)
						fmt.Fprint(w, `{}`)
					} else {
						fmt.Fprint(w, `{"name":"REMOVE","value":"old"}`)
					}
				default:
					t.Error("unexpected read", r.URL)
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
			input := repositoryInput("settings: {environments: [{name: kept-policy}, {name: kept-vars, variables: []}]}\n")
			if mode == "empty" {
				input = repositoryInput("settings: {environments: []}\n")
			}
			if mode == "omitted" {
				input = "version: daiksud/gh-settings@v0\n"
			}
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			scope := "environments"
			if mode == "all" || mode == "omitted" {
				scope = "all"
			}
			if mode == "variables-prune" {
				scope = "environments,environment-variables"
			}
			// Plan has no mutations and must not serialize child deletion evidence.
			plan, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"plan", "--prune", scope}, false)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "custom-denied" {
				if plan.runErr == nil {
					t.Fatal("denied candidate planned")
				}
			} else if plan.runErr != nil {
				t.Fatal(plan.stdout.String(), plan.stderr.String(), plan.runErr)
			}
			if mode != "omitted" && mode != "custom-denied" && !strings.Contains(plan.stdout.String(), "settings.environments: delete") {
				t.Fatal("missing deletion", plan.stdout.String())
			}
			if strings.Contains(plan.stdout.String()+plan.stderr.String(), "CANARY") {
				t.Fatal("private child evidence leaked")
			}
			if len(events) != 0 {
				t.Fatal(events)
			}
			// Reset the simulated custom-rule race to the apply-only timeline: two
			// acquisition reads, two collection recheck reads, then last-delete reads.
			mu.Lock()
			customReads = 0
			mu.Unlock()
			applied, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply", "--prune", scope}, false)
			if err != nil {
				t.Fatal(err)
			}
			bad := mode == "custom-denied" || mode == "late-custom" || mode == "recreated"
			if (applied.runErr != nil) != bad {
				t.Fatal(applied.stdout.String(), applied.stderr.String(), applied.runErr)
			}
			if strings.Contains(applied.stdout.String()+applied.stderr.String(), "CANARY") {
				t.Fatal("private child evidence leaked")
			}
			mu.Lock()
			count := len(events)
			ordered := strings.Join(events, ",")
			mu.Unlock()
			if mode == "omitted" || mode == "custom-denied" || mode == "late-custom" {
				if count != 0 {
					t.Fatal(events)
				}
			} else if mode == "variables-prune" {
				if ordered != "/repos/owner/current/environments/kept-vars/variables/REMOVE,/repos/owner/current/environments/old" {
					t.Fatal(events)
				}
			} else if count != 1 {
				t.Fatal(events)
			}
			if !bad {
				again, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply", "--prune", scope}, false)
				if err != nil || again.runErr != nil {
					t.Fatal(again.stderr.String(), err)
				}
				mu.Lock()
				if len(events) != count {
					t.Error("repeat delete", events)
				}
				mu.Unlock()
			}
			data, _ := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if string(data) != input {
				t.Fatal("YAML rewritten")
			}
		})
	}
}
