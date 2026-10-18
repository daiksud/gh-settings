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

const observedMergeMethods = `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":false,"rebaseMergeAllowed":true,"deleteBranchOnMerge":true}}}`

func TestMergeMethodsPlanAndDirectApplyAcceptance(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, command, input, body, want, errorText string
		requests                                    []string
	}{
		{name: "merge-only plan", command: "plan", input: "allow_rebase_merge: false\nallow_squash_merge: true\nallow_merge_commit: false\n", body: observedMergeMethods, want: "settings.repository.allow_merge_commit: true → false\nsettings.repository.allow_squash_merge: false → true\nsettings.repository.allow_rebase_merge: true → false\n", requests: []string{"POST /graphql"}},
		{name: "false stays false", command: "plan", input: "allow_squash_merge: false\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "mixed read order", command: "plan", input: "has_issues: false\nallow_merge_commit: false\n", body: observedMergeMethods, want: "settings.repository.has_issues: true → false\nsettings.repository.allow_merge_commit: true → false\n", requests: []string{"GET /repos/owner/current", "POST /graphql"}},
		{name: "missing not false", command: "plan", input: "allow_merge_commit: false\n", body: `{"data":{"repository":{"squashMergeAllowed":false,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false}}}`, errorText: "missing mergeCommitAllowed", requests: []string{"POST /graphql"}},
		{name: "null not false", command: "plan", input: "allow_merge_commit: false\n", body: `{"data":{"repository":{"mergeCommitAllowed":null,"squashMergeAllowed":false,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false}}}`, errorText: "must be a boolean", requests: []string{"POST /graphql"}},
		{name: "partial errors discard earlier REST comparison", command: "plan", input: "has_issues: false\nallow_merge_commit: false\n", body: `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":true,"rebaseMergeAllowed":true,"deleteBranchOnMerge":false}},"errors":[{"message":"fixture permission denied"}]}`, errorText: "fixture permission denied", requests: []string{"GET /repos/owner/current", "POST /graphql"}},
		{name: "permission failure", command: "plan", input: "allow_merge_commit: false\n", body: `{"data":{"repository":null},"errors":[{"message":"Resource not accessible by integration"}]}`, errorText: "Resource not accessible", requests: []string{"POST /graphql"}},
		{name: "direct false PATCH", command: "apply", input: "allow_merge_commit: false\nallow_squash_merge: true\nallow_rebase_merge: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}},
		{name: "partial PATCH", command: "apply", input: "allow_rebase_merge: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}},
		{name: "deletion-only plan", command: "plan", input: "delete_branch_on_merge: false\n", body: observedMergeMethods, want: "settings.repository.delete_branch_on_merge: true → false\n", requests: []string{"POST /graphql"}},
		{name: "deletion true equality", command: "plan", input: "delete_branch_on_merge: true\n", body: observedMergeMethods, want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "deletion false equality", command: "plan", input: "delete_branch_on_merge: false\n", body: strings.Replace(observedMergeMethods, `"deleteBranchOnMerge":true`, `"deleteBranchOnMerge":false`, 1), want: "No differences for managed repository settings.\n", requests: []string{"POST /graphql"}},
		{name: "deletion missing fails", command: "plan", input: "delete_branch_on_merge: false\n", body: strings.Replace(observedMergeMethods, `,"deleteBranchOnMerge":true`, "", 1), errorText: "missing deleteBranchOnMerge", requests: []string{"POST /graphql"}},
		{name: "deletion null fails", command: "plan", input: "delete_branch_on_merge: false\n", body: strings.Replace(observedMergeMethods, `"deleteBranchOnMerge":true`, `"deleteBranchOnMerge":null`, 1), errorText: "must be a boolean", requests: []string{"POST /graphql"}},
		{name: "deletion partial errors", command: "plan", input: "has_issues: false\ndelete_branch_on_merge: false\n", body: strings.TrimSuffix(observedMergeMethods, "}") + `,"errors":[{"message":"deletion fixture denied"}]}`, errorText: "deletion fixture denied", requests: []string{"GET /repos/owner/current", "POST /graphql"}},
		{name: "deletion direct false", command: "apply", input: "delete_branch_on_merge: false\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}},
		{name: "deletion direct true", command: "apply", input: "delete_branch_on_merge: true\n", want: "Applied repository settings.\n", requests: []string{"PATCH /repos/owner/current"}},
		{name: "deletion invalid local", command: "apply", input: "delete_branch_on_merge: null\n", errorText: "must be a boolean"},
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
					want := map[string]any{"allow_rebase_merge": false}
					if test.name == "direct false PATCH" {
						want = map[string]any{"allow_merge_commit": false, "allow_squash_merge": true, "allow_rebase_merge": false}
					}
					if strings.HasPrefix(test.name, "deletion direct") {
						want = map[string]any{"delete_branch_on_merge": test.name == "deletion direct true"}
					}
					if !reflect.DeepEqual(payload, want) {
						t.Errorf("payload=%v want=%v", payload, want)
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
			if fields["allow_merge_commit"] != true || fields["allow_squash_merge"] != false || fields["allow_rebase_merge"] != true || fields["delete_branch_on_merge"] != true || len(fields) != 9 {
				t.Fatalf("fields=%v", fields)
			}
			input := repositoryInput("allow_merge_commit: false\nallow_squash_merge: true\nallow_rebase_merge: false\ndelete_branch_on_merge: false\n")
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
			changes := typed["changes"].([]any)
			if typed["managed"] != float64(4) || typed["outcome"] != "changed" || len(changes) != 4 {
				t.Fatalf("report=%s", data)
			}
			for _, change := range changes {
				c := change.(map[string]any)
				if _, ok := c["before"].(bool); !ok {
					t.Fatal("non-boolean before")
				}
				if _, ok := c["after"].(bool); !ok {
					t.Fatal("non-boolean after")
				}
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
			if !strings.Contains(html.UnescapeString(string(data)), "settings.repository.allow_merge_commit: true → false") || !strings.Contains(html.UnescapeString(string(data)), "settings.repository.delete_branch_on_merge: true → false") {
				t.Fatalf("summary=%s", data)
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(paths, []string{"GET /repos/owner/current", "POST /graphql", "GET /repos/owner/current/topics", "GET /repos/owner/current/labels", "POST /graphql"}) {
				t.Fatalf("paths=%v", paths)
			}
		})
	}
}
