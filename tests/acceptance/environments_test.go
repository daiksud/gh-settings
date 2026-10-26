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

func TestEnvironmentBinaryNameUpsertDefaultsAndRepeatWithoutWriteback(t *testing.T) {
	for _, mode := range []string{"create", "update", "actions", "unsupported", "readback"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			writes := 0
			exists := mode != "create" && mode != "actions"
			state := map[string]any{"id": 5, "name": "production", "protection_rules": []any{map[string]any{"type": "wait_timer", "wait_timer": 30}}, "deployment_branch_policy": nil}
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/current":
						fmt.Fprint(w, `{"id":71,"full_name":"owner/current","archived":false}`)
						return
					case "/repos/owner/current/environments":
						if exists {
							fmt.Fprint(w, `{"total_count":2,"environments":[{"id":5,"name":"production"},{"id":9,"name":"unmanaged"}]}`)
						} else {
							fmt.Fprint(w, `{"total_count":1,"environments":[{"id":9,"name":"unmanaged"}]}`)
						}
						return
					case "/repos/owner/current/environments/production":
						if mode == "readback" && writes > 0 {
							w.WriteHeader(403)
							fmt.Fprint(w, `{"message":"fixture unavailable"}`)
							return
						}
						json.NewEncoder(w).Encode(state)
						return
					case "/repos/owner/current/environments/production/deployment_protection_rules":
						if mode == "unsupported" {
							fmt.Fprint(w, `{"total_count":1,"custom_deployment_protection_rules":[{"id":1,"enabled":true}]}`)
						} else {
							fmt.Fprint(w, `{"total_count":0,"custom_deployment_protection_rules":[]}`)
						}
						return
					}
				}
				if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/environments/production" {
					t.Error("unmanaged/unrelated mutation or read", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				writes++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				want := map[string]any{"wait_timer": float64(0), "prevent_self_review": false, "reviewers": []any{}, "deployment_branch_policy": nil}
				if !reflect.DeepEqual(payload, want) {
					t.Error("incomplete/identity-bearing policy", payload)
				}
				state["protection_rules"], exists = []any{}, true
				json.NewEncoder(w).Encode(state)
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := labelIDFixtureEnv(fixture.cfg, mode == "actions")
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := "# preserve caller YAML\nschemaVersion: 1\nsettings:\n  environments:\n    - name: production # name is identity\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			validated, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"validate"}, false)
			if err != nil || validated.runErr != nil {
				t.Fatal(validated, err)
			}
			planned, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"plan"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "unsupported" {
				if planned.runErr == nil || !strings.Contains(planned.stderr.String(), "custom environment protection") {
					t.Fatal(planned)
				}
			} else {
				if planned.runErr != nil || !strings.Contains(planned.stdout.String(), "settings.environments:") || !strings.Contains(planned.stdout.String(), `"wait_timer":0`) {
					t.Fatal(planned)
				}
			}
			applied, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "unsupported" || mode == "readback" {
				if applied.runErr == nil {
					t.Fatal("failure became success", applied)
				}
				if mode == "readback" && !strings.Contains(applied.stderr.String(), "write may have applied") {
					t.Fatal(applied)
				}
			} else {
				if applied.runErr != nil || !strings.Contains(applied.stdout.String(), "Applied repository settings.") || strings.Contains(applied.stdout.String(), "Created") {
					t.Fatal(applied)
				}
				again, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply"}, false)
				if err != nil || again.runErr != nil {
					t.Fatal(again, err)
				}
				plan, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"plan"}, false)
				if err != nil || plan.runErr != nil || !strings.Contains(plan.stdout.String(), "No differences") {
					t.Fatal(plan, err)
				}
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || string(data) != input {
				t.Fatal("environment wrote ID or reformatted YAML", string(data), err)
			}
			mu.Lock()
			defer mu.Unlock()
			want := 1
			if mode == "unsupported" {
				want = 0
			}
			if writes != want {
				t.Fatal("repeated or unsafe upsert", writes, want)
			}
		})
	}
}
