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

func TestClassicStatusChecksProcess(t *testing.T) {
	for _, mode := range []string{"update", "create-post-app-error"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			var rule map[string]any
			if mode == "update" {
				rule = processClassicRule(7, "main")
				rule["requiresStatusChecks"] = false
				rule["requiredStatusCheckContexts"] = nil
				rule["requiredStatusChecks"] = nil
			}
			writes, apps, calls := 0, 0, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls++
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "POST" || r.URL.Path != "/graphql" {
					w.WriteHeader(403)
					fmt.Fprint(w, "{}")
					return
				}
				var request struct {
					Query     string
					Variables map[string]any
				}
				if json.NewDecoder(r.Body).Decode(&request) != nil {
					t.Error("invalid query")
					w.WriteHeader(400)
					return
				}
				body := map[string]any{}
				switch {
				case strings.Contains(request.Query, "ClassicStatusApp"):
					apps++
					if request.Variables["id"] != "APP_status" {
						t.Error("wrong App selector")
					}
					body["node"] = map[string]any{"__typename": "App", "id": "APP_status"}
					if mode == "create-post-app-error" && writes > 0 {
						json.NewEncoder(w).Encode(map[string]any{"data": body, "errors": []any{map[string]any{"message": "PRIVATE_APP_RESOLVER"}}})
						return
					}
				case strings.Contains(request.Query, "ClassicProtectionInventory"):
					target := processClassicTarget()
					nodes := []any{}
					if rule != nil {
						nodes = append(nodes, rule)
					}
					target["branchProtectionRules"] = processClassicPage(nodes)
					body["repository"] = target
				case strings.Contains(request.Query, "ClassicProtectionTarget"):
					body["repository"] = processClassicTarget()
				case strings.Contains(request.Query, "ClassicProtectionActors"):
					group := ""
					for _, name := range []string{"pushAllowances", "reviewDismissalAllowances", "bypassForcePushAllowances", "bypassPullRequestAllowances"} {
						if strings.Contains(request.Query, name+"(") {
							group = name
						}
					}
					body["node"] = map[string]any{"__typename": "BranchProtectionRule", "id": rule["id"], "databaseId": rule["databaseId"], "repository": processClassicTarget(), group: processClassicPage([]any{})}
				case strings.Contains(request.Query, "ClassicProtectionRule"):
					node := processClassicClone(rule)
					node["__typename"] = "BranchProtectionRule"
					body["node"] = node
				case strings.Contains(request.Query, "mutation ClassicProtectionWrite"):
					writes++
					input := request.Variables["input"].(map[string]any)
					if input["requiresStatusChecks"] != true || input["requiresStrictStatusChecks"] != false || input["requiredStatusCheckContexts"] != nil {
						t.Error("incomplete/legacy activation", input)
					}
					mutation := "updateBranchProtectionRule"
					wantSize := 4
					if rule == nil {
						rule = processClassicRule(100, "main")
						mutation = "createBranchProtectionRule"
						wantSize = 5
					}
					if len(input) != wantSize {
						t.Error("unowned input", input)
					}
					checks, ok := input["requiredStatusChecks"].([]any)
					if !ok || len(checks) != 2 {
						t.Error("incomplete checks")
						w.WriteHeader(400)
						return
					}
					contexts, raw := []string{}, []any{}
					for _, item := range checks {
						v := item.(map[string]any)
						context := v["context"].(string)
						source, ok := v["appId"].(string)
						if !ok || source == "" {
							t.Error("implicit App provider")
						}
						contexts = append(contexts, context)
						var app any
						if source != "any" {
							app = map[string]any{"id": source}
						}
						raw = append(raw, map[string]any{"context": context, "app": app})
					}
					rule["requiresStatusChecks"], rule["requiresStrictStatusChecks"] = true, false
					rule["requiredStatusCheckContexts"], rule["requiredStatusChecks"] = contexts, raw
					ack := map[string]any{}
					for _, key := range []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced", "requiresDeployments", "requiredDeploymentEnvironments", "requiresStatusChecks", "requiresStrictStatusChecks", "requiredStatusCheckContexts", "requiredStatusChecks", "repository"} {
						ack[key] = rule[key]
					}
					body[mutation] = map[string]any{"branchProtectionRule": ack}
				default:
					w.WriteHeader(403)
					fmt.Fprint(w, "{}")
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"data": body})
			}, false, "/classic-protection")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := append(labelIDFixtureEnv(fixture.cfg, false), "GH_DEBUG=api")
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			run := func(bad bool, args ...string) *executionResult {
				t.Helper()
				r, e := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
				if e != nil || (r.runErr != nil) != bad {
					t.Fatal(args, e, r.runErr, r.stderr.String())
				}
				if strings.Contains(r.stdout.String()+r.stderr.String(), "PRIVATE_APP_RESOLVER") {
					t.Fatal("upstream source error leaked")
				}
				return r
			}
			run(false, "export")
			if mode == "update" {
				data, _ := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
				if !strings.Contains(string(data), "require_status_checks_to_pass_before_merging:") || !strings.Contains(string(data), "enabled: false") {
					t.Fatal(string(data))
				}
			}
			id := ""
			if mode == "update" {
				id = "id: 7, "
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{" + id + "branch_name_pattern: main, require_status_checks_to_pass_before_merging: {enabled: true, require_branches_to_be_up_to_date_before_merging: false, status_checks: [{name: screening, accept_updates_from: APP_status}, {name: legacy, accept_updates_from: any}]}}]}}\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			beforeCalls := calls
			mu.Unlock()
			run(false, "validate")
			mu.Lock()
			if calls != beforeCalls {
				t.Error("validation contacted API")
			}
			mu.Unlock()
			if !strings.Contains(run(false, "plan").stdout.String(), "require_status_checks_to_pass_before_merging") {
				t.Fatal("missing active policy plan")
			}
			result := run(mode != "update", "apply")
			if mode != "update" {
				data, _ := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
				if string(data) != input || !strings.Contains(result.stderr.String(), "id 100") || !strings.Contains(result.stderr.String(), "unconfirmed effects may remain") {
					t.Fatal(string(data), result.stderr.String())
				}
				return
			}
			run(false, "apply")
			mu.Lock()
			defer mu.Unlock()
			if writes != 1 || apps != 2 {
				t.Fatal("repeat was not lookup-free noop", writes, apps)
			}
		})
	}
}
