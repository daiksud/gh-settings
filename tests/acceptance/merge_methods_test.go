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
		{name: "merge-only plan", command: "plan", input: "allow_rebase_merge: false\nallow_squash_merge: true\nallow_merge_commit: false\n", body: observedMergeMethods, want: "settings.repository.allow_merge_commit: true → false\nsettings.repository.allow_squash_merge: false → true\nsettings.repository.allow_rebase_merge: true → false\n", requests: []string{"POST /graphql"}},
		{name: "false stays false", command: "plan", input: "allow_squash_merge: false\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "mixed read order", command: "plan", input: "has_issues: false\nallow_merge_commit: false\n", body: observedMergeMethods, want: "settings.repository.has_issues: true → false\nsettings.repository.allow_merge_commit: true → false\n", requests: []string{"GET /repos/owner/current", "POST /graphql"}},
		{name: "missing not false", command: "plan", input: "allow_merge_commit: false\n", body: `{"data":{"repository":{"squashMergeAllowed":false,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"autoMergeAllowed":false,"allowUpdateBranch":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}}}`, errorText: "missing mergeCommitAllowed", requests: []string{"POST /graphql"}},
		{name: "partial errors discard earlier REST comparison", command: "plan", input: "has_issues: false\nallow_merge_commit: false\n", body: `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":true,"rebaseMergeAllowed":true,"deleteBranchOnMerge":false,"autoMergeAllowed":false,"allowUpdateBranch":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}},"errors":[{"message":"fixture permission denied"}]}`, errorText: "fixture permission denied", requests: []string{"GET /repos/owner/current", "POST /graphql"}},
		{name: "permission failure", command: "plan", input: "allow_merge_commit: false\n", body: `{"data":{"repository":null},"errors":[{"message":"Resource not accessible by integration"}]}`, errorText: "Resource not accessible", requests: []string{"POST /graphql"}},
		{name: "direct false PATCH", command: "apply", input: "allow_merge_commit: false\nallow_squash_merge: true\nallow_rebase_merge: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_merge_commit": false, "allow_squash_merge": true, "allow_rebase_merge": false}},
		{name: "partial PATCH", command: "apply", input: "allow_rebase_merge: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_rebase_merge": false}},
		{name: "deletion-only plan", command: "plan", input: "delete_branch_on_merge: false\n", body: observedMergeMethods, want: "settings.repository.delete_branch_on_merge: true → false\n", requests: []string{"POST /graphql"}},
		{name: "deletion true equality", command: "plan", input: "delete_branch_on_merge: true\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "deletion false equality", command: "plan", input: "delete_branch_on_merge: false\n", body: strings.Replace(observedMergeMethods, `"deleteBranchOnMerge":true`, `"deleteBranchOnMerge":false`, 1), want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "deletion direct false", command: "apply", input: "delete_branch_on_merge: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"delete_branch_on_merge": false}},
		{name: "deletion direct true", command: "apply", input: "delete_branch_on_merge: true\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"delete_branch_on_merge": true}},
		{name: "squash title-only plan", command: "plan", input: "squash_merge_commit_title: PR_TITLE\n", body: observedMergeMethods, want: "settings.repository.squash_merge_commit_title: \"COMMIT_OR_PR_TITLE\" → \"PR_TITLE\"\n", requests: []string{"POST /graphql"}},
		{name: "squash pair equality", command: "plan", input: "squash_merge_commit_title: COMMIT_OR_PR_TITLE\nsquash_merge_commit_message: COMMIT_MESSAGES\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "squash title direct", command: "apply", input: "squash_merge_commit_title: PR_TITLE\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"squash_merge_commit_title": "PR_TITLE"}},
		{name: "squash pair direct", command: "apply", input: "squash_merge_commit_title: PR_TITLE\nsquash_merge_commit_message: BLANK\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"squash_merge_commit_title": "PR_TITLE", "squash_merge_commit_message": "BLANK"}},
		{name: "squash message-only rejected", command: "apply", input: "squash_merge_commit_message: BLANK\n", errorText: "requires squash_merge_commit_title"},
		{name: "squash unknown rejected", command: "plan", input: "squash_merge_commit_title: UNKNOWN\n", errorText: "must be PR_TITLE"},
		{name: "PR options plan", command: "plan", input: "allow_update_branch: true\nallow_auto_merge: true\n", body: observedMergeMethods, want: "settings.repository.allow_auto_merge: false → true\nsettings.repository.allow_update_branch: false → true\n", requests: []string{"POST /graphql"}},
		{name: "PR options false equality", command: "plan", input: "allow_auto_merge: false\nallow_update_branch: false\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "PR options true equality", command: "plan", input: "allow_auto_merge: true\nallow_update_branch: true\n", body: strings.ReplaceAll(strings.ReplaceAll(observedMergeMethods, `"autoMergeAllowed":false`, `"autoMergeAllowed":true`), `"allowUpdateBranch":false`, `"allowUpdateBranch":true`), want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "PR auto direct false", command: "apply", input: "allow_auto_merge: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_auto_merge": false}},
		{name: "PR update direct false", command: "apply", input: "allow_update_branch: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_update_branch": false}},
		{name: "PR options direct pair", command: "apply", input: "allow_auto_merge: true\nallow_update_branch: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"allow_auto_merge": true, "allow_update_branch": false}},
		{name: "PR update invalid local", command: "apply", input: "allow_update_branch: 'false'\n", errorText: "must be a boolean"},
		{name: "merge title plan", command: "plan", input: "merge_commit_title: PR_TITLE\n", body: observedMergeMethods, want: "settings.repository.merge_commit_title: \"MERGE_MESSAGE\" → \"PR_TITLE\"\n", requests: []string{"POST /graphql"}},
		{name: "merge pair equality", command: "plan", input: "merge_commit_title: MERGE_MESSAGE\nmerge_commit_message: PR_TITLE\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "merge title direct", command: "apply", input: "merge_commit_title: PR_TITLE\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"merge_commit_title": "PR_TITLE"}},
		{name: "merge pair direct", command: "apply", input: "merge_commit_title: PR_TITLE\nmerge_commit_message: BLANK\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}, patch: map[string]any{"merge_commit_title": "PR_TITLE", "merge_commit_message": "BLANK"}},
		{name: "invalid local", command: "plan", input: "allow_merge_commit: null\n", errorText: "must be a boolean"},
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
			input := repositoryInput(test.input)
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
			fields := document["settings"].(map[string]any)["repository"].(map[string]any)
			if fields["allow_merge_commit"] != true || fields["allow_squash_merge"] != false || fields["allow_rebase_merge"] != true || fields["delete_branch_on_merge"] != true || fields["squash_merge_commit_title"] != "COMMIT_OR_PR_TITLE" || fields["squash_merge_commit_message"] != "COMMIT_MESSAGES" || fields["allow_auto_merge"] != false || fields["allow_update_branch"] != false || fields["merge_commit_title"] != "MERGE_MESSAGE" || fields["merge_commit_message"] != "PR_TITLE" || fields["has_discussions"] != false || fields["web_commit_signoff_required"] != false || len(fields) != 17 {
				t.Fatalf("fields=%v", fields)
			}
			input := repositoryInput("merge_commit_message: BLANK\nmerge_commit_title: PR_TITLE\nsquash_merge_commit_message: BLANK\nsquash_merge_commit_title: PR_TITLE\nallow_update_branch: true\nallow_auto_merge: true\ndelete_branch_on_merge: false\nallow_rebase_merge: false\nallow_squash_merge: true\nallow_merge_commit: false\nhas_discussions: true\nhas_wiki: false\nhas_projects: false\nhas_issues: false\nhomepage: https://example.com/new\ndescription: ''\n")
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
					map[string]any{"field": "repository.allow_merge_commit", "before": true, "after": false},
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
				`settings.repository.description: "old" → ""`,
				`settings.repository.homepage: "old-url" → "https://example.com/new"`,
				"settings.repository.has_issues: true → false",
				"settings.repository.has_projects: true → false",
				"settings.repository.has_wiki: true → false",
				"settings.repository.has_discussions: false → true",
				"settings.repository.allow_merge_commit: true → false",
				"settings.repository.allow_squash_merge: false → true",
				"settings.repository.allow_rebase_merge: true → false",
				"settings.repository.delete_branch_on_merge: true → false",
				"settings.repository.allow_auto_merge: false → true",
				"settings.repository.allow_update_branch: false → true",
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
			if !reflect.DeepEqual(paths, []string{"GET /repos/owner/current", "POST /graphql", "GET /repos/owner/current/topics", "GET /repos/owner/current/labels", "GET /repos/owner/current", "POST /graphql"}) {
				t.Fatalf("paths=%v", paths)
			}
		})
	}
}
