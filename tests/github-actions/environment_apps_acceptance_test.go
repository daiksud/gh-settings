//go:build acceptance

package githubactions

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeEnvironmentApps(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			const name = "ENVIRONMENT_NAME_CANARY"
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {environments: [{name: " + name + ", deployment_protection_rules: {custom_rules: [2]}}]}}\n"
			f.write(filepath.Join(f.caller, ".github/settings.yml"), input)
			state, err := os.MkdirTemp("/tmp", "ghs-apps-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(state) })
			listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			operations := []string{}
			app := func(id int64) map[string]any {
				return map[string]any{"id": id, "node_id": fmt.Sprint("PRIVATE_APP_NODE_", id), "slug": fmt.Sprint("private-app-slug-", id), "integration_url": fmt.Sprint("https://api.github.com/apps/custom-", id)}
			}
			rule := func(id int64) map[string]any {
				return map[string]any{"id": id + 100, "node_id": fmt.Sprint("PRIVATE_RULE_NODE_", id), "enabled": true, "app": app(id)}
			}
			rules := []any{rule(1)}
			path := "/repos/owner/config/environments/" + name
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token apps-explicit-token" {
					t.Error("escaped credential/target boundary")
					w.WriteHeader(400)
					return
				}
				if r.Method == "GET" {
					switch r.URL.Path {
					case "/repos/owner/config":
						fmt.Fprint(w, `{"id":71,"full_name":"owner/config","archived":false}`)
						return
					case "/repos/owner/config/environments":
						json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "environments": []any{map[string]any{"id": 7, "name": name}}})
						return
					case path:
						json.NewEncoder(w).Encode(map[string]any{"id": 7, "name": name, "protection_rules": []any{}, "can_admins_bypass": false, "deployment_branch_policy": nil})
						return
					case path + "/deployment_protection_rules":
						json.NewEncoder(w).Encode(map[string]any{"total_count": len(rules), "custom_deployment_protection_rules": rules})
						return
					case path + "/deployment_protection_rules/apps":
						json.NewEncoder(w).Encode(map[string]any{"total_count": 1, "available_custom_deployment_protection_rule_integrations": []any{app(2)}})
						return
					case path + "/deployment_protection_rules/102":
						if failure {
							w.WriteHeader(403)
							fmt.Fprint(w, `{}`)
						} else {
							json.NewEncoder(w).Encode(rule(2))
						}
						return
					}
				}
				if r.Method == "POST" && r.URL.Path == path+"/deployment_protection_rules" {
					operations = append(operations, "add")
					var body map[string]int64
					json.NewDecoder(r.Body).Decode(&body)
					if len(body) != 1 || body["integration_id"] != 2 {
						t.Error(body)
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
				t.Error("unexpected App request", r.Method, r.URL)
				w.WriteHeader(400)
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(state, "config.yml"), "http_unix_socket: "+filepath.Join(state, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = state, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			for _, mode := range []string{"detailed", "status-only"} {
				result := f.run(t, "plan", "apps-explicit-token", map[string]string{"github.action_ref": version, "inputs.plan-reporting": mode, "GH_DEBUG": "api"})
				if result.status != 0 || len(result.remainingReports) != 0 {
					t.Fatal(result)
				}
				if mode == "detailed" && !strings.Contains(result.summary, "custom_rules") {
					t.Fatal(result.summary)
				}
				if mode == "status-only" && (strings.Contains(result.summary+result.log, name) || strings.Contains(result.summary, "custom_rules")) {
					t.Fatal("private selection leaked", result)
				}
				if strings.Contains(result.summary+result.log, "PRIVATE_APP_NODE_") || strings.Contains(result.summary+result.log, "PRIVATE_RULE_NODE_") {
					t.Fatal("raw bindings leaked", result)
				}
			}
			result := f.run(t, "apply", "apps-explicit-token", map[string]string{"github.action_ref": version})
			if (result.status == 0) == failure || failure && len(result.outputs) != 0 {
				t.Fatal(result)
			}
			if failure && !strings.Contains(result.log, "binding rule ID 102") {
				t.Fatal("lost recovery", result.log)
			}
			if !failure {
				again := f.run(t, "apply", "apps-explicit-token", map[string]string{"github.action_ref": version})
				if again.status != 0 {
					t.Fatal(again)
				}
			}
			mu.Lock()
			want := "add,remove"
			if failure {
				want = "add"
			}
			if strings.Join(operations, ",") != want {
				t.Error(operations)
			}
			mu.Unlock()
			data, _ := os.ReadFile(filepath.Join(f.caller, ".github/settings.yml"))
			if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("App binding changed source identity")
			}
		})
	}
}
