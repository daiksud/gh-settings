//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
)

const observedMergeMethods = `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":false,"rebaseMergeAllowed":true,"deleteBranchOnMerge":true,"autoMergeAllowed":false,"allowUpdateBranch":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}}}`

func TestMergeMethodsPlanAndDirectApplyAcceptance(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, command, input, body, want, errorText string
		requests                                    []string
		patch                                       map[string]any // nil means no PATCH; otherwise the complete expected object.
	}{
		{name: "merge-only plan", command: "plan", input: "pull_requests: {allow_rebase_merging: false, allow_squash_merging: true, allow_merge_commits: false}\n", body: observedMergeMethods, want: "repository.settings.general.pull_requests.allow_merge_commits: true → false\nrepository.settings.general.pull_requests.allow_squash_merging: false → true\nrepository.settings.general.pull_requests.allow_rebase_merging: true → false\n", requests: []string{"POST /graphql"}},
		{name: "false stays false", command: "plan", input: "pull_requests: {allow_squash_merging: false}\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "mixed read order", command: "plan", input: "features: {issues: false}\npull_requests: {allow_merge_commits: false}\n", body: observedMergeMethods, want: "repository.settings.general.features.issues: true → false\nrepository.settings.general.pull_requests.allow_merge_commits: true → false\n", requests: []string{"GET /repos/owner/current", "POST /graphql"}},
		{name: "missing not false", command: "plan", input: "pull_requests: {allow_merge_commits: false}\n", body: `{"data":{"repository":{"squashMergeAllowed":false,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"autoMergeAllowed":false,"allowUpdateBranch":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}}}`, errorText: "missing mergeCommitAllowed", requests: []string{"POST /graphql"}},
		{name: "partial errors discard earlier REST comparison", command: "plan", input: "features: {issues: false}\npull_requests: {allow_merge_commits: false}\n", body: `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":true,"rebaseMergeAllowed":true,"deleteBranchOnMerge":false,"autoMergeAllowed":false,"allowUpdateBranch":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}},"errors":[{"message":"fixture permission denied"}]}`, errorText: "fixture permission denied", requests: []string{"GET /repos/owner/current", "POST /graphql"}},
		{name: "permission failure", command: "plan", input: "pull_requests: {allow_merge_commits: false}\n", body: `{"data":{"repository":null},"errors":[{"message":"Resource not accessible by integration"}]}`, errorText: "Resource not accessible", requests: []string{"POST /graphql"}},
		{name: "direct false PATCH", command: "apply", input: "pull_requests: {allow_merge_commits: false, allow_squash_merging: true, allow_rebase_merging: false}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_merge_commit": false, "allow_squash_merge": true, "allow_rebase_merge": false}},
		{name: "partial PATCH", command: "apply", input: "pull_requests: {allow_rebase_merging: false}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_rebase_merge": false}},
		{name: "deletion-only plan", command: "plan", input: "pull_requests: {automatically_delete_head_branches: false}\n", body: observedMergeMethods, want: "repository.settings.general.pull_requests.automatically_delete_head_branches: true → false\n", requests: []string{"POST /graphql"}},
		{name: "deletion true equality", command: "plan", input: "pull_requests: {automatically_delete_head_branches: true}\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "deletion false equality", command: "plan", input: "pull_requests: {automatically_delete_head_branches: false}\n", body: strings.Replace(observedMergeMethods, `"deleteBranchOnMerge":true`, `"deleteBranchOnMerge":false`, 1), want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "deletion direct false", command: "apply", input: "pull_requests: {automatically_delete_head_branches: false}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"delete_branch_on_merge": false}},
		{name: "deletion direct true", command: "apply", input: "pull_requests: {automatically_delete_head_branches: true}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"delete_branch_on_merge": true}},
		{name: "squash commit details plan", command: "plan", input: "pull_requests: {squash_merge: {default_commit_message: pull_request_title_and_commit_details}}\n", body: observedMergeMethods, want: "settings.repository.squash_merge_commit_title: \"COMMIT_OR_PR_TITLE\" → \"PR_TITLE\"\n", requests: []string{"POST /graphql"}},
		{name: "squash pair equality", command: "plan", input: "pull_requests: {squash_merge: {default_commit_message: default_message}}\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "squash commit details direct", command: "apply", input: "pull_requests: {squash_merge: {default_commit_message: pull_request_title_and_commit_details}}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"squash_merge_commit_title": "PR_TITLE", "squash_merge_commit_message": "COMMIT_MESSAGES"}},
		{name: "squash pair direct", command: "apply", input: "pull_requests: {squash_merge: {default_commit_message: pull_request_title}}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"squash_merge_commit_title": "PR_TITLE", "squash_merge_commit_message": "BLANK"}},
		{name: "obsolete API message key rejected", command: "apply", input: "squash_merge_commit_message: BLANK\n", errorText: `unknown repository.settings.general key "squash_merge_commit_message"`},
		{name: "squash unknown rejected", command: "plan", input: "pull_requests: {squash_merge: {default_commit_message: UNKNOWN}}\n", errorText: `unsupported general.pull_requests.squash_merge.default_commit_message "UNKNOWN"`},
		{name: "PR options plan", command: "plan", input: "pull_requests: {always_suggest_updating_pull_request_branches: true, allow_auto_merge: true}\n", body: observedMergeMethods, want: "repository.settings.general.pull_requests.allow_auto_merge: false → true\nrepository.settings.general.pull_requests.always_suggest_updating_pull_request_branches: false → true\n", requests: []string{"POST /graphql"}},
		{name: "PR options false equality", command: "plan", input: "pull_requests: {allow_auto_merge: false, always_suggest_updating_pull_request_branches: false}\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "PR options true equality", command: "plan", input: "pull_requests: {allow_auto_merge: true, always_suggest_updating_pull_request_branches: true}\n", body: strings.ReplaceAll(strings.ReplaceAll(observedMergeMethods, `"autoMergeAllowed":false`, `"autoMergeAllowed":true`), `"allowUpdateBranch":false`, `"allowUpdateBranch":true`), want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "PR auto direct false", command: "apply", input: "pull_requests: {allow_auto_merge: false}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_auto_merge": false}},
		{name: "PR update direct false", command: "apply", input: "pull_requests: {always_suggest_updating_pull_request_branches: false}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_update_branch": false}},
		{name: "PR options direct pair", command: "apply", input: "pull_requests: {allow_auto_merge: true, always_suggest_updating_pull_request_branches: false}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_auto_merge": true, "allow_update_branch": false}},
		{name: "PR update invalid local", command: "apply", input: "pull_requests: {always_suggest_updating_pull_request_branches: 'false'}\n", errorText: "must be a boolean"},
		{name: "merge menu plan", command: "plan", input: "pull_requests: {merge_commit: {default_commit_message: pull_request_title}}\n", body: observedMergeMethods, want: "settings.repository.merge_commit_title: \"MERGE_MESSAGE\" → \"PR_TITLE\"\nsettings.repository.merge_commit_message: \"PR_TITLE\" → \"BLANK\"\n", requests: []string{"POST /graphql"}},
		{name: "disabled merge title rejected by validate", command: "validate", input: "pull_requests: {allow_merge_commits: false, merge_commit: {default_commit_message: pull_request_title}}\n", errorText: "general.pull_requests.merge_commit.default_commit_message requires the merge method to be enabled"},
		{name: "disabled merge title rejected by plan", command: "plan", input: "pull_requests: {allow_merge_commits: false, merge_commit: {default_commit_message: pull_request_title}}\n", errorText: "general.pull_requests.merge_commit.default_commit_message requires the merge method to be enabled"},
		{name: "observed disabled merge title rejected by plan", command: "plan", input: "pull_requests: {merge_commit: {default_commit_message: pull_request_title}}\n", body: strings.Replace(observedMergeMethods, `"mergeCommitAllowed":true`, `"mergeCommitAllowed":false`, 1), errorText: "require allow_merge_commit: true", requests: []string{"POST /graphql"}},
		{name: "explicit merge enable overrides observed disabled", command: "plan", input: "pull_requests: {allow_merge_commits: true, merge_commit: {default_commit_message: pull_request_title}}\n", body: strings.Replace(observedMergeMethods, `"mergeCommitAllowed":true`, `"mergeCommitAllowed":false`, 1), want: "repository.settings.general.pull_requests.allow_merge_commits: false → true\nsettings.repository.merge_commit_title: \"MERGE_MESSAGE\" → \"PR_TITLE\"\nsettings.repository.merge_commit_message: \"PR_TITLE\" → \"BLANK\"\n", requests: []string{"POST /graphql"}},
		{name: "merge pair equality", command: "plan", input: "pull_requests: {merge_commit: {default_commit_message: default_message}}\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "merge description menu direct", command: "apply", input: "pull_requests: {merge_commit: {default_commit_message: pull_request_title_and_description}}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"merge_commit_title": "PR_TITLE", "merge_commit_message": "PR_BODY"}},
		{name: "merge pair direct", command: "apply", input: "pull_requests: {merge_commit: {default_commit_message: pull_request_title}}\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"merge_commit_title": "PR_TITLE", "merge_commit_message": "BLANK"}},
		{name: "invalid local", command: "plan", input: "pull_requests: {allow_merge_commits: null}\n", errorText: "repository.settings.general.pull_requests.allow_merge_commits must not be null"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch {
				case servePRAccessFixtureRead(w, r):
				case serveIssueCreationPolicyFixtureRead(w, r):
				case serveSponsorshipsFixtureRead(w, r):
				case isMergeMethodsFixtureRead(r):
					fmt.Fprint(w, test.body)
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case r.Method == "PATCH" && r.URL.Path == "/repos/owner/current":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if test.patch == nil || !reflect.DeepEqual(payload, test.patch) {
						t.Errorf("payload=%v want=%v", payload, test.patch)
					}
					fmt.Fprint(w, `{}`)
				default:
					t.Errorf("unexpected request=%s %s", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, test.command != "apply")
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := generalInput(test.input)
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{test.command}, false)
			if err != nil {
				t.Fatal(err)
			}
			if test.errorText != "" {
				if result.runErr == nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), test.errorText) {
					t.Fatalf("result=%+v", result)
				}
			} else if result.runErr != nil || result.stderr.Len() != 0 || result.stdout.String() != test.want {
				t.Fatalf("result=%+v want=%q", result, test.want)
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(requests, test.requests) {
				t.Fatalf("requests=%v want=%v", requests, test.requests)
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMergeMethodsExportAndActionPresentation(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			var mu sync.Mutex
			var paths []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				paths = append(paths, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch {
				case servePRAccessFixtureRead(w, r):
				case serveIssueCreationPolicyFixtureRead(w, r):
				case serveSponsorshipsFixtureRead(w, r):
				case isMergeMethodsFixtureRead(r):
					if fail {
						fmt.Fprint(w, `{"data":{"repository":null},"errors":[{"message":"fixture read denied"}]}`)
					} else {
						fmt.Fprint(w, observedMergeMethods)
					}
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, true)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			before := "previous configuration\n"
			if err := writeSettingsFile(fixture.work, before); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
			if err != nil {
				t.Fatal(err)
			}
			if fail {
				if result.runErr == nil || !strings.Contains(result.stderr.String(), "fixture read denied") || result.stdout.Len() != 0 {
					t.Fatalf("result=%+v", result)
				}
				if err := checkSettingsUnchanged(fixture.work, before); err != nil {
					t.Fatal(err)
				}
				mu.Lock()
				defer mu.Unlock()
				if !reflect.DeepEqual(paths, []string{"GET /repos/owner/current", "POST /graphql"}) {
					t.Fatalf("paths=%v", paths)
				}
				return
			}
			if result.runErr != nil || result.stdout.Len() != 0 || result.stderr.Len() != 0 {
				t.Fatalf("result=%+v", result)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := yaml.Unmarshal(data, &document); err != nil {
				t.Fatal(err)
			}
			wantExport := map[string]any{
				"version": "daiksud/gh-settings@v0",
				"repository": map[string]any{
					"about":  map[string]any{"description": "old", "website": "old-url", "topics": []any{}},
					"labels": []any{},
					"settings": map[string]any{"general": map[string]any{
						"template_repository": false, "default_branch": "main",
						"features":      map[string]any{"wikis": true, "issues": true, "sponsorships": false, "discussions": false, "projects": true, "pull_requests": false, "issue_permissions": map[string]any{"creation_allowed_by": "all_users"}},
						"pull_requests": map[string]any{"allow_merge_commits": true, "allow_squash_merging": false, "allow_rebase_merging": true, "automatically_delete_head_branches": true, "allow_auto_merge": false, "always_suggest_updating_pull_request_branches": false, "merge_commit": map[string]any{"default_commit_message": "default_message"}},
						"commits":       map[string]any{"require_contributors_to_sign_off_on_web_based_commits": false},
					}},
				},
			}
			if !reflect.DeepEqual(document, wantExport) {
				t.Fatalf("export=%v want=%v", document, wantExport)
			}
			input := repositoryInput(`about: {description: '', website: https://example.com/new}
settings:
  general:
    features: {discussions: true, wikis: false, projects: false, issues: false}
    pull_requests:
      merge_commit: {default_commit_message: pull_request_title}
      squash_merge: {default_commit_message: pull_request_title}
      always_suggest_updating_pull_request_branches: true
      allow_auto_merge: true
      automatically_delete_head_branches: false
      allow_rebase_merging: false
      allow_squash_merging: true
      allow_merge_commits: true
`)

			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			report := filepath.Join(fixture.root, "plan.json")
			result, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"action", "plan", report}, false)
			if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
				t.Fatalf("plan=%+v err=%v", result, err)
			}
			data, err = os.ReadFile(report)
			if err != nil {
				t.Fatal(err)
			}
			var typed map[string]any
			if err := json.Unmarshal(data, &typed); err != nil {
				t.Fatal(err)
			}
			wantReport := map[string]any{
				"outcome": "changed", "target": "owner/current", "managed": float64(16),
				"changes": []any{
					map[string]any{"field": "repository.description", "before": "old", "after": ""},
					map[string]any{"field": "repository.homepage", "before": "old-url", "after": "https://example.com/new"},
					map[string]any{"field": "repository.has_issues", "before": true, "after": false},
					map[string]any{"field": "repository.has_projects", "before": true, "after": false},
					map[string]any{"field": "repository.has_wiki", "before": true, "after": false},
					map[string]any{"field": "repository.has_discussions", "before": false, "after": true},
					map[string]any{"field": "repository.allow_squash_merge", "before": false, "after": true},
					map[string]any{"field": "repository.allow_rebase_merge", "before": true, "after": false},
					map[string]any{"field": "repository.delete_branch_on_merge", "before": true, "after": false},
					map[string]any{"field": "repository.allow_auto_merge", "before": false, "after": true},
					map[string]any{"field": "repository.allow_update_branch", "before": false, "after": true},
					map[string]any{"field": "repository.squash_merge_commit_title", "before": "COMMIT_OR_PR_TITLE", "after": "PR_TITLE"},
					map[string]any{"field": "repository.squash_merge_commit_message", "before": "COMMIT_MESSAGES", "after": "BLANK"},
					map[string]any{"field": "repository.merge_commit_title", "before": "MERGE_MESSAGE", "after": "PR_TITLE"},
					map[string]any{"field": "repository.merge_commit_message", "before": "PR_TITLE", "after": "BLANK"},
				},
			}
			if !reflect.DeepEqual(typed, wantReport) {
				t.Fatalf("report=%s want=%v", data, wantReport)
			}
			summary := filepath.Join(fixture.root, "summary.md")
			summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=pull_request", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
			result, err = executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
			if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
				t.Fatalf("summary=%+v err=%v", result, err)
			}
			data, err = os.ReadFile(summary)
			if err != nil {
				t.Fatal(err)
			}
			visible := html.UnescapeString(string(data))
			for _, line := range []string{
				`repository.about.description: "old" → ""`,
				`repository.about.website: "old-url" → "https://example.com/new"`,
				"repository.settings.general.features.issues: true → false",
				"repository.settings.general.features.projects: true → false",
				"repository.settings.general.features.wikis: true → false",
				"repository.settings.general.features.discussions: false → true",
				"repository.settings.general.pull_requests.allow_squash_merging: false → true",
				"repository.settings.general.pull_requests.allow_rebase_merging: true → false",
				"repository.settings.general.pull_requests.automatically_delete_head_branches: true → false",
				"repository.settings.general.pull_requests.allow_auto_merge: false → true",
				"repository.settings.general.pull_requests.always_suggest_updating_pull_request_branches: false → true",
				`settings.repository.squash_merge_commit_title: "COMMIT_OR_PR_TITLE" → "PR_TITLE"`,
				`settings.repository.squash_merge_commit_message: "COMMIT_MESSAGES" → "BLANK"`,
				`settings.repository.merge_commit_title: "MERGE_MESSAGE" → "PR_TITLE"`,
				`settings.repository.merge_commit_message: "PR_TITLE" → "BLANK"`,
			} {
				position := strings.Index(visible, line)
				if position < 0 {
					t.Fatalf("Summary missing ordered line %q: %s", line, data)
				}
				visible = visible[position+len(line):]
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(paths, []string{"GET /repos/owner/current", "POST /graphql", "POST /graphql", "POST /graphql", "POST /graphql", "GET /repos/owner/current/topics", "GET /repos/owner/current/labels", "GET /repos/owner/current", "POST /graphql"}) {
				t.Fatalf("paths=%v", paths)
			}
		})
	}
}
