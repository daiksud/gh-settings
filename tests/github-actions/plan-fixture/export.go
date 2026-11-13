package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

// Export uses a separate read-only credential and a closed route inventory.
// Known unavailable families are deliberate; unexpected calls still fail assertions.
func serveExport(w http.ResponseWriter, r *http.Request, log io.Writer) bool {
	if r.Host != "api.github.com" {
		return false
	}
	if r.Method == http.MethodPost && r.URL.RequestURI() == "/graphql" {
		var request struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			return false
		}
		query := strings.Join(strings.Fields(request.Query), " ")
		classic := query == classicExportFixtureQuery && reflect.DeepEqual(request.Variables, map[string]any{"owner": "example-owner", "name": "settings-fixture", "cursor": nil})
		if !classic && !reflect.DeepEqual(request.Variables, map[string]any{"owner": "example-owner", "name": "settings-fixture"}) {
			return false
		}
		known := classic
		for _, fields := range []string{"hasPullRequestsEnabled pullRequestCreationPolicy", "issueCreationPolicy", "hasSponsorshipsEnabled"} {
			known = known || query == "query($owner: String!, $name: String!) { repository(owner: $owner, name: $name, followRenames: false) { "+fields+" } }"
		}
		known = known || query == "query($owner: String!, $name: String!) { repository(owner: $owner, name: $name) { mergeCommitAllowed squashMergeAllowed rebaseMergeAllowed deleteBranchOnMerge autoMergeAllowed allowUpdateBranch squashMergeCommitTitle squashMergeCommitMessage mergeCommitTitle mergeCommitMessage } }"
		if !known {
			return false
		}
		fmt.Fprintln(log, "POST /graphql query")
		http.Error(w, `{"message":"fixture observation unavailable"}`, http.StatusForbidden)
		return true
	}
	if r.Method != http.MethodGet {
		return false
	}
	const base = "/repos/example-owner/settings-fixture"
	var response string
	switch r.URL.RequestURI() {
	case base:
		response = `{"default_branch":"main","description":"Exported repository","homepage":"","has_issues":true,"has_projects":true,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`
	case base + "/topics":
		response = `{"names":["export-fixture"]}`
	case base + "/labels?per_page=100&page=1":
		response = `[]`
	case base + "/actions/variables?per_page=30&page=1":
		response = `{"total_count":1,"variables":[{"name":"EXPORT_CANARY","value":"CANARY-local-export-value"}]}`
	case base + "/agents/variables?per_page=30&page=1":
		response = `{"total_count":0,"variables":[]}`
	case base + "/environments?per_page=100&page=1":
		response = `{"total_count":0,"environments":[]}`
	case base + "/autolinks", base + "/actions/permissions", base + "/actions/permissions/workflow",
		base + "/actions/permissions/artifact-and-log-retention", base + "/actions/permissions/fork-pr-contributor-approval",
		base + "/actions/cache/retention-limit", base + "/actions/cache/storage-limit",
		base + "/actions/policies?has_parents=false&per_page=100&page=1", base + "/rulesets?includes_parents=false&per_page=100&page=1",
		base + "/copilot/code-review/settings", base + "/interaction-limits/pulls/bypass-list",
		base + "/interaction-limits/pulls/creation-cap", base + "/pages", base + "/code-scanning/default-setup",
		base + "/private-vulnerability-reporting", base + "/immutable-releases", base + "/vulnerability-alerts":
	default:
		return false
	}
	fmt.Fprintln(log, r.Method, r.URL.RequestURI())
	w.Header().Set("Content-Type", "application/json")
	if response == "" {
		w.WriteHeader(http.StatusForbidden)
		response = `{"message":"fixture observation unavailable"}`
	}
	fmt.Fprintln(w, response)
	return true
}

// Literal inspected query: an unavailable classic family is still a bounded read.
const classicExportFixtureQuery = `query ClassicProtectionInventory($owner: String!, $name: String!, $cursor: String) { repository(owner:$owner,name:$name,followRenames:false) { id nameWithOwner isArchived isPrivate owner { __typename } squashMergeAllowed rebaseMergeAllowed branchProtectionRules(first:100,after:$cursor) { totalCount pageInfo { hasNextPage endCursor } nodes { id databaseId pattern requiresConversationResolution requiresCommitSignatures requiresLinearHistory allowsDeletions lockBranch isAdminEnforced requiresDeployments requiredDeploymentEnvironments requiresStatusChecks requiresStrictStatusChecks requiredStatusCheckContexts requiredStatusChecks { context app { id } } allowsForcePushes blocksCreations dismissesStaleReviews lockAllowsFetchAndMerge requireLastPushApproval requiresApprovingReviews requiresCodeOwnerReviews restrictsPushes restrictsReviewDismissals requiredApprovingReviewCount repository { id nameWithOwner isArchived isPrivate owner { __typename } squashMergeAllowed rebaseMergeAllowed } } } } }`
