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

func TestBuiltCompositeClassicForcePush(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "select", "everyone", "disable", "actor-drift", "matching-actors-identity-failure", "matching-actors-graphql-error", "matching-actors-http-error", "matching-actors-malformed-response"} {
		t.Run(scenario, func(t *testing.T) {
			identityFailure := strings.HasPrefix(scenario, "matching-actors-")
			f := newCompositeFixture(t)
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			group := "{enabled: true, who_can_force_push: specified_actors, actors: [{type: app, id: A_PRIVATE}]}"
			if scenario == "everyone" {
				group = "{enabled: true, who_can_force_push: everyone}"
			}
			if scenario == "disable" {
				group = "{enabled: false}"
			}
			extra := ""
			if identityFailure {
				extra = ", require_signed_commits: true"
			}
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{id: 7, branch_name_pattern: 'PRIVATE_FORCE/**', allow_force_pushes: " + group + extra + "}]}}\n"
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			directory, err := os.MkdirTemp("/tmp", "ghs-force-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(directory) })
			listener, err := net.Listen("unix", filepath.Join(directory, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			target := map[string]any{"id": "R_config", "nameWithOwner": "owner/config", "isArchived": false, "isPrivate": false, "isFork": false, "owner": map[string]any{"__typename": "User"}, "squashMergeAllowed": true, "rebaseMergeAllowed": false}
			rule := map[string]any{"id": "B_7", "databaseId": 7, "pattern": "PRIVATE_FORCE/**", "repository": target, "requiresConversationResolution": false, "requiresCommitSignatures": false, "requiresLinearHistory": false, "allowsDeletions": false, "lockBranch": false, "isAdminEnforced": true, "allowsForcePushes": false, "blocksCreations": false, "dismissesStaleReviews": false, "lockAllowsFetchAndMerge": true, "requireLastPushApproval": false, "requiresApprovingReviews": false, "requiresCodeOwnerReviews": false, "requiresDeployments": false, "requiresStatusChecks": false, "requiresStrictStatusChecks": false, "restrictsPushes": false, "restrictsReviewDismissals": false, "requiredApprovingReviewCount": nil, "requiredDeploymentEnvironments": nil, "requiredStatusCheckContexts": nil, "requiredStatusChecks": nil}
			actors := []any{}
			if scenario == "matching" || identityFailure {
				actors = []any{map[string]any{"actor": map[string]any{"__typename": "App", "id": "A_PRIVATE"}}}
			}
			if scenario == "everyone" || scenario == "disable" {
				actors = []any{map[string]any{"actor": map[string]any{"__typename": "Team", "id": "T_PRIVATE"}}}
			}
			wanted := map[string]any{"branchProtectionRuleId": "B_7", "allowsForcePushes": false, "bypassForcePushActorIds": []any{"A_PRIVATE"}}
			if scenario == "everyone" {
				wanted = map[string]any{"branchProtectionRuleId": "B_7", "allowsForcePushes": true}
			}
			if scenario == "disable" {
				wanted["bypassForcePushActorIds"] = []any{}
			}
			page := func(nodes []any) map[string]any {
				return map[string]any{"totalCount": len(nodes), "nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}}
			}
			clone := func(m map[string]any) map[string]any {
				n := map[string]any{}
				for k, v := range m {
					n[k] = v
				}
				return n
			}
			var mu sync.Mutex
			writes, identityReads := 0, 0
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Method != "POST" || r.URL.Path != "/graphql" || r.Header.Get("Authorization") != "token force-fixture-token" {
					t.Error("escaped credential/target")
					w.WriteHeader(400)
					return
				}
				var req struct {
					Query     string
					Variables map[string]any
				}
				if json.NewDecoder(r.Body).Decode(&req) != nil {
					t.Error("invalid request")
					w.WriteHeader(400)
					return
				}
				body := map[string]any{}
				switch {
				case strings.Contains(req.Query, "ClassicForcePushIdentities"):
					identityReads++
					w.Header().Set("Content-Type", "application/json")
					if scenario == "matching-actors-graphql-error" {
						_ = json.NewEncoder(w).Encode(map[string]any{"errors": []any{map[string]any{"message": "A_PRIVATE_UPSTREAM_IDENTITY_FAILURE"}}})
						return
					}
					if scenario == "matching-actors-http-error" {
						w.WriteHeader(http.StatusForbidden)
						_ = json.NewEncoder(w).Encode(map[string]any{"message": "A_PRIVATE_UPSTREAM_IDENTITY_FAILURE"})
						return
					}
					if scenario == "matching-actors-malformed-response" {
						body["A_PRIVATE_UPSTREAM_IDENTITY_FAILURE"] = true
					}
					if scenario == "matching-actors-identity-failure" {
						body["nodes"] = []any{nil}
					} else {
						body["nodes"] = []any{map[string]any{"id": "A_PRIVATE", "__typename": "App"}}
					}
				case strings.Contains(req.Query, "ClassicProtectionInventory"):
					node := clone(target)
					node["branchProtectionRules"] = page([]any{rule})
					body["repository"] = node
				case strings.Contains(req.Query, "ClassicProtectionActors"):
					name := ""
					for _, k := range []string{"pushAllowances", "reviewDismissalAllowances", "bypassForcePushAllowances", "bypassPullRequestAllowances"} {
						if strings.Contains(req.Query, k+"(") {
							name = k
						}
					}
					if name == "" || req.Variables["id"] != "B_7" {
						t.Error("invalid actor read")
						w.WriteHeader(400)
						return
					}
					nodes := []any{}
					if name == "bypassForcePushAllowances" {
						nodes = actors
					}
					body["node"] = map[string]any{"__typename": "BranchProtectionRule", "id": "B_7", "databaseId": 7, "repository": target, name: page(nodes)}
				case strings.Contains(req.Query, "ClassicProtectionRule"):
					node := clone(rule)
					node["__typename"] = "BranchProtectionRule"
					body["node"] = node
				case strings.Contains(req.Query, "mutation ClassicProtectionWrite"):
					if !reflect.DeepEqual(req.Variables["input"], wanted) {
						t.Error("unexpected mutation", req.Variables)
						w.WriteHeader(400)
						return
					}
					writes++
					for k, v := range wanted {
						if k == "bypassForcePushActorIds" {
							actors = []any{}
							for _, id := range v.([]any) {
								actors = append(actors, map[string]any{"actor": map[string]any{"__typename": "App", "id": id}})
							}
						} else if k != "branchProtectionRuleId" {
							rule[k] = v
						}
					}
					ack := map[string]any{}
					for _, k := range []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced", "requiresApprovingReviews", "requiredApprovingReviewCount", "dismissesStaleReviews", "requiresCodeOwnerReviews", "requireLastPushApproval", "requiresDeployments", "requiredDeploymentEnvironments", "lockAllowsFetchAndMerge", "requiresStatusChecks", "requiresStrictStatusChecks", "requiredStatusCheckContexts", "requiredStatusChecks", "allowsForcePushes", "repository"} {
						ack[k] = rule[k]
					}
					body["updateBranchProtectionRule"] = map[string]any{"branchProtectionRule": ack}
					if scenario == "actor-drift" {
						actors = []any{}
					}
				default:
					t.Error("unexpected query", req.Query)
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
			for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[k] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			for _, mode := range []string{"detailed", "status-only"} {
				result := f.run(t, "plan", "force-fixture-token", map[string]string{"github.action_ref": version, "inputs.plan-reporting": mode, "GH_DEBUG": "api"})
				if result.status != 0 || len(result.remainingReports) != 0 {
					t.Fatal(result)
				}
				if mode == "status-only" && (strings.Contains(result.log+result.summary, "PRIVATE") || strings.Contains(result.log+result.summary, "allow_force_pushes")) {
					t.Fatal("identity leak", result)
				}
				if strings.Contains(result.log, "T_PRIVATE") || strings.Contains(result.log, "A_PRIVATE") {
					t.Fatal("debug actor leak", result)
				}
			}
			mu.Lock()
			plannedWrites := writes
			mu.Unlock()
			if plannedWrites != 0 {
				t.Fatal("plan wrote")
			}
			result := f.run(t, "apply", "force-fixture-token", map[string]string{"github.action_ref": version, "GH_DEBUG": "api"})
			if strings.Contains(result.log, "T_PRIVATE") || strings.Contains(result.log, "A_PRIVATE") {
				t.Fatal("apply debug actor leak", result)
			}
			success := scenario != "actor-drift" && !identityFailure
			if (result.status == 0) != success || success && len(result.outputs) != 4 || !success && len(result.outputs) != 0 {
				t.Fatal(result)
			}
			if scenario == "actor-drift" && !strings.Contains(result.log, "unconfirmed effects may remain") {
				t.Fatal(result)
			}
			if identityFailure && strings.Contains(result.log, "unconfirmed effects may remain") {
				t.Fatal("prewrite lookup claimed uncertain mutation", result)
			}
			if success {
				again := f.run(t, "apply", "force-fixture-token", map[string]string{"github.action_ref": version})
				if again.status != 0 {
					t.Fatal(again)
				}
			}
			mu.Lock()
			gotWrites, gotReads := writes, identityReads
			mu.Unlock()
			wantWrites, wantReads := 1, 0
			if scenario == "matching" {
				wantWrites = 0
			}
			if scenario == "select" || scenario == "actor-drift" || identityFailure {
				wantReads = 1
			}
			if identityFailure {
				wantWrites = 0
			}
			if gotWrites != wantWrites || gotReads != wantReads {
				t.Fatal(gotWrites, gotReads, wantWrites, wantReads)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("Action changed source", err)
			}
		})
	}
}
