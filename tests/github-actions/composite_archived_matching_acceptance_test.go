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

func TestBuiltCompositeArchivedMatching(t *testing.T) {
	product, version := suppliedCompositeProduct(t)
	for _, scenario := range []string{"matching", "description differs", "topics differ", "topics missing", "topics denied", "identity changed", "label differs", "labels denied", "mixed matching", "mixed unverified default", "mixed security differs", "graphql matching", "graphql merge differs", "graphql merge unknown", "graphql pr differs", "graphql pr unknown", "graphql issue differs", "graphql issue unknown"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCompositeFixture(t)
			f.base["GITHUB_ACTIONS"] = "true"
			f.digest = fmt.Sprintf("%x", sha256.Sum256(product))
			f.write(f.base["MOCK_ASSET"], string(product))
			f.write(f.base["MOCK_CHECKSUMS"], f.digest+"  gh-settings-linux-amd64\n")
			f.write(filepath.Join(f.packagePath, "action/cli-version"), version+"\n")
			input := "version: daiksud/gh-settings@v0\nrepository: {about: {description: same, topics: [alpha, beta]}, labels: [{id: 7, name: bug, color: '#112233'}], settings: {general: {danger_zone: {archive_this_repository: true}}}}\n"
			if strings.HasPrefix(scenario, "mixed ") {
				effort := "balanced"
				if scenario == "mixed unverified default" {
					effort = "default"
				}
				input = `version: daiksud/gh-settings@v0
repository:
  about: {description: same, topics: [alpha, beta]}
  labels: [{id: 7, name: bug, color: '#112233'}]
  settings:
    general: {features: {issues: true}, danger_zone: {archive_this_repository: true}}
    interaction_limits:
      pull_request_limits: {limit_open_pull_requests_from_users_without_write_access: true, exemption_list: []}
    copilot: {code_review: {review_effort_level: ` + effort + `}}
    actions:
      general:
        check_workflow_run_status_artifact_and_log_retention: 30
        approval_for_running_fork_pull_request_workflows_from_contributors: require_approval_for_first_time_contributors
    pages: {build_and_deployment: {source: deploy_from_a_branch, branch: none}}
    advanced_security: {private_vulnerability_reporting: false}
`
			}
			if strings.HasPrefix(scenario, "graphql ") {
				input = `version: daiksud/gh-settings@v0
repository:
  about: {description: same, topics: [alpha, beta]}
  labels: [{id: 7, name: bug, color: '#112233'}]
  settings:
    general:
      features:
        issues: true
        pull_requests: true
        pull_request_permissions: {creation_allowed_by: all_users}
        issue_permissions: {creation_allowed_by: all_users}
      pull_requests:
        allow_merge_commits: true
        merge_commit: {default_commit_message: pull_request_title_and_description}
      danger_zone: {archive_this_repository: true}
`
			}
			path := filepath.Join(f.caller, ".github/settings.yml")
			f.write(path, input)
			dir, err := os.MkdirTemp("/tmp", "ghs-archive-about-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			listener, err := net.Listen("unix", filepath.Join(dir, "api.sock"))
			if err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			reads, writes := 0, 0
			var graphReads []string
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if r.Host != "api.github.com" || r.Header.Get("Authorization") != "token archive-fixture" {
					t.Error("unexpected target or credential")
					w.WriteHeader(400)
					return
				}
				if r.Method == "POST" && r.URL.Path == "/graphql" {
					var request struct {
						Query     string            `json:"query"`
						Variables map[string]string `json:"variables"`
					}
					if json.NewDecoder(r.Body).Decode(&request) != nil || len(request.Variables) != 2 || request.Variables["owner"] != "owner" || request.Variables["name"] != "config" {
						t.Error("unexpected GraphQL request or target")
						writes++
						w.WriteHeader(400)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					switch strings.Join(strings.Fields(request.Query), " ") {
					case `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name) { mergeCommitAllowed squashMergeAllowed rebaseMergeAllowed deleteBranchOnMerge autoMergeAllowed allowUpdateBranch squashMergeCommitTitle squashMergeCommitMessage mergeCommitTitle mergeCommitMessage } }`:
						graphReads = append(graphReads, "merge")
						title, message := "PR_TITLE", "PR_BODY"
						if scenario == "graphql merge differs" {
							message = "BLANK"
						}
						if scenario == "graphql merge unknown" {
							title = "FUTURE"
						}
						fmt.Fprintf(w, `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":false,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"autoMergeAllowed":false,"allowUpdateBranch":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES","mergeCommitTitle":%q,"mergeCommitMessage":%q}}}`, title, message)
					case `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name, followRenames: false) { hasPullRequestsEnabled pullRequestCreationPolicy } }`:
						graphReads = append(graphReads, "pr")
						policy := "ALL"
						if scenario == "graphql pr differs" {
							policy = "COLLABORATORS_ONLY"
						}
						if scenario == "graphql pr unknown" {
							policy = "FUTURE"
						}
						fmt.Fprintf(w, `{"data":{"repository":{"hasPullRequestsEnabled":true,"pullRequestCreationPolicy":%q}}}`, policy)
					case `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name, followRenames: false) { issueCreationPolicy } }`:
						graphReads = append(graphReads, "issue")
						policy := "ALL"
						if scenario == "graphql issue differs" {
							policy = "COLLABORATORS_ONLY"
						}
						if scenario == "graphql issue unknown" {
							policy = "FUTURE"
						}
						fmt.Fprintf(w, `{"data":{"repository":{"issueCreationPolicy":%q}}}`, policy)
					default:
						t.Error("unexpected GraphQL query or mutation")
						writes++
						w.WriteHeader(400)
					}
					return
				}
				if r.Method != "GET" {
					writes++
					w.WriteHeader(400)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/repos/owner/config":
					reads++
					id := 1
					if scenario == "identity changed" && reads == 3 {
						id = 2
					}
					description := "same"
					if scenario == "description differs" {
						description = "different"
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "full_name": "owner/config", "archived": true, "visibility": "public", "description": description, "homepage": "", "default_branch": "main", "has_issues": true, "has_projects": false, "has_wiki": false, "has_discussions": false, "web_commit_signoff_required": false, "is_template": false})
				case "/repos/owner/config/interaction-limits/pulls/creation-cap":
					fmt.Fprint(w, `{"enabled":true,"max_open_pull_requests":5,"include_drafts":false}`)
				case "/repos/owner/config/interaction-limits/pulls/bypass-list":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/config/copilot/code-review/settings":
					configured := `"balanced"`
					if scenario == "mixed unverified default" {
						configured = `null`
					}
					fmt.Fprintf(w, `{"review_effort":{"configured":%s,"effective":"balanced"},"available_review_efforts":["lite","balanced"],"severity_filter":"no_filter","custom_instructions_enabled":false,"mcp_enabled":false}`, configured)
				case "/repos/owner/config/actions/permissions/artifact-and-log-retention":
					fmt.Fprint(w, `{"days":30,"maximum_allowed_days":90}`)
				case "/repos/owner/config/actions/permissions/fork-pr-contributor-approval":
					fmt.Fprint(w, `{"approval_policy":"first_time_contributors"}`)
				case "/repos/owner/config/private-vulnerability-reporting":
					fmt.Fprintf(w, `{"enabled":%t}`, scenario == "mixed security differs")
				case "/repos/owner/config/pages":
					w.WriteHeader(404)
					fmt.Fprint(w, `{"message":"Not Found"}`)
				case "/repos/owner/config/labels":
					if scenario == "labels denied" {
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"denied"}`)
						return
					}
					description := ""
					if scenario == "label differs" {
						description = "owned value"
					}
					_ = json.NewEncoder(w).Encode([]any{map[string]any{"id": 7, "name": "bug", "color": "112233", "description": description}})
				case "/repos/owner/config/topics":
					switch scenario {
					case "topics denied":
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"denied"}`)
					case "topics missing":
						fmt.Fprint(w, `{}`)
					case "topics differ":
						fmt.Fprint(w, `{"names":["other"]}`)
					default:
						fmt.Fprint(w, `{"names":["beta","alpha"]}`)
					}
				default:
					t.Error("unexpected route", r.URL.Path)
					w.WriteHeader(400)
				}
			})}
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { _ = server.Close() })
			f.write(filepath.Join(dir, "config.yml"), "http_unix_socket: "+filepath.Join(dir, "api.sock")+"\n")
			f.base["GH_CONFIG_DIR"], f.base["GH_PATH_DISABLED"] = dir, "1"
			for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
				f.base[key] = "http://127.0.0.1:1"
			}
			f.base["NO_PROXY"], f.base["no_proxy"] = "", ""
			result := f.run(t, "apply", "archive-fixture", map[string]string{"github.action_ref": version})
			if (result.status == 0) != (scenario == "matching" || scenario == "mixed matching" || scenario == "graphql matching") || (result.status != 0 && len(result.outputs) != 0) {
				t.Fatalf("unexpected result: %+v", result)
			}

			mu.Lock()
			defer mu.Unlock()
			if strings.HasPrefix(scenario, "graphql ") && strings.Join(graphReads, ",") != "merge,pr,issue" {
				t.Fatalf("expected all three read-only GraphQL queries, got %v; action result=%+v", graphReads, result)
			}
			if writes != 0 {
				t.Fatalf("unexpected remote writes: %d", writes)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input || f.git("rev-parse", "HEAD") != f.sha {
				t.Fatal("configuration or commit changed", err)
			}
		})
	}
}
