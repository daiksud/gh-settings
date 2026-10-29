//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestApplyAcceptance(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, test := range []struct {
		name, input, payload, wantError, response                    string
		status                                                       int
		noAPI, noOrigin, noAuth, nested, outputFailure, globalOrigin bool
		gitSetup                                                     func(string, []string) error
	}{
		{name: "default-branch-change", input: generalInput("default_branch: release/v2\n"), payload: `{"default_branch":"release/v2"}`, response: `{"default_branch":"release/v2"}`},
		{name: "default-branch-mixed", input: generalInput("default_branch: release/v2\nfeatures:\n  issues: false\n"), payload: `{"default_branch":"release/v2","has_issues":false}`, response: `{"default_branch":"release/v2"}`},
		{name: "default-branch-missing-response", input: generalInput("default_branch: release/v2\n"), payload: `{"default_branch":"release/v2"}`, wantError: "lacks valid default_branch"},
		{name: "default-branch-ignored-stops-topics", input: "version: daiksud/gh-settings@v0\nrepository:\n  about: {topics: []}\n  settings: {general: {default_branch: release/v2}}\n", payload: `{"default_branch":"release/v2"}`, response: `{"default_branch":"main"}`, wantError: "expected"},
		{name: "default-branch-not-found", input: generalInput("default_branch: missing-branch\n"), payload: `{"default_branch":"missing-branch"}`, status: 422, wantError: "422"},
		{name: "default-branch-forbidden", input: generalInput("default_branch: release/v2\n"), payload: `{"default_branch":"release/v2"}`, status: 403, wantError: "403"},
		{name: "default-branch-null-before-origin", input: generalInput("default_branch: null\n"), noAPI: true, noOrigin: true, noAuth: true, wantError: "must not be null"},
		{name: "template-disable", input: generalInput("template_repository: false\n"), payload: `{"is_template":false}`},
		{name: "web-signoff-disable", input: generalInput("commits:\n  require_contributors_to_sign_off_on_web_based_commits: false\n"), payload: `{"web_commit_signoff_required":false}`},
		{name: "template-enable", input: generalInput("template_repository: true\n"), payload: `{"is_template":true}`},
		{name: "web-signoff-enable", input: generalInput("commits:\n  require_contributors_to_sign_off_on_web_based_commits: true\n"), payload: `{"web_commit_signoff_required":true}`},
		{name: "template-rejected", input: generalInput("template_repository: true\n"), payload: `{"is_template":true}`, status: 422, wantError: "422"},
		{name: "template-forbidden", input: generalInput("template_repository: false\n"), payload: `{"is_template":false}`, status: 403, wantError: "403"},
		{name: "web-signoff-forbidden", input: generalInput("commits:\n  require_contributors_to_sign_off_on_web_based_commits: false\n"), payload: `{"web_commit_signoff_required":false}`, status: 403, wantError: "403"},
		{name: "template-invalid-before-origin", input: generalInput("template_repository: null\n"), noAPI: true, noOrigin: true, noAuth: true, wantError: "must not be null"},
		{name: "web-signoff-invalid-before-origin", input: generalInput("commits:\n  require_contributors_to_sign_off_on_web_based_commits: null\n"), noAPI: true, noOrigin: true, noAuth: true, wantError: "must not be null"},
		{name: "discussions-false", input: generalInput("features:\n  discussions: false\n"), payload: `{"has_discussions":false}`, response: `{"has_discussions":false}`},
		{name: "discussions-ignored", input: generalInput("features:\n  discussions: false\n"), payload: `{"has_discussions":false}`, response: `{"has_discussions":true}`, wantError: "expected false"},
		{name: "discussions-missing-response", input: generalInput("features:\n  discussions: false\n"), payload: `{"has_discussions":false}`, wantError: "lacks boolean has_discussions"},
		{name: "discussions-forbidden", input: generalInput("features:\n  discussions: false\n"), payload: `{"has_discussions":false}`, status: 403, wantError: "403"},
		{name: "discussions-invalid-before-origin", input: generalInput("features:\n  discussions: null\n"), noAPI: true, noOrigin: true, noAuth: true, wantError: "must not be null"},
		{name: "partial-false", input: generalInput("features:\n  issues: false\n"), payload: `{"has_issues":false}`, nested: true},
		{name: "all-zero-values", input: repositoryInput("about:\n  description: ''\n  website: ''\nsettings:\n  general:\n    features:\n      issues: false\n      projects: false\n      wikis: false\n"), payload: `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`},
		{name: "true-and-escaped-string", input: repositoryInput("about:\n  description: \"quote\\\"\\n\\\\\"\nsettings:\n  general:\n    features:\n      wikis: true\n"), payload: `{"description":"quote\"\n\\","has_wiki":true}`},
		{name: "rewritten-origin", input: generalInput("features:\n  projects: true\n"), payload: `{"has_projects":true}`, gitSetup: setupRewrittenOrigin},
		{name: "aliased-origin", input: aboutInput("website: ''\n"), noAPI: true, wantError: "origin", gitSetup: setupAliasedOrigin},
		{name: "worktree-origin", input: aboutInput("description: new\n"), payload: `{"description":"new"}`, gitSetup: setupWorktreeOrigin},
		{name: "global-origin-before-local", input: generalInput("features:\n  issues: false\n"), payload: `{"has_issues":false}`, globalOrigin: true},
		{name: "global-origin-before-alias", input: generalInput("features:\n  issues: false\n"), noAPI: true, wantError: "origin", globalOrigin: true, gitSetup: setupAliasedOrigin},
		{name: "global-origin-before-worktree", input: generalInput("features:\n  issues: false\n"), payload: `{"has_issues":false}`, globalOrigin: true, gitSetup: setupWorktreeOrigin},
		{name: "global-only-origin", input: generalInput("features:\n  issues: false\n"), wantError: "checkout-scoped", noAPI: true, noOrigin: true, globalOrigin: true},
		{name: "no-managed-fields", input: "version: daiksud/gh-settings@v0\n", noAPI: true, noAuth: true, noOrigin: true},
		{name: "empty-settings", input: "version: daiksud/gh-settings@v0\nrepository: {settings: {}}\n", noAPI: true, noAuth: true, noOrigin: true},
		{name: "empty-repository", input: "version: daiksud/gh-settings@v0\nrepository: {}\n", noAPI: true, noAuth: true, noOrigin: true},
		{name: "malformed-before-api", input: generalInput("features:\n  issues: [\n"), wantError: "decode YAML", noAPI: true},
		{name: "invalid-before-origin-auth", input: generalInput("features:\n  issues: 'false'\n"), wantError: "must be a boolean", noAPI: true, noOrigin: true, noAuth: true},
		{name: "unknown-key", input: generalInput("private: true\n"), wantError: "unknown repository.settings.general key", noAPI: true},
		{name: "duplicate-key", input: generalInput("features:\n  wikis: true\n  wikis: false\n"), wantError: "duplicate key", noAPI: true},
		{name: "explicit-null", input: aboutInput("description: null\n"), wantError: "must be a string", noAPI: true},
		{name: "missing-version", input: "repository: {}\n", wantError: "missing version", noAPI: true},
		{name: "unsupported-version", input: "version: daiksud/gh-settings@v1\n", wantError: "version must be the string", noAPI: true},
		{name: "multiple-documents", input: "version: daiksud/gh-settings@v0\n---\nversion: daiksud/gh-settings@v0\n", wantError: "one YAML document", noAPI: true},
		{name: "missing-file", wantError: "read .github/settings.yml", noAPI: true},
		{name: "missing-origin", input: generalInput("features:\n  wikis: false\n"), wantError: "a GitHub origin remote is required", noAPI: true, noOrigin: true},
		{name: "authentication", input: generalInput("features:\n  wikis: false\n"), wantError: "configure GitHub API client", noAPI: true, noAuth: true},
		{name: "unauthorized", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, status: 401, wantError: "401"},
		{name: "forbidden", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, status: 403, wantError: "403"},
		{name: "server-validation", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, status: 422, wantError: "422"},
		{name: "redirect-301", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, status: 301, wantError: "301"},
		{name: "redirect-302", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, status: 302, wantError: "302"},
		{name: "redirect-303", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, status: 303, wantError: "303"},
		{name: "redirect-307", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, status: 307, wantError: "307"},
		{name: "redirect-308", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, status: 308, wantError: "308"},
		{name: "output-after-patch", input: generalInput("features:\n  wikis: false\n"), payload: `{"has_wiki":false}`, outputFailure: true, wantError: "write output:"},
		{name: "no-op-output-failure", input: "version: daiksud/gh-settings@v0\n", noAPI: true, noAuth: true, noOrigin: true, outputFailure: true, wantError: "write output:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := make(chan string, 10)
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				requests <- fmt.Sprintf("%s %s %s", r.Method, r.URL.Path, body)
				w.Header().Set("Content-Type", "application/json")
				status := test.status
				if status == 0 {
					status = 200
				}
				if status >= 300 && status < 400 {
					if r.URL.Path == "/repos/owner/current" {
						w.Header().Set("Location", "/repos/owner/renamed")
					} else {
						status = 200
					}
				}
				w.WriteHeader(status)
				responseBody := test.response
				if responseBody == "" {
					responseBody = `{"message":"fixture response"}`
				}
				fmt.Fprint(w, responseBody)
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if test.noAuth {
				env = fixtureEnv(fixture.cfg)
			}
			setup := setupOrdinaryOrigin
			if test.noOrigin {
				setup = initGitCheckout
			} else if test.gitSetup != nil {
				setup = test.gitSetup
			}
			if err := setup(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			if test.globalOrigin {
				global := filepath.Join(fixture.root, "global.conf")
				if err := os.WriteFile(global, []byte("[remote \"origin\"]\n\turl = https://github.com/unrelated/wrong.git\n"), 0600); err != nil {
					t.Fatal(err)
				}
				var isolated []string
				for _, entry := range env {
					if !strings.HasPrefix(entry, "GIT_CONFIG_GLOBAL=") {
						isolated = append(isolated, entry)
					}
				}
				env = append(isolated, "GIT_CONFIG_GLOBAL="+global)
			}
			if err := writeSettingsFile(fixture.work, test.input); err != nil {
				t.Fatal(err)
			}
			work := fixture.work
			if test.nested {
				work = filepath.Join(work, "nested")
				if err := os.Mkdir(work, 0700); err != nil {
					t.Fatal(err)
				}
			}
			result, err := executeCLI(binary, work, fixture.root, env, []string{"apply"}, test.outputFailure)
			if err != nil {
				t.Fatal(err)
			}
			wantRequests := 1
			if test.noAPI {
				wantRequests = 0
			}
			if len(requests) != wantRequests {
				t.Fatalf("requests=%d want=%d", len(requests), wantRequests)
			}
			if wantRequests == 1 {
				if got := <-requests; got != "PATCH /repos/owner/current "+test.payload {
					t.Fatalf("request=%s want body=%s", got, test.payload)
				}
			}
			if test.wantError == "" {
				want := "Applied repository settings.\n"
				if test.noAPI {
					want = "No managed repository settings.\n"
				}
				if result.runErr != nil || result.stdout.String() != want || result.stderr.Len() != 0 {
					t.Fatalf("execution=%v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
				}
			} else {
				exit, ok := result.runErr.(*exec.ExitError)
				if !ok || exit.ExitCode() != 1 || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), test.wantError) {
					t.Fatalf("execution=%v stdout=%q stderr=%q", result.runErr, result.stdout.String(), result.stderr.String())
				}
			}
			if err := checkSettingsUnchanged(fixture.work, test.input); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExportEditValidatePlanApply(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	remote := map[string]any{"default_branch": "main", "description": "old", "homepage": "old-url", "has_issues": true, "has_projects": true, "has_wiki": true, "has_discussions": false, "web_commit_signoff_required": false, "is_template": false}
	var methods []string
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		methods = append(methods, r.Method)
		if servePRAccessFixtureRead(w, r) {
			return
		}
		if serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if r.Method == "GET" && r.URL.Path == "/repos/owner/current/labels" {
			fmt.Fprint(w, `[]`)
			return
		}
		if r.Method == "GET" && r.URL.Path == "/repos/owner/current/topics" {
			fmt.Fprint(w, `{"names":[]}`)
			return
		}
		if r.URL.Path != "/repos/owner/current" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.Method == http.MethodPatch {
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(payload, map[string]any{"has_issues": false}) {
				t.Errorf("payload=%v", payload)
			}
			for key, value := range payload {
				remote[key] = value
			}
		} else if r.Method != http.MethodGet {
			t.Errorf("method=%s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
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
	exported, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export"}, false)
	if err != nil || exported.runErr != nil || exported.stdout.Len() != 0 || exported.stderr.Len() != 0 {
		t.Fatalf("export=%v result=%+v", err, exported)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github", "settings.yml"))
	if err != nil || !strings.Contains(string(data), "version: daiksud/gh-settings@v0") || !strings.Contains(string(data), "features:") || !strings.Contains(string(data), "issues: true") || strings.Contains(string(data), "has_issues") {
		t.Fatalf("exported file=%q error=%v", data, err)
	}
	// Edit to own only issues; omit the exported string/project/wiki fields.
	input := generalInput("features: {issues: false}\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"validate", "plan"} {
		result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
		if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
			t.Fatalf("%s=%v result=%+v", command, err, result)
		}
		want := ""
		if command == "plan" {
			want = "repository.settings.general.features.issues: true → false\n"
		}
		if result.stdout.String() != want {
			t.Fatalf("%s output=%q", command, result.stdout.String())
		}
	}
	mu.Lock()
	remote["description"] = "external change"
	remote["has_issues"] = false // Even if already equal after plan, apply must send the declaration.
	mu.Unlock()
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || result.runErr != nil || result.stdout.String() != "Applied repository settings.\n" || result.stderr.Len() != 0 {
		t.Fatalf("apply=%v result=%+v", err, result)
	}
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(methods, []string{"GET", "POST", "POST", "POST", "POST", "GET", "GET", "GET", "PATCH"}) || !reflect.DeepEqual(remote, map[string]any{"default_branch": "main", "description": "external change", "homepage": "old-url", "has_issues": false, "has_projects": true, "has_wiki": true, "has_discussions": false, "web_commit_signoff_required": false, "is_template": false}) {
		t.Fatalf("methods=%v remote=%v", methods, remote)
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
}
