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

const classicProtectionInput = "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    branch_protection_rules:\n      - branch_name_pattern: 'release/**'\n        require_conversation_resolution_before_merging: false\n        require_signed_commits: true\n        require_linear_history: true\n        allow_deletions: false\n"

func TestClassicProtectionValidatesLiteralControls(t *testing.T) {
	f := newActionsEventPoliciesFixture(t, "")
	f.write(t, classicProtectionInput)
	f.run(t, false, "validate")
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) != 0 {
		t.Fatal("validation contacted remote", f.calls)
	}
}

func processClassicTarget() map[string]any {
	return map[string]any{"id": "R_current", "nameWithOwner": "owner/current", "isPrivate": false, "isArchived": false, "owner": map[string]any{"__typename": "User"}, "squashMergeAllowed": true, "rebaseMergeAllowed": true}
}
func processClassicRule(id int, pattern string) map[string]any {
	return map[string]any{
		"id": fmt.Sprintf("B_%d", id), "databaseId": id, "pattern": pattern, "repository": processClassicTarget(),
		"requiresConversationResolution": false, "requiresCommitSignatures": false, "requiresLinearHistory": false, "allowsDeletions": false,
		"isAdminEnforced": true, "allowsForcePushes": false, "blocksCreations": false, "dismissesStaleReviews": true, "lockBranch": false, "lockAllowsFetchAndMerge": false, "requireLastPushApproval": true, "requiresApprovingReviews": true, "requiresCodeOwnerReviews": true, "requiresDeployments": false, "requiresStatusChecks": true, "requiresStrictStatusChecks": true, "restrictsPushes": false, "restrictsReviewDismissals": false,
		"requiredApprovingReviewCount": 2, "requiredDeploymentEnvironments": nil, "requiredStatusCheckContexts": []string{"build"}, "requiredStatusChecks": []any{map[string]any{"context": "build", "app": map[string]any{"id": "PRIVATE_APP_CANARY"}}},
	}
}
func processClassicPage(nodes []any) map[string]any {
	return map[string]any{"totalCount": len(nodes), "nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}}
}
func processClassicClone(v map[string]any) map[string]any {
	data, _ := json.Marshal(v)
	var result map[string]any
	_ = json.Unmarshal(data, &result)
	return result
}
func TestClassicProtectionExecutableLifecycle(t *testing.T) {
	for _, mode := range []string{"update", "create-order", "create-no-writeback", "actions-create", "create-partial-ack-error", "unavailable-export", "companion-drift", "lock"} {
		t.Run(mode, func(t *testing.T) {
			var mu sync.Mutex
			rules := []map[string]any{processClassicRule(7, "Release/**")}
			writes := []map[string]any{}
			denied := false
			partialErrorSent, followupAfterError := false, 0
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "POST" || r.URL.Path != "/graphql" {
					w.WriteHeader(403)
					fmt.Fprint(w, `{}`)
					return
				}
				var request struct {
					Query     string
					Variables map[string]any
				}
				if json.NewDecoder(r.Body).Decode(&request) != nil {
					t.Error("invalid request")
					w.WriteHeader(400)
					return
				}
				if !strings.Contains(request.Query, "ClassicProtection") {
					w.WriteHeader(403)
					fmt.Fprint(w, `{}`)
					return
				}
				if partialErrorSent {
					followupAfterError++
				}
				if strings.Contains(request.Query, "matchingRefs") {
					t.Error("inventoried branches instead of rules")
				}
				body := map[string]any{}
				var selected map[string]any
				for _, v := range rules {
					if v["id"] == request.Variables["id"] {
						selected = processClassicClone(v)
					}
				}
				switch {
				case strings.Contains(request.Query, "ClassicProtectionInventory"):
					if denied {
						w.WriteHeader(403)
						fmt.Fprint(w, `{}`)
						return
					}
					target := processClassicTarget()
					nodes := []any{}
					for _, v := range rules {
						nodes = append(nodes, v)
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
					if group == "" {
						t.Error("missing allowance group")
					}
					body["node"] = map[string]any{"__typename": "BranchProtectionRule", "id": selected["id"], "databaseId": selected["databaseId"], "repository": selected["repository"], group: processClassicPage([]any{})}
				case strings.Contains(request.Query, "ClassicProtectionRule"):
					if selected != nil {
						selected["__typename"] = "BranchProtectionRule"
					}
					body["node"] = selected
				case strings.Contains(request.Query, "mutation ClassicProtectionWrite"):
					input := request.Variables["input"].(map[string]any)
					writes = append(writes, input)
					mutation := "updateBranchProtectionRule"
					var rule map[string]any
					if input["repositoryId"] != nil {
						mutation = "createBranchProtectionRule"
						rule = processClassicRule(100+len(rules), input["pattern"].(string))
						rules = append(rules, rule)
					} else {
						for _, v := range rules {
							if v["id"] == input["branchProtectionRuleId"] {
								rule = v
							}
						}
					}
					if rule == nil {
						t.Error("mutation selected unknown identity")
						w.WriteHeader(400)
						return
					}
					for key, value := range input {
						if key != "repositoryId" && key != "branchProtectionRuleId" {
							rule[key] = value
						}
					}
					ack := map[string]any{}
					for _, key := range []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "repository"} {
						ack[key] = rule[key]
					}
					body[mutation] = map[string]any{"branchProtectionRule": ack}
					if mode == "create-partial-ack-error" {
						ack["repository"] = nil
						partialErrorSent = true
						json.NewEncoder(w).Encode(map[string]any{"data": body, "errors": []any{map[string]any{"message": "repository resolver unavailable", "path": []string{mutation, "branchProtectionRule", "repository"}}}})
						return
					}
					if mode == "companion-drift" {
						rule["requiredApprovingReviewCount"] = 3
					}
				default:
					t.Error("unexpected classic request", request.Query)
					w.WriteHeader(400)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"data": body})
			}, false, "/classic-protection")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := labelIDFixtureEnv(fixture.cfg, false)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			run := func(bad bool, args ...string) *executionResult {
				t.Helper()
				result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, args, false)
				if err != nil || (result.runErr != nil) != bad {
					t.Fatal(args, result.stdout.String(), result.stderr.String(), result.runErr, err)
				}
				return result
			}
			run(false, "export")
			path := filepath.Join(fixture.work, ".github/settings.yml")
			data, _ := os.ReadFile(path)
			for _, want := range []string{"branch_protection_rules:", "id: 7", "branch_name_pattern: Release/**", "require_signed_commits: false", "allow_deletions: false", "lock_branch: false"} {
				if !strings.Contains(string(data), want) {
					t.Fatal(want, string(data))
				}
			}
			if strings.Contains(string(data), "PRIVATE_APP") || strings.Contains(string(data), "required_approving") {
				t.Fatal("unowned policy exported", string(data))
			}
			if mode == "unavailable-export" {
				mu.Lock()
				denied = true
				mu.Unlock()
				result := run(false, "export")
				after, _ := os.ReadFile(path)
				if string(after) != string(data) || !strings.Contains(result.stderr.String(), "branch_protection_rules") {
					t.Fatal("unavailable family lost prior intent", string(after), result.stderr.String())
				}
				return
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: 'Release/**', require_signed_commits: true}]}}\n"
			if mode == "lock" {
				input = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: 'Release/**', lock_branch: true}]}}\n"
			}
			if strings.Contains(mode, "create") {
				input = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{branch_name_pattern: 'z/**', require_signed_commits: true}, {branch_name_pattern: 'z/*', allow_deletions: false}]}}\n"
			}
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			plan := run(false, "plan")
			if !strings.Contains(plan.stdout.String(), "branch_protection_rules:") {
				t.Fatal(plan.stdout.String())
			}
			report := filepath.Join(fixture.root, "classic.json")
			run(false, "action", "plan", report)
			reportData, _ := os.ReadFile(report)
			if strings.Contains(string(reportData), "PRIVATE_APP") || !strings.Contains(string(reportData), "branch_protection_rules") {
				t.Fatal(string(reportData))
			}
			run(false, "action", "plan", report, "status-only")
			env = append(env, "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=fixture-actor")
			privateSummary := filepath.Join(fixture.root, "classic-summary")
			run(false, "action", "summary", report, privateSummary, "status-only")
			privateData, _ := os.ReadFile(privateSummary)
			if strings.Contains(string(privateData), "Release") || strings.Contains(string(privateData), "z/**") || strings.Contains(string(privateData), "require_signed") {
				t.Fatal("status-only leaked", string(privateData))
			}
			args := []string{"apply"}
			if mode == "create-no-writeback" {
				args = append(args, "--no-writeback")
			}
			if mode == "actions-create" {
				env = append(env, "GITHUB_ACTIONS=true")
			}
			applied := run(mode == "companion-drift" || mode == "create-partial-ack-error", args...)
			mu.Lock()
			count, followups := len(writes), followupAfterError
			captured := append([]map[string]any{}, writes...)
			mu.Unlock()
			if mode == "create-partial-ack-error" {
				data, err := os.ReadFile(path)
				if err != nil || string(data) != input || count != 1 || followups != 0 || applied.stdout.Len() != 0 {
					t.Fatal("uncertain creation changed YAML or continued work", string(data), count, followups, err)
				}
				for _, want := range []string{"id 101", "B_101", "unconfirmed effects may remain", "repository resolver unavailable"} {
					if !strings.Contains(applied.stderr.String(), want) {
						t.Fatal("missing recovery or uncertainty", want, applied.stderr.String())
					}
				}
				return
			}
			if mode == "companion-drift" {
				if !strings.Contains(applied.stderr.String(), "unowned classic protection changed") || count != 1 {
					t.Fatal(applied.stderr.String(), count)
				}
				return
			}
			if mode == "lock" && !reflect.DeepEqual(captured, []map[string]any{{"branchProtectionRuleId": "B_7", "lockBranch": true}}) {
				t.Fatal(captured)
			}
			if mode == "update" && !reflect.DeepEqual(captured, []map[string]any{{"branchProtectionRuleId": "B_7", "requiresCommitSignatures": true}}) {
				t.Fatal(captured)
			}
			if strings.Contains(mode, "create") {
				if count != 2 || captured[0]["pattern"] != "z/**" || captured[1]["pattern"] != "z/*" {
					t.Fatal(captured)
				}
				data, _ := os.ReadFile(path)
				if mode == "create-no-writeback" || mode == "actions-create" {
					if string(data) != input {
						t.Fatal("writeback suppression failed", string(data))
					}
					run(true, "apply")
					mu.Lock()
					if len(writes) != count {
						t.Error("create collision mutated", writes)
					}
					mu.Unlock()
					return
				}
				if !strings.Contains(string(data), "id: 101") || !strings.Contains(string(data), "id: 102") {
					t.Fatal("missing guarded writeback", string(data))
				}
			}
			run(false, "apply")
			mu.Lock()
			if len(writes) != count {
				t.Error("repeat mutation", writes)
			}
			mu.Unlock()
		})
	}
}
