//go:build acceptance

package acceptance

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestClassicPruneEmptyCollection(t *testing.T) {
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		if !isClassicProtectionFixtureRead(r) {
			t.Error("unexpected operation", r.Method, r.URL)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		target := processClassicTarget()
		target["branchProtectionRules"] = processClassicPage([]any{})
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": target}})
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := labelIDFixtureEnv(fixture.cfg, false)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsFile(fixture.work, "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: []}}\n"); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"plan", "apply", "apply"} {
		result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{command, "--prune", "branch-protection-rules"}, false)
		if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
			t.Fatal(command, result.runErr, result.stdout.String(), result.stderr.String(), err)
		}
		if command == "plan" && result.stdout.String() != "No differences for managed repository settings.\n" {
			t.Fatal(result.stdout.String())
		}
	}
}

func TestClassicPruneExecutableLifecycle(t *testing.T) {
	for _, mode := range []string{"delete-all", "retain-duplicate-pattern", "create-before-delete", "actions-create-prune", "omitted", "without-prune", "partial-ack", "new-arrival", "incomplete-global-preflight"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			rules := []map[string]any{processClassicRule(7, "PRIVATE_PATTERN"), processClassicRule(8, "PRIVATE_PATTERN")}
			events := []string{}
			inventories := 0
			applying := false
			unknown := false
			afterUnknown := 0
			var path string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if unknown {
					afterUnknown++
				}
				if r.Method != "POST" || r.URL.Path != "/graphql" {
					t.Error("unexpected request", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				var request struct {
					Query     string
					Variables map[string]any
				}
				if json.NewDecoder(r.Body).Decode(&request) != nil {
					t.Error("bad request")
					w.WriteHeader(400)
					return
				}
				body := map[string]any{}
				var selected map[string]any
				for _, rule := range rules {
					if rule["id"] == request.Variables["id"] {
						selected = processClassicClone(rule)
					}
				}
				switch {
				case strings.Contains(request.Query, "ClassicProtectionInventory"):
					inventories++
					if applying && mode == "new-arrival" && inventories == 2 {
						rules = append(rules, processClassicRule(9, "LATE_PATTERN"))
					}
					target := processClassicTarget()
					nodes := []any{}
					for _, rule := range rules {
						copy := processClassicClone(rule)
						if mode == "incomplete-global-preflight" {
							delete(copy, "requiredStatusChecks")
						}
						nodes = append(nodes, copy)
					}
					target["branchProtectionRules"] = processClassicPage(nodes)
					body["repository"] = target
				case strings.Contains(request.Query, "ClassicProtectionTarget"):
					body["repository"] = processClassicTarget()
				case strings.Contains(request.Query, "ClassicProtectionActors"):
					if selected == nil {
						t.Error("missing actor parent")
						w.WriteHeader(400)
						return
					}
					group := ""
					for _, key := range []string{"pushAllowances", "reviewDismissalAllowances", "bypassForcePushAllowances", "bypassPullRequestAllowances"} {
						if strings.Contains(request.Query, key+"(") {
							group = key
						}
					}
					body["node"] = map[string]any{"__typename": "BranchProtectionRule", "id": selected["id"], "databaseId": selected["databaseId"], "repository": selected["repository"], group: processClassicPage([]any{})}
				case strings.Contains(request.Query, "ClassicProtectionRule"):
					if selected != nil {
						selected["__typename"] = "BranchProtectionRule"
					}
					body["node"] = selected
				case strings.Contains(request.Query, "mutation ClassicProtectionWrite"):
					input := request.Variables["input"].(map[string]any)
					if len(input) != 2 || input["repositoryId"] != "R_current" || input["pattern"] != "new/**" {
						t.Error("unexpected creation payload", input)
					}
					rule := processClassicRule(101, "new/**")
					rules = append(rules, rule)
					events = append(events, "create:101")
					ack := map[string]any{}
					for _, key := range []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "requiresDeployments", "requiredDeploymentEnvironments", "repository"} {
						ack[key] = rule[key]
					}
					body["createBranchProtectionRule"] = map[string]any{"branchProtectionRule": ack}
				case strings.Contains(request.Query, "mutation ClassicProtectionDelete"):
					input := request.Variables["input"].(map[string]any)
					if len(input) != 1 {
						t.Error("unexpected delete companions", input)
					}
					if mode == "create-before-delete" {
						data, _ := os.ReadFile(path)
						if !strings.Contains(string(data), "id: 101") {
							t.Error("deletion preceded required ID writeback", string(data))
						}
					}
					node, _ := input["branchProtectionRuleId"].(string)
					events = append(events, "delete:"+node)
					for i, rule := range rules {
						if rule["id"] == node {
							rules = append(rules[:i], rules[i+1:]...)
							break
						}
					}
					body["deleteBranchProtectionRule"] = map[string]any{"__typename": "DeleteBranchProtectionRulePayload"}
					if mode == "partial-ack" {
						unknown = true
						json.NewEncoder(w).Encode(map[string]any{"data": body, "errors": []any{map[string]any{"message": "delete resolver failed", "path": []string{"deleteBranchProtectionRule"}}}})
						return
					}
				default:
					t.Error("unexpected GraphQL query", request.Query)
					w.WriteHeader(400)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"data": body})
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := append(labelIDFixtureEnv(fixture.cfg, false), "GH_DEBUG=api")
			if mode == "actions-create-prune" {
				env = append(env, "GITHUB_ACTIONS=true")
			}
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: []}}\n"
			switch mode {
			case "retain-duplicate-pattern":
				input = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: PRIVATE_PATTERN}]}}\n"
			case "create-before-delete", "actions-create-prune":
				input = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{branch_name_pattern: 'new/**'}]}}\n"
			case "omitted":
				input = "version: daiksud/gh-settings@v0\nrepository: {}\n"
			case "incomplete-global-preflight":
				input = "version: daiksud/gh-settings@v0\nrepository: {about: {description: changed}, settings: {branch_protection_rules: []}}\n"
			}
			path = filepath.Join(fixture.work, ".github/settings.yml")
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			run := func(bad bool, args ...string) *executionResult {
				t.Helper()
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
				if err != nil || (result.runErr != nil) != bad {
					t.Fatal(args, result.stdout.String(), result.stderr.String(), err)
				}
				if strings.Contains(result.stdout.String()+result.stderr.String(), "PRIVATE_APP_CANARY") {
					t.Fatal("private raw evidence leaked")
				}
				return result
			}
			args := []string{"--prune", "branch-protection-rules"}
			if mode == "without-prune" {
				args = nil
			}
			if mode != "incomplete-global-preflight" {
				plan := run(false, append([]string{"plan"}, args...)...)
				if mode != "omitted" && mode != "without-prune" && !strings.Contains(plan.stdout.String(), "removes the whole rule") {
					t.Fatal(plan.stdout.String())
				}
			}
			mu.Lock()
			if len(events) != 0 {
				t.Fatal("plan mutated", events)
			}
			inventories = 0
			applying = true
			mu.Unlock()
			applyArgs := []string{"apply", "--prune", "all"}
			if mode == "without-prune" {
				applyArgs = []string{"apply"}
			}
			result := run(mode == "partial-ack" || mode == "incomplete-global-preflight", applyArgs...)
			mu.Lock()
			captured := append([]string{}, events...)
			remaining := len(rules)
			followups := afterUnknown
			mu.Unlock()
			switch mode {
			case "delete-all":
				if !reflect.DeepEqual(captured, []string{"delete:B_7", "delete:B_8"}) || remaining != 0 {
					t.Fatal(captured, remaining)
				}
			case "retain-duplicate-pattern":
				if !reflect.DeepEqual(captured, []string{"delete:B_8"}) || remaining != 1 {
					t.Fatal(captured, remaining)
				}
			case "create-before-delete", "actions-create-prune":
				if !reflect.DeepEqual(captured, []string{"create:101", "delete:B_7", "delete:B_8"}) || remaining != 1 {
					t.Fatal(captured, remaining)
				}
			case "omitted", "without-prune", "incomplete-global-preflight":
				if len(captured) != 0 || remaining != 2 {
					t.Fatal(captured, remaining)
				}
			case "new-arrival":
				if !reflect.DeepEqual(captured, []string{"delete:B_7", "delete:B_8"}) || remaining != 1 {
					t.Fatal(captured, remaining)
				}
			case "partial-ack":
				if !reflect.DeepEqual(captured, []string{"delete:B_7"}) || followups != 0 || !strings.Contains(result.stderr.String(), "unconfirmed effects may remain") || !strings.Contains(result.stderr.String(), "id 7 node \"B_7\"") {
					t.Fatal(captured, followups, result.stderr.String())
				}
			}
			data, _ := os.ReadFile(path)
			if mode != "create-before-delete" && string(data) != input {
				t.Fatal("deletion changed YAML", string(data))
			}
			if mode == "delete-all" || mode == "retain-duplicate-pattern" || mode == "create-before-delete" {
				run(false, applyArgs...)
				mu.Lock()
				if !reflect.DeepEqual(events, captured) {
					t.Error("repeat mutation", events)
				}
				mu.Unlock()
			}
		})
	}
}
