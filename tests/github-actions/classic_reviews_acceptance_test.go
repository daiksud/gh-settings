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
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestBuiltCompositeClassicReviews(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "enable", "zero", "disable", "guarded", "readback-drift"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			desired := scenario != "disable" && scenario != "guarded"
			group := "{enabled: false}"
			if desired {
				group = "{enabled: true, require_approvals: false, dismiss_stale_pull_request_approvals_when_new_commits_are_pushed: false, require_review_from_code_owners: false, require_approval_of_the_most_recent_reviewable_push: false}"
			}
			const declarationKey = "require_a_pull_request_before_merging"
			input := fmt.Sprintf("version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: 'PRIVATE_REVIEW/**', %s: %s}]}}\n", declarationKey, group)
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			directory, err := os.MkdirTemp("/tmp", "ghs-reviews-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(directory) })
			listener, err := net.Listen("unix", filepath.Join(directory, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			target := map[string]any{"id": "R_config", "nameWithOwner": "owner/config", "isArchived": false, "isPrivate": false, "owner": map[string]any{"__typename": "User"}, "squashMergeAllowed": true, "rebaseMergeAllowed": false}

			rule := map[string]any{
				"id": "B_7", "databaseId": 7, "pattern": "PRIVATE_REVIEW/**", "repository": target,
				"requiresConversationResolution": false, "requiresCommitSignatures": false, "requiresLinearHistory": false, "allowsDeletions": false, "lockBranch": false,
				"isAdminEnforced": true, "allowsForcePushes": false, "blocksCreations": false, "dismissesStaleReviews": false, "lockAllowsFetchAndMerge": true, "requireLastPushApproval": false, "requiresApprovingReviews": false, "requiresCodeOwnerReviews": false, "requiresDeployments": false, "requiresStatusChecks": false, "requiresStrictStatusChecks": false, "restrictsPushes": false, "restrictsReviewDismissals": false,
				"requiredApprovingReviewCount": nil, "requiredDeploymentEnvironments": nil, "requiredStatusCheckContexts": nil, "requiredStatusChecks": nil,
			}
			currentActive := scenario == "matching" || scenario == "zero" || scenario == "disable" || scenario == "guarded"
			rule["requiresApprovingReviews"] = currentActive
			if currentActive {
				rule["requiredApprovingReviewCount"] = 2
			}
			if scenario == "matching" {
				rule["requiredApprovingReviewCount"] = 0
			}
			if scenario == "guarded" {
				rule["restrictsReviewDismissals"] = true
			}
			wanted := map[string]any{"branchProtectionRuleId": "B_7"}
			if !desired {
				wanted["requiresApprovingReviews"] = false
			} else if currentActive {
				wanted["requiredApprovingReviewCount"] = float64(0)
			} else {
				wanted["requiresApprovingReviews"] = true
				wanted["requiredApprovingReviewCount"] = float64(0)
				wanted["dismissesStaleReviews"] = false
				wanted["requiresCodeOwnerReviews"] = false
				wanted["requireLastPushApproval"] = false
			}
			page := func(nodes []any) map[string]any {
				return map[string]any{"totalCount": len(nodes), "nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}}
			}
			clone := func(m map[string]any) map[string]any {
				out := map[string]any{}
				for k, v := range m {
					out[k] = v
				}
				return out
			}
			var mu sync.Mutex
			writes := 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Method != "POST" || r.URL.Path != "/graphql" || r.Header.Get("Authorization") != "token reviews-fixture-token" {
					t.Error("escaped credential/target", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				var request struct {
					Query     string
					Variables map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				body := map[string]any{}
				switch {
				case strings.Contains(request.Query, "ClassicProtectionInventory"):
					node := clone(target)
					node["branchProtectionRules"] = page([]any{rule})
					body["repository"] = node
				case strings.Contains(request.Query, "ClassicProtectionActors"):
					group := ""
					for _, k := range []string{"pushAllowances", "reviewDismissalAllowances", "bypassForcePushAllowances", "bypassPullRequestAllowances"} {
						if strings.Contains(request.Query, k+"(") {
							group = k
						}
					}
					if group == "" || request.Variables["id"] != "B_7" {
						t.Error("invalid actor read")
						w.WriteHeader(400)
						return
					}
					body["node"] = map[string]any{"__typename": "BranchProtectionRule", "id": "B_7", "databaseId": 7, "repository": target, group: page([]any{})}
				case strings.Contains(request.Query, "ClassicProtectionRule"):
					if request.Variables["id"] != "B_7" {
						t.Error("wrong node")
						w.WriteHeader(400)
						return
					}
					node := clone(rule)
					node["__typename"] = "BranchProtectionRule"
					body["node"] = node
				case strings.Contains(request.Query, "mutation ClassicProtectionWrite"):
					if !reflect.DeepEqual(request.Variables["input"], wanted) {
						t.Error("unexpected mutation", request.Variables)
						w.WriteHeader(400)
						return
					}
					writes++
					for key, value := range wanted {
						if key != "branchProtectionRuleId" {
							rule[key] = value
						}
					}
					ack := map[string]any{}
					for _, k := range []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced", "requiresApprovingReviews", "requiredApprovingReviewCount", "dismissesStaleReviews", "requiresCodeOwnerReviews", "requireLastPushApproval", "requiresDeployments", "requiredDeploymentEnvironments", "requiresStatusChecks", "requiresStrictStatusChecks", "requiredStatusCheckContexts", "requiredStatusChecks", "repository"} {
						ack[k] = rule[k]
					}
					body["updateBranchProtectionRule"] = map[string]any{"branchProtectionRule": ack}
					if scenario == "readback-drift" {
						rule["requiredApprovingReviewCount"] = 1
					}
				default:
					t.Error("unexpected query", request.Query)
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"data": body})
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(directory, "config.yml"), "http_unix_socket: "+filepath.Join(directory, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = directory, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			if scenario == "guarded" {
				for _, operation := range []string{"plan", "apply"} {
					result := f.run(t, operation, "reviews-fixture-token", map[string]string{"github.action_ref": version})
					if result.status == 0 || len(result.outputs) != 0 || !strings.Contains(result.log, "unowned bypass or dismissal") {
						t.Fatal(result)
					}
				}
				mu.Lock()
				defer mu.Unlock()
				data, _ := os.ReadFile(path)
				if writes != 0 || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
					t.Fatal("guard changed source or remote", writes)
				}
				return
			}
			for _, mode := range []string{"detailed", "status-only"} {
				result := f.run(t, "plan", "reviews-fixture-token", map[string]string{"github.action_ref": version, "inputs.plan-reporting": mode, "GH_DEBUG": "api"})
				if result.status != 0 || len(result.remainingReports) != 0 {
					t.Fatal(result)
				}
				if mode == "status-only" && (strings.Contains(result.log+result.summary, "PRIVATE_REVIEW") || strings.Contains(result.log+result.summary, declarationKey)) {
					t.Fatal("private policy leaked", result)
				}
				if mode == "detailed" && scenario != "matching" && !strings.Contains(result.summary, declarationKey) {
					t.Fatal(result.summary)
				}
			}
			mu.Lock()
			plannedWrites := writes
			mu.Unlock()
			if plannedWrites != 0 {
				t.Fatal("plan mutated", plannedWrites)
			}
			result := f.run(t, "apply", "reviews-fixture-token", map[string]string{"github.action_ref": version})
			success := scenario != "readback-drift"
			if (result.status == 0) != success || success && len(result.outputs) != 4 || !success && len(result.outputs) != 0 {
				t.Fatal(result)
			}
			if !success && !strings.Contains(result.log, "unconfirmed effects may remain") {
				t.Fatal("uncertain progress lost", result)
			}
			if success {
				if again := f.run(t, "apply", "reviews-fixture-token", map[string]string{"github.action_ref": version}); again.status != 0 {
					t.Fatal(again)
				}
			}
			mu.Lock()
			gotWrites := writes
			mu.Unlock()
			wantWrites := 1
			if scenario == "matching" {
				wantWrites = 0
			}
			if gotWrites != wantWrites {
				t.Fatal("unexpected mutations", gotWrites)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("Action changed source", err)
			}
		})
	}
}
