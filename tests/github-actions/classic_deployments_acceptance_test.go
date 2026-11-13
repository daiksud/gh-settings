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

func TestBuiltCompositeClassicDeployments(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "enable", "disable", "empty", "replaced-reference", "reference-before-graphql", "reference-after-graphql", "reference-before-http", "reference-after-http", "reference-before-transport", "reference-after-transport", "reference-before-member", "reference-after-member"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			desired := scenario != "disable"
			group := "{enabled: false}"
			if desired {
				group = "{enabled: true, environments: [PRIVATE_ENV]}"
			}
			if scenario == "empty" {
				group = "{enabled: true, environments: []}"
			}
			input := fmt.Sprintf("version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: 'PRIVATE_DEPLOYMENTS/**', require_deployments_to_succeed_before_merging: %s}]}}\n", group)
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			directory, err := os.MkdirTemp("/tmp", "ghs-deployments-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(directory) })
			listener, err := net.Listen("unix", filepath.Join(directory, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			target := map[string]any{"id": "R_config", "nameWithOwner": "owner/config", "isArchived": false, "isPrivate": false, "owner": map[string]any{"__typename": "User"}, "squashMergeAllowed": true, "rebaseMergeAllowed": false}
			lock := !desired
			if scenario == "matching" {
				lock = desired
			}
			rule := map[string]any{
				"id": "B_7", "databaseId": 7, "pattern": "PRIVATE_DEPLOYMENTS/**", "repository": target,
				"requiresConversationResolution": false, "requiresCommitSignatures": false, "requiresLinearHistory": false, "allowsDeletions": false, "lockBranch": false,
				"isAdminEnforced": true, "allowsForcePushes": false, "blocksCreations": false, "dismissesStaleReviews": false, "lockAllowsFetchAndMerge": true, "requireLastPushApproval": false, "requiresApprovingReviews": false, "requiresCodeOwnerReviews": false, "requiresDeployments": lock, "requiresStatusChecks": false, "requiresStrictStatusChecks": false, "restrictsPushes": false, "restrictsReviewDismissals": false,
				"requiredApprovingReviewCount": nil, "requiredDeploymentEnvironments": nil, "requiredStatusCheckContexts": nil, "requiredStatusChecks": nil,
			}
			rule["requiredDeploymentEnvironments"] = []any{"PRIVATE_OLD"}
			if scenario == "matching" || scenario == "disable" {
				rule["requiredDeploymentEnvironments"] = []any{"PRIVATE_ENV"}
			}
			wanted := map[string]any{"branchProtectionRuleId": "B_7", "requiresDeployments": desired}
			if desired {
				wanted["requiredDeploymentEnvironments"] = []any{"PRIVATE_ENV"}
			}
			if scenario == "empty" {
				wanted["requiredDeploymentEnvironments"] = []any{}
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
			writes, referenceReads := 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Method != "POST" || r.URL.Path != "/graphql" || r.Header.Get("Authorization") != "token deployments-fixture-token" {
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
				case strings.Contains(request.Query, "ClassicDeploymentEnvironments"):
					referenceReads++
					if scenario == "matching" || scenario == "disable" || scenario == "empty" {
						t.Error("unnecessary Environment acquisition")
						w.WriteHeader(403)
						return
					}
					diagnosticFailure := strings.HasPrefix(scenario, "reference-") && ((strings.Contains(scenario, "-after-") && writes > 0) || (strings.Contains(scenario, "-before-") && writes == 0))
					const upstreamCanary = "A_PRIVATE_DEPLOYMENT_UPSTREAM_FAILURE"
					if diagnosticFailure {
						switch {
						case strings.HasSuffix(scenario, "-graphql"):
							json.NewEncoder(w).Encode(map[string]any{"errors": []any{map[string]any{"message": upstreamCanary}}})
							return
						case strings.HasSuffix(scenario, "-http"):
							w.WriteHeader(403)
							json.NewEncoder(w).Encode(map[string]any{"message": upstreamCanary})
							return
						case strings.HasSuffix(scenario, "-transport"):
							conn, buffer, err := w.(http.Hijacker).Hijack()
							if err != nil {
								t.Error(err)
								return
							}
							fmt.Fprint(buffer, upstreamCanary+"\r\n\r\n")
							buffer.Flush()
							conn.Close()
							return
						}
					}
					environmentID := "E_1"
					if scenario == "replaced-reference" && writes > 0 {
						environmentID = "E_replaced"
					}
					current := clone(target)
					current["environments"] = page([]any{map[string]any{"id": environmentID, "databaseId": nil, "name": "PRIVATE_ENV"}})
					if diagnosticFailure && strings.HasSuffix(scenario, "-member") {
						current["environments"].(map[string]any)["nodes"].([]any)[0].(map[string]any)[upstreamCanary] = true
					}
					body["repository"] = current
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
					for k, v := range wanted {
						if k != "branchProtectionRuleId" {
							rule[k] = v
						}
					}
					ack := map[string]any{}
					for _, k := range []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced", "requiresApprovingReviews", "requiredApprovingReviewCount", "dismissesStaleReviews", "requiresCodeOwnerReviews", "requireLastPushApproval", "requiresDeployments", "requiredDeploymentEnvironments", "requiresStatusChecks", "requiresStrictStatusChecks", "requiredStatusCheckContexts", "requiredStatusChecks", "allowsForcePushes", "repository"} {
						ack[k] = rule[k]
					}
					body["updateBranchProtectionRule"] = map[string]any{"branchProtectionRule": ack}
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
			for _, mode := range []string{"detailed", "status-only"} {
				result := f.run(t, "plan", "deployments-fixture-token", map[string]string{"github.action_ref": version, "inputs.plan-reporting": mode, "GH_DEBUG": "api"})
				if result.status != 0 || len(result.remainingReports) != 0 {
					t.Fatal(result)
				}
				if mode == "status-only" && (strings.Contains(result.log+result.summary, "PRIVATE_DEPLOYMENTS") || strings.Contains(result.log+result.summary, "require_deployments_to_succeed_before_merging") || strings.Contains(result.log+result.summary, "PRIVATE_ENV")) {
					t.Fatal("private policy leaked", result)
				}
				if strings.Contains(result.log+result.summary, "PRIVATE_OLD") {
					t.Fatal("inactive raw names leaked", result)
				}
				if mode == "detailed" && scenario != "matching" && !strings.Contains(result.summary, "require_deployments_to_succeed_before_merging") {
					t.Fatal(result.summary)
				}
			}
			mu.Lock()
			plannedWrites := writes
			mu.Unlock()
			if plannedWrites != 0 {
				t.Fatal("plan mutated", plannedWrites)
			}
			result := f.run(t, "apply", "deployments-fixture-token", map[string]string{"github.action_ref": version, "GH_DEBUG": "api"})
			success := scenario != "replaced-reference" && !strings.HasPrefix(scenario, "reference-")
			beforeFailure := strings.Contains(scenario, "-before-")
			if (result.status == 0) != success || success && len(result.outputs) != 4 || !success && len(result.outputs) != 0 {
				t.Fatal(result)
			}
			if strings.Contains(result.log+result.summary+fmt.Sprint(result.outputs), "A_PRIVATE_DEPLOYMENT_UPSTREAM_FAILURE") {
				t.Fatal("upstream response leaked", result)
			}
			if !success && strings.Contains(result.log, "unconfirmed effects may remain") == beforeFailure {
				t.Fatal("uncertain progress lost", result)
			}
			if success {
				if again := f.run(t, "apply", "deployments-fixture-token", map[string]string{"github.action_ref": version}); again.status != 0 {
					t.Fatal(again)
				}
			}
			mu.Lock()
			gotWrites, gotReferences := writes, referenceReads
			mu.Unlock()
			wantWrites := 1
			if scenario == "matching" || beforeFailure {
				wantWrites = 0
			}
			wantReferences := 0
			if scenario == "enable" || scenario == "replaced-reference" || strings.HasPrefix(scenario, "reference-") {
				wantReferences = 2
			}
			if beforeFailure {
				wantReferences = 1
			}
			if gotReferences != wantReferences {
				t.Fatal("reference reads", gotReferences, wantReferences)
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
