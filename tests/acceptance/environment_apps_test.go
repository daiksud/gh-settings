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
	"sync/atomic"
	"testing"
)

func TestEnvironmentAppsBinaryValidationAndExport(t *testing.T) {
	var unavailable atomic.Bool
	f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/graphql" && r.Method == "POST" {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"message":"fixture unavailable"}`)
			return
		}
		if r.Method != "GET" {
			t.Error("unexpected mutation", r.Method)
			w.WriteHeader(400)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current/environments":
			fmt.Fprint(w, `{"total_count":1,"environments":[{"id":7,"name":"production"}]}`)
		case "/repos/owner/current/environments/production":
			fmt.Fprint(w, `{"id":7,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":false}`)
		case "/repos/owner/current/environments/production/deployment_protection_rules":
			if unavailable.Load() {
				fmt.Fprint(w, `{"total_count":1,"custom_deployment_protection_rules":[{"id":91,"enabled":true,"private":"CUSTOM_RAW_CANARY"}]}`)
				return
			}
			fmt.Fprint(w, `{"total_count":1,"custom_deployment_protection_rules":[{"id":91,"node_id":"rule91","enabled":true,"app":{"id":101,"node_id":"app101","slug":"custom-app","integration_url":"https://api.github.com/apps/custom-app"}}]}`)
		default:
			w.WriteHeader(403)
			fmt.Fprint(w, `{"message":"fixture unavailable"}`)
		}
	}, false, "/environments")
	if err != nil {
		t.Fatal(err)
	}
	defer f.close()
	env := labelIDFixtureEnv(f.cfg, false)
	if err := setupOrdinaryOrigin(f.work, env); err != nil {
		t.Fatal(err)
	}
	input := repositoryInput("settings: {environments: [{name: production, deployment_protection_rules: {custom_rules: [101]}}]}\n")
	t.Run("validate", func(t *testing.T) {
		if err := writeSettingsFile(f.work, input); err != nil {
			t.Fatal(err)
		}
		got, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"validate"}, false)
		if err != nil || got.runErr != nil {
			t.Fatal(got, err)
		}
	})
	t.Run("export", func(t *testing.T) {
		if err := writeSettingsFile(f.work, repositoryInput("{}\n")); err != nil {
			t.Fatal(err)
		}
		got, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"export"}, false)
		if err != nil || got.runErr != nil {
			t.Fatal(got, err)
		}
		data, err := os.ReadFile(filepath.Join(f.work, ".github/settings.yml"))
		if err != nil || !strings.Contains(string(data), "custom_rules:") || !strings.Contains(string(data), "101") || strings.Contains(string(data), "rule91") {
			t.Fatalf("complete App selection missing: %s; %v", data, err)
		}
	})
	t.Run("unavailable-retains-complete-family-without-debug-leak", func(t *testing.T) {
		unavailable.Store(true)
		prior := repositoryInput("settings: {environments: [{name: production, deployment_protection_rules: {wait_timer: 7, custom_rules: [101]}}]}\n")
		if err := writeSettingsFile(f.work, prior); err != nil {
			t.Fatal(err)
		}
		got, err := executeCLI(acceptanceBinary(t), f.work, f.root, append(env, "GH_DEBUG=api"), []string{"export"}, false)
		if err != nil || got.runErr != nil || strings.Contains(got.stderr.String()+got.stdout.String(), "CUSTOM_RAW_CANARY") {
			t.Fatal(got, err)
		}
		data, err := os.ReadFile(filepath.Join(f.work, ".github/settings.yml"))
		if err != nil || !strings.Contains(string(data), "wait_timer: 7") || !strings.Contains(string(data), "101") {
			t.Fatal(string(data), err)
		}
	})
}

func TestEnvironmentAppsBinaryReconciliation(t *testing.T) {
	for _, mode := range []string{"replace", "readback-failure", "parent-update-app-rejected", "omitted"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			rule := func(id int64) map[string]any {
				return map[string]any{"id": id + 100, "node_id": fmt.Sprint("rule", id), "enabled": true, "app": map[string]any{"id": id, "node_id": fmt.Sprint("app", id), "slug": fmt.Sprint("custom-", id), "integration_url": fmt.Sprint("https://api.github.com/apps/custom-", id)}}
			}
			rules := []map[string]any{rule(1)}
			operations := []string{}
			timer := 0
			parentUpdated, updatedParentRead := false, false
			path := "/repos/owner/current/environments/production"
			f, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				parent := func() {
					protection := []any{}
					if timer > 0 {
						protection = append(protection, map[string]any{"type": "wait_timer", "wait_timer": timer})
					}
					json.NewEncoder(w).Encode(map[string]any{"id": 7, "name": "production", "protection_rules": protection, "deployment_branch_policy": nil, "can_admins_bypass": false})
				}
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/current":
						fmt.Fprint(w, `{"id":71,"full_name":"owner/current","archived":false}`)
						return
					case "/repos/owner/current/environments":
						fmt.Fprint(w, `{"total_count":1,"environments":[{"id":7,"name":"production"}]}`)
						return
					case path:
						if parentUpdated && timer == 5 {
							updatedParentRead = true
						}
						parent()
						return
					case path + "/deployment_protection_rules":
						json.NewEncoder(w).Encode(map[string]any{"total_count": len(rules), "custom_deployment_protection_rules": rules})
						return
					case path + "/deployment_protection_rules/apps":
						json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "available_custom_deployment_protection_rule_integrations": []any{rule(2)["app"]}})
						return
					case path + "/deployment_protection_rules/102":
						if mode == "readback-failure" {
							w.WriteHeader(403)
							fmt.Fprint(w, `{}`)
						} else {
							json.NewEncoder(w).Encode(rule(2))
						}
						return
					}
				}
				if r.Method == "PUT" && r.URL.Path == path {
					operations = append(operations, "parent")
					var body map[string]any
					json.NewDecoder(r.Body).Decode(&body)
					if _, ok := body["custom_rules"]; ok {
						t.Error("custom payload leaked", body)
					}
					if mode == "parent-update-app-rejected" && body["wait_timer"] != float64(5) {
						t.Error("parent update did not request the desired wait timer", body)
					}
					timer = 5
					parentUpdated = true
					parent()
					return
				}
				if r.Method == "POST" && r.URL.Path == path+"/deployment_protection_rules" {
					operations = append(operations, "add")
					var body map[string]int64
					json.NewDecoder(r.Body).Decode(&body)
					if len(body) != 1 || body["integration_id"] != 2 {
						t.Error(body)
					}
					if mode == "parent-update-app-rejected" {
						if !updatedParentRead {
							t.Error("App enable attempted before updated parent readback")
						}
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"fixture rejected App enable"}`)
						return
					}
					rules = append(rules, rule(2))
					w.WriteHeader(201)
					json.NewEncoder(w).Encode(rule(2))
					return
				}
				if r.Method == "DELETE" && r.URL.Path == path+"/deployment_protection_rules/101" {
					operations = append(operations, "remove")
					rules = rules[1:]
					w.WriteHeader(204)
					return
				}
				t.Error("unrelated request", r.Method, r.URL)
				w.WriteHeader(400)
			}, false, "/environments")
			if err != nil {
				t.Fatal(err)
			}
			defer f.close()
			env := labelIDFixtureEnv(f.cfg, false)
			if err := setupOrdinaryOrigin(f.work, env); err != nil {
				t.Fatal(err)
			}
			controls := "custom_rules: [2]"
			if mode == "omitted" {
				controls = "wait_timer: 5"
			} else if mode == "parent-update-app-rejected" {
				controls = "wait_timer: 5, custom_rules: [2]"
			}
			input := repositoryInput("settings: {environments: [{name: production, deployment_protection_rules: {" + controls + "}}]}\n")
			if err := writeSettingsFile(f.work, input); err != nil {
				t.Fatal(err)
			}
			planned, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"plan"}, false)
			if err != nil || planned.runErr != nil || !strings.Contains(planned.stdout.String(), "settings.environments: update") {
				t.Fatal(planned, err)
			}
			applied, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"apply"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "parent-update-app-rejected" {
				if applied.runErr == nil || !strings.Contains(applied.stderr.String(), "earlier successful writes remain applied") || !strings.Contains(applied.stderr.String(), "enable custom environment protection App 2") || !strings.Contains(applied.stderr.String(), "write may have applied") {
					t.Fatal(applied)
				}
			} else if mode == "readback-failure" {
				if applied.runErr == nil || !strings.Contains(applied.stderr.String(), "binding rule ID 102") {
					t.Fatal(applied)
				}
			} else {
				if applied.runErr != nil {
					t.Fatal(applied)
				}
				again, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"apply"}, false)
				if err != nil || again.runErr != nil {
					t.Fatal(again, err)
				}
				plan, err := executeCLI(acceptanceBinary(t), f.work, f.root, env, []string{"plan"}, false)
				if err != nil || plan.runErr != nil || !strings.Contains(plan.stdout.String(), "No differences") {
					t.Fatal(plan, err)
				}
			}
			data, _ := os.ReadFile(filepath.Join(f.work, ".github/settings.yml"))
			if string(data) != input {
				t.Fatal("binding IDs wrote back to YAML", string(data))
			}
			mu.Lock()
			defer mu.Unlock()
			want := "add,remove"
			if mode == "readback-failure" {
				want = "add"
			}
			if mode == "omitted" {
				want = "parent"
			}
			if mode == "parent-update-app-rejected" {
				want = "parent,add"
				if timer != 5 || !reflect.DeepEqual(rules, []map[string]any{rule(1)}) {
					t.Fatal("parent update or original binding was not retained", timer, rules)
				}
			}
			if strings.Join(operations, ",") != want {
				t.Fatal(operations)
			}
		})
	}
}
