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

func TestBuiltCompositeClassicPrune(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	f := newCompositeFixture(t)
	f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
	f.write(f.base["MOCK_ASSET"], string(product))
	f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
	f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
	const input = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: []}}\n"
	f.write(filepath.Join(f.caller, ".github/settings.yml"), input)
	state, err := os.MkdirTemp("/tmp", "ghs-classic-prune-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(state) })
	listener, err := net.Listen("unix", filepath.Join(state, "api.sock"))
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	deleted := false
	writes := 0
	target := map[string]any{"id": "R_config", "nameWithOwner": "owner/config", "isPrivate": false, "isArchived": false, "owner": map[string]any{"__typename": "User"}, "squashMergeAllowed": true, "rebaseMergeAllowed": false}
	rule := map[string]any{"id": "B_7", "databaseId": 7, "pattern": "PRIVATE_PATTERN_CANARY", "repository": target,
		"requiresConversationResolution": false, "requiresCommitSignatures": false, "requiresLinearHistory": false, "allowsDeletions": false,
		"isAdminEnforced": true, "allowsForcePushes": false, "blocksCreations": false, "dismissesStaleReviews": true, "lockBranch": false, "lockAllowsFetchAndMerge": false, "requireLastPushApproval": true, "requiresApprovingReviews": true, "requiresCodeOwnerReviews": true, "requiresDeployments": false, "requiresStatusChecks": true, "requiresStrictStatusChecks": true, "restrictsPushes": false, "restrictsReviewDismissals": false,
		"requiredApprovingReviewCount": 2, "requiredDeploymentEnvironments": nil, "requiredStatusCheckContexts": []string{"CHECK_CONTEXT_CANARY"}, "requiredStatusChecks": []any{map[string]any{"context": "CHECK_CONTEXT_CANARY", "app": map[string]any{"id": "STATUS_APP_SOURCE"}}}}
	page := func(nodes []any) map[string]any {
		return map[string]any{"totalCount": len(nodes), "nodes": nodes, "pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}}
	}
	clone := func(value map[string]any) map[string]any {
		data, _ := json.Marshal(value)
		var result map[string]any
		_ = json.Unmarshal(data, &result)
		return result
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token prune-explicit-token" || r.Method != "POST" || r.URL.Path != "/graphql" {
			t.Error("escaped token/host/endpoint boundary")
			w.WriteHeader(400)
			return
		}
		var request struct {
			Query     string
			Variables map[string]any
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("malformed query")
			w.WriteHeader(400)
			return
		}
		body := map[string]any{}
		switch {
		case strings.Contains(request.Query, "ClassicProtectionInventory"):
			if request.Variables["owner"] != "owner" || request.Variables["name"] != "config" {
				t.Error("wrong repository", request.Variables)
			}
			repo := clone(target)
			nodes := []any{}
			if !deleted {
				nodes = append(nodes, clone(rule))
			}
			repo["branchProtectionRules"] = page(nodes)
			body["repository"] = repo
		case strings.Contains(request.Query, "ClassicProtectionActors"):
			if request.Variables["id"] != "B_7" || deleted {
				t.Error("wrong allowance parent", request.Variables)
			}
			group := ""
			for _, key := range []string{"pushAllowances", "reviewDismissalAllowances", "bypassForcePushAllowances", "bypassPullRequestAllowances"} {
				if strings.Contains(request.Query, key+"(") {
					group = key
				}
			}
			body["node"] = map[string]any{"__typename": "BranchProtectionRule", "id": "B_7", "databaseId": 7, "repository": clone(target), group: page([]any{})}
		case strings.Contains(request.Query, "ClassicProtectionRule"):
			if request.Variables["id"] != "B_7" || deleted {
				t.Error("wrong selected node", request.Variables)
			}
			node := clone(rule)
			node["__typename"] = "BranchProtectionRule"
			body["node"] = node
		case strings.Contains(request.Query, "mutation ClassicProtectionDelete"):
			input, ok := request.Variables["input"].(map[string]any)
			if !ok || len(input) != 1 || input["branchProtectionRuleId"] != "B_7" || deleted {
				t.Error("wrong deletion payload", request.Variables)
			}
			writes++
			deleted = true
			body["deleteBranchProtectionRule"] = map[string]any{"__typename": "DeleteBranchProtectionRulePayload"}
		default:
			t.Error("unexpected GraphQL request", request.Query)
			w.WriteHeader(400)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": body})
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
		extra := map[string]string{"github.action_ref": version, "inputs.prune": "branch-protection-rules", "inputs.plan-reporting": mode, "GH_DEBUG": "api"}
		result := f.run(t, "plan", "prune-explicit-token", extra)
		if result.status != 0 || (mode == "status-only" && result.outputs["plan-outcome"] != "changed") || len(result.remainingReports) != 0 {
			t.Fatal(result)
		}
		combined := result.log + result.summary
		if mode == "status-only" && (strings.Contains(combined, "STATUS_APP_SOURCE") || strings.Contains(combined, "CHECK_CONTEXT_CANARY")) {
			t.Fatal("snapshot evidence leaked")
		}
		if mode == "status-only" && strings.Contains(combined, "PRIVATE_PATTERN_CANARY") {
			t.Fatal("private Classic pattern leaked")
		}
		if mode == "detailed" && !strings.Contains(result.summary, "settings.branch_protection_rules: delete") {
			t.Fatal(result.summary)
		}
		mu.Lock()
		if writes != 0 {
			t.Error("plan deleted", writes)
		}
		mu.Unlock()
	}
	extra := map[string]string{"github.action_ref": version, "inputs.prune": "all"}
	result := f.run(t, "apply", "prune-explicit-token", extra)
	if result.status != 0 || result.summary != "" {
		t.Fatal(result)
	}
	again := f.run(t, "apply", "prune-explicit-token", extra)
	if again.status != 0 {
		t.Fatal(again)
	}
	mu.Lock()
	if writes != 1 {
		t.Error("repeat deletion", writes)
	}
	mu.Unlock()
	data, _ := os.ReadFile(filepath.Join(f.caller, ".github/settings.yml"))
	if string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
		t.Fatal("Action changed YAML/commit")
	}
}
