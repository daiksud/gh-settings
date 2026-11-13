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

func TestBuiltCompositeClassicStatusChecks(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, mode := range []string{"matching", "changed", "post-App-failure"} {
		t.Run(mode, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			f.base["GITHUB_ACTIONS"], f.base["GH_DEBUG"] = "true", "api"
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: main, require_status_checks_to_pass_before_merging: {enabled: true, require_branches_to_be_up_to_date_before_merging: false, status_checks: [{name: CHECK_NAME_CANARY, accept_updates_from: APP_SOURCE_CANARY}]}}]}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			target := map[string]any{"id": "R_config", "nameWithOwner": "owner/config", "isArchived": false, "isPrivate": false, "owner": map[string]any{"__typename": "User"}, "squashMergeAllowed": true, "rebaseMergeAllowed": true}
			rule := map[string]any{"id": "B_7", "databaseId": 7, "pattern": "main", "repository": target, "requiresConversationResolution": false, "requiresCommitSignatures": false, "requiresLinearHistory": false, "allowsDeletions": false, "lockBranch": false, "isAdminEnforced": false, "allowsForcePushes": false, "blocksCreations": false, "dismissesStaleReviews": false, "lockAllowsFetchAndMerge": false, "requireLastPushApproval": false, "requiresApprovingReviews": false, "requiresCodeOwnerReviews": false, "requiresDeployments": false, "requiresStatusChecks": true, "requiresStrictStatusChecks": mode != "matching", "restrictsPushes": false, "restrictsReviewDismissals": false, "requiredApprovingReviewCount": nil, "requiredDeploymentEnvironments": nil, "requiredStatusCheckContexts": []string{"CHECK_NAME_CANARY"}, "requiredStatusChecks": []any{map[string]any{"context": "CHECK_NAME_CANARY", "app": map[string]any{"id": "APP_SOURCE_CANARY"}}}}
			page := func(nodes []any) map[string]any {
				return map[string]any{"totalCount": len(nodes), "nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}}
			}
			clone := func(m map[string]any) map[string]any {
				v := map[string]any{}
				for k, x := range m {
					v[k] = x
				}
				return v
			}
			dir, err := os.MkdirTemp("/tmp", "ghs-status-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			writes, apps := 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Host != "api.github.com" || r.Method != "POST" || r.URL.Path != "/graphql" || r.Header.Get("Authorization") != "token status-fixture-token" {
					t.Error("credential/target boundary")
					w.WriteHeader(400)
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
					if request.Variables["id"] != "APP_SOURCE_CANARY" {
						t.Error("wrong source identity")
					}
					body["node"] = map[string]any{"__typename": "App", "id": "APP_SOURCE_CANARY"}
					if mode == "post-App-failure" && writes > 0 {
						json.NewEncoder(w).Encode(map[string]any{"data": body, "errors": []any{map[string]any{"message": "PRIVATE_RESOLVER_CANARY"}}})
						return
					}
				case strings.Contains(request.Query, "ClassicProtectionInventory"):
					v := clone(target)
					v["branchProtectionRules"] = page([]any{rule})
					body["repository"] = v
				case strings.Contains(request.Query, "ClassicProtectionActors"):
					group := ""
					for _, name := range []string{"pushAllowances", "reviewDismissalAllowances", "bypassForcePushAllowances", "bypassPullRequestAllowances"} {
						if strings.Contains(request.Query, name+"(") {
							group = name
						}
					}
					body["node"] = map[string]any{"__typename": "BranchProtectionRule", "id": "B_7", "databaseId": 7, "repository": target, group: page([]any{})}
				case strings.Contains(request.Query, "ClassicProtectionRule"):
					v := clone(rule)
					v["__typename"] = "BranchProtectionRule"
					body["node"] = v
				case strings.Contains(request.Query, "mutation ClassicProtectionWrite"):
					payload := request.Variables["input"].(map[string]any)
					if len(payload) != 2 || payload["branchProtectionRuleId"] != "B_7" || payload["requiresStrictStatusChecks"] != false {
						t.Error("not minimal strict update", payload)
					}
					writes++
					rule["requiresStrictStatusChecks"] = false
					ack := map[string]any{}
					for _, key := range []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced", "requiresStatusChecks", "requiresStrictStatusChecks", "requiredStatusCheckContexts", "requiredStatusChecks", "repository"} {
						ack[key] = rule[key]
					}
					body["updateBranchProtectionRule"] = map[string]any{"branchProtectionRule": ack}
				default:
					t.Error("unexpected query", request.Query)
					w.WriteHeader(400)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"data": body})
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			for _, privacy := range []string{"detailed", "status-only"} {
				result := f.run(t, "plan", "status-fixture-token", map[string]string{"github.action_ref": version, "inputs.plan-reporting": privacy})
				if result.status != 0 || len(result.remainingReports) != 0 {
					t.Fatal(result)
				}
				if privacy == "status-only" && (strings.Contains(result.log+result.summary, "CHECK_NAME_CANARY") || strings.Contains(result.log+result.summary, "APP_SOURCE_CANARY")) {
					t.Fatal("status-only leaked source", result)
				}
				if privacy == "detailed" && mode != "matching" && !strings.Contains(result.summary, "require_status_checks_to_pass_before_merging") {
					t.Fatal(result)
				}
			}
			extra := map[string]string{"github.action_ref": version}
			result := f.run(t, "apply", "status-fixture-token", extra)
			bad := mode == "post-App-failure"
			if (result.status != 0) != bad || bad && len(result.outputs) != 0 || !bad && len(result.outputs) != 4 || strings.Contains(result.log, "PRIVATE_RESOLVER_CANARY") {
				t.Fatal(result)
			}
			data, _ := os.ReadFile(path)
			if string(data) != input {
				t.Fatal("Action changed YAML")
			}
			if bad {
				if !strings.Contains(result.log, "unconfirmed effects may remain") {
					t.Fatal(result)
				}
				return
			}
			repeat := f.run(t, "apply", "status-fixture-token", extra)
			if repeat.status != 0 {
				t.Fatal(repeat)
			}
			mu.Lock()
			defer mu.Unlock()
			wantWrites, wantApps := 1, 2
			if mode == "matching" {
				wantWrites, wantApps = 0, 0
			}
			if writes != wantWrites || apps != wantApps {
				t.Fatal("noop performed work", writes, apps)
			}
		})
	}
}
