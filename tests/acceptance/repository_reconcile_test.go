//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestRepositoryReconciliationAcceptance(t *testing.T) {
	for _, test := range []struct {
		name       string
		matching   bool
		preDenied  bool
		postDenied bool
		stale      bool
		drift      bool
	}{
		{name: "matching skips mutation", matching: true},
		{name: "changed then repeat skips mutation"},
		{name: "denied preflight stops before mutation", preDenied: true},
		{name: "denied readback stops later resources", postDenied: true},
		{name: "acknowledged but stale readback fails", stale: true},
		{name: "verify matching declared member omitted from patch", drift: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			remote := map[string]any{
				"default_branch": "main", "description": "old", "homepage": "https://same.example",
				"has_issues": true, "has_projects": true, "has_wiki": false, "has_discussions": false,
				"web_commit_signoff_required": false, "is_template": false,
			}
			if test.matching {
				remote["description"], remote["has_issues"] = "new", false
			}
			var requests []string
			var patches []map[string]any
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != "/repos/owner/current" {
					t.Errorf("unexpected later resource: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected resource", 400)
					return
				}
				switch r.Method {
				case http.MethodGet:
					if test.preDenied || (test.postDenied && len(patches) > 0) {
						http.Error(w, "denied fixture observation", 403)
						return
					}
				case http.MethodPatch:
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					patches = append(patches, payload)
					if !test.stale {
						for key, value := range payload {
							remote[key] = value
						}
					}
					if test.drift {
						remote["homepage"] = "https://changed.example"
					}
				default:
					t.Errorf("unexpected method: %s", r.Method)
				}
				if err := json.NewEncoder(w).Encode(remote); err != nil {
					t.Error(err)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			wantFailure := test.preDenied || test.postDenied || test.stale || test.drift
			input := "version: daiksud/gh-settings@v0\nrepository:\n  about:\n    description: new\n    website: https://same.example\n"
			if wantFailure {
				input += "    topics: [later]\n"
			}
			input += "  settings:\n    general:\n      features:\n        issues: false\n"
			if err := writeSettingsFile(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			binary := acceptanceBinary(t)
			attempts := 1
			if !wantFailure {
				attempts = 2
			}
			for attempt := 0; attempt < attempts; attempt++ {
				result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
				if err != nil {
					t.Fatal(err)
				}
				if wantFailure {
					if result.runErr == nil || result.stdout.Len() != 0 || strings.TrimSpace(result.stderr.String()) == "" {
						t.Fatalf("failure=%v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
					}
				} else if result.runErr != nil || result.stdout.String() != "Applied repository settings.\n" || hasNonInfoDiagnostic(result.stderr.String()) {
					t.Fatalf("attempt=%d error=%v stdout=%q stderr=%q", attempt, result.runErr, result.stdout.String(), result.stderr.String())
				}
			}
			mu.Lock()
			defer mu.Unlock()
			wantRequests := []string{"GET /repos/owner/current"}
			if !test.matching && !test.preDenied {
				wantRequests = append(wantRequests, "PATCH /repos/owner/current", "GET /repos/owner/current")
				wantPayload := map[string]any{"description": "new", "has_issues": false}
				if len(patches) != 1 || !reflect.DeepEqual(patches[0], wantPayload) {
					t.Fatalf("patches=%v want one %v", patches, wantPayload)
				}
			} else if len(patches) != 0 {
				t.Fatalf("unexpected mutations: %v", patches)
			}
			if attempts == 2 {
				wantRequests = append(wantRequests, "GET /repos/owner/current")
			}
			if !reflect.DeepEqual(requests, wantRequests) {
				t.Fatal(fmt.Sprintf("requests=%v want=%v", requests, wantRequests))
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRepositoryNineteenMembersConvergeAndRepeatWithoutMutation(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	base := map[string]any{
		"default_branch": "main", "description": "old", "homepage": "https://same.example",
		"has_issues": true, "has_projects": true, "has_wiki": false, "has_discussions": false,
		"web_commit_signoff_required": false, "is_template": false,
	}
	merge := map[string]any{
		"mergeCommitAllowed": true, "squashMergeAllowed": true, "rebaseMergeAllowed": true,
		"deleteBranchOnMerge": true, "autoMergeAllowed": false, "allowUpdateBranch": true,
		"squashMergeCommitTitle": "COMMIT_OR_PR_TITLE", "squashMergeCommitMessage": "COMMIT_MESSAGES",
		"mergeCommitTitle": "MERGE_MESSAGE", "mergeCommitMessage": "PR_TITLE",
	}
	var requests []string
	patches := 0
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case isFixedRepositoryFixtureRead(r, `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name, followRenames: false) { mergeCommitAllowed squashMergeAllowed rebaseMergeAllowed deleteBranchOnMerge autoMergeAllowed allowUpdateBranch squashMergeCommitTitle squashMergeCommitMessage mergeCommitTitle mergeCommitMessage } }`):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": merge}})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/current":
			_ = json.NewEncoder(w).Encode(base)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/current":
			patches++
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			want := map[string]any{"description": "new", "has_issues": false, "allow_rebase_merge": false, "merge_commit_title": "PR_TITLE", "merge_commit_message": "PR_BODY"}
			if !reflect.DeepEqual(payload, want) {
				t.Errorf("patch=%v want=%v", payload, want)
			}
			base["description"], base["has_issues"] = "new", false
			merge["rebaseMergeAllowed"] = false
			merge["mergeCommitTitle"], merge["mergeCommitMessage"] = "PR_TITLE", "PR_BODY"
			_ = json.NewEncoder(w).Encode(base)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", 400)
		}
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := `version: daiksud/gh-settings@v0
repository:
  about:
    description: new
    website: https://same.example
  settings:
    general:
      default_branch: main
      template_repository: false
      commits:
        require_contributors_to_sign_off_on_web_based_commits: false
      features:
        issues: false
        projects: true
        wikis: false
        discussions: false
      pull_requests:
        allow_merge_commits: true
        allow_squash_merging: true
        allow_rebase_merging: false
        automatically_delete_head_branches: true
        allow_auto_merge: false
        always_suggest_updating_pull_request_branches: true
        squash_merge:
          default_commit_message: default_message
        merge_commit:
          default_commit_message: pull_request_title_and_description
`
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
		if err != nil || result.runErr != nil || result.stdout.String() != "Applied repository settings.\n" || hasNonInfoDiagnostic(result.stderr.String()) {
			t.Fatalf("attempt=%d error=%v result=%+v", attempt, err, result)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{"GET /repos/owner/current", "POST /graphql", "PATCH /repos/owner/current", "GET /repos/owner/current", "POST /graphql", "GET /repos/owner/current", "POST /graphql"}
	if patches != 1 || !reflect.DeepEqual(requests, want) {
		t.Fatalf("patches=%d requests=%v want=%v", patches, requests, want)
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
}
