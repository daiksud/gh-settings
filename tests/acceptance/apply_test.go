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
		name, input, payload, wantError                string
		status                                         int
		noAPI, noOrigin, noAuth, nested, outputFailure bool
		gitSetup                                       func(string, []string) error
	}{
		{name: "partial-false", input: repositoryInput("has_issues: false\n"), payload: `{"has_issues":false}`, nested: true},
		{name: "all-zero-values", input: repositoryInput("description: ''\nhomepage: ''\nhas_issues: false\nhas_projects: false\nhas_wiki: false\n"), payload: `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false}`},
		{name: "true-and-escaped-string", input: repositoryInput("description: \"quote\\\"\\n\\\\\"\nhas_wiki: true\n"), payload: `{"description":"quote\"\n\\","has_wiki":true}`},
		{name: "rewritten-origin", input: repositoryInput("has_projects: true\n"), payload: `{"has_projects":true}`, gitSetup: setupRewrittenOrigin},
		{name: "aliased-origin", input: repositoryInput("homepage: ''\n"), payload: `{"homepage":""}`, gitSetup: setupAliasedOrigin},
		{name: "worktree-origin", input: repositoryInput("description: new\n"), payload: `{"description":"new"}`, gitSetup: setupWorktreeOrigin},
		{name: "no-managed-fields", input: "schemaVersion: 1\n", noAPI: true, noAuth: true, noOrigin: true},
		{name: "empty-settings", input: "schemaVersion: 1\nsettings: {}\n", noAPI: true, noAuth: true, noOrigin: true},
		{name: "empty-repository", input: "schemaVersion: 1\nsettings:\n  repository: {}\n", noAPI: true, noAuth: true, noOrigin: true},
		{name: "malformed-before-api", input: repositoryInput("has_issues: [\n"), wantError: "decode YAML", noAPI: true},
		{name: "invalid-before-origin-auth", input: repositoryInput("has_issues: 'false'\n"), wantError: "must be a boolean", noAPI: true, noOrigin: true, noAuth: true},
		{name: "unknown-key", input: repositoryInput("private: true\n"), wantError: "unknown setting", noAPI: true},
		{name: "duplicate-key", input: repositoryInput("has_wiki: true\nhas_wiki: false\n"), wantError: "duplicate key", noAPI: true},
		{name: "explicit-null", input: repositoryInput("description: null\n"), wantError: "must be a string", noAPI: true},
		{name: "missing-version", input: "settings: {}\n", wantError: "missing schemaVersion", noAPI: true},
		{name: "unsupported-version", input: "schemaVersion: 2\n", wantError: "unsupported schemaVersion", noAPI: true},
		{name: "multiple-documents", input: "schemaVersion: 1\n---\nschemaVersion: 1\n", wantError: "one YAML document", noAPI: true},
		{name: "missing-file", wantError: "read .github/settings.yml", noAPI: true},
		{name: "missing-origin", input: repositoryInput("has_wiki: false\n"), wantError: "a GitHub origin remote is required", noAPI: true, noOrigin: true},
		{name: "authentication", input: repositoryInput("has_wiki: false\n"), wantError: "configure GitHub API client", noAPI: true, noAuth: true},
		{name: "unauthorized", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, status: 401, wantError: "401"},
		{name: "forbidden", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, status: 403, wantError: "403"},
		{name: "server-validation", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, status: 422, wantError: "422"},
		{name: "redirect-301", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, status: 301, wantError: "301"},
		{name: "redirect-302", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, status: 302, wantError: "302"},
		{name: "redirect-303", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, status: 303, wantError: "303"},
		{name: "redirect-307", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, status: 307, wantError: "307"},
		{name: "redirect-308", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, status: 308, wantError: "308"},
		{name: "output-after-patch", input: repositoryInput("has_wiki: false\n"), payload: `{"has_wiki":false}`, outputFailure: true, wantError: "write output:"},
		{name: "no-op-output-failure", input: "schemaVersion: 1\n", noAPI: true, noAuth: true, noOrigin: true, outputFailure: true, wantError: "write output:"},
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
				fmt.Fprint(w, `{"message":"fixture response"}`)
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
	remote := map[string]any{"description": "old", "homepage": "old-url", "has_issues": true, "has_projects": true, "has_wiki": true}
	var methods []string
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		methods = append(methods, r.Method)
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
	if err != nil || exported.runErr != nil || !strings.Contains(exported.stdout.String(), "has_issues: true") || exported.stderr.Len() != 0 {
		t.Fatalf("export=%v result=%+v", err, exported)
	}
	// Edit to own only issues; omit the exported string/project/wiki fields.
	input := repositoryInput("has_issues: false\n")
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
			want = "settings.repository.has_issues: true → false\n"
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
	if !reflect.DeepEqual(methods, []string{"GET", "GET", "PATCH"}) || !reflect.DeepEqual(remote, map[string]any{"description": "external change", "homepage": "old-url", "has_issues": false, "has_projects": true, "has_wiki": true}) {
		t.Fatalf("methods=%v remote=%v", methods, remote)
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
}
