//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestCopilotInstructionsProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	endpoint := "/repos/owner/current/copilot/code-review/settings"
	var mu sync.Mutex
	enabled, denied, redirect, mismatch := true, false, false, false
	var requests, all []string
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		all = append(all, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == endpoint || r.URL.Path == "/unexpected" {
			requests = append(requests, r.Method+" "+r.URL.Path)
			if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
				t.Error("wrong Copilot API version")
			}
			if redirect {
				w.Header().Set("Location", "/unexpected")
				w.WriteHeader(307)
				return
			}
			if denied {
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"denied"}`)
				return
			}
			if r.Method == "PATCH" {
				var body map[string]any
				if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 1 {
					t.Error("unexpected PATCH members")
					w.WriteHeader(400)
					return
				}
				value, ok := body["custom_instructions_enabled"].(bool)
				if !ok {
					t.Error("missing native boolean PATCH")
					w.WriteHeader(400)
					return
				}
				if !mismatch {
					enabled = value
				}
			} else if r.Method != "GET" {
				t.Errorf("unexpected method %s", r.Method)
				w.WriteHeader(400)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"custom_instructions_enabled": enabled, "mcp_enabled": true, "severity_filter": "no_filter"})
			return
		}
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			fmt.Fprint(w, planResponse)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		default:
			t.Errorf("unexpected API %s %s", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false, "/copilot/code-review/settings", "/labels", "/topics")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current")
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	declaration := func(value string) string {
		return "version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {use_custom_instructions: " + value + "}}}}\n"
	}
	lastExport := false
	run := func(args []string, want string, fail bool) (string, string) {
		t.Helper()
		lastExport = len(args) > 0 && args[0] == "export"
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (result.runErr != nil) != fail || !fail && (result.stdout.String() != want || hasNonInfoDiagnostic(result.stderr.String())) || fail && result.stdout.Len() != 0 {
			t.Fatalf("args=%v exit=%v stdout=%q stderr=%q err=%v", args, result.runErr, result.stdout.String(), result.stderr.String(), err)
		}
		return result.stdout.String(), withoutExportInfo(result.stderr.String())
	}
	expect := func(want ...string) {
		if lastExport {
			want = []string{"GET " + endpoint, "GET " + endpoint, "GET " + endpoint}
		}
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if !reflect.DeepEqual(requests, want) {
			t.Fatalf("requests=%v want=%v", requests, want)
		}
		requests = nil
		all = nil
	}
	input := "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {issues: false}}}}\n"
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"plan"}, "repository.settings.general.features.issues: true → false\n", false)
	mu.Lock()
	if len(all) == 0 {
		t.Fatal("unrelated managed setting was not observed")
	}
	mu.Unlock()
	expect()
	input = declaration("false")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"plan"}, "repository.settings.copilot.code_review.use_custom_instructions: true → false\n", false)
	expect("GET " + endpoint)
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint)
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET " + endpoint)
	input = declaration("true")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint)
	run([]string{"plan"}, "No differences for managed repository settings.\n", false)
	expect("GET " + endpoint)
	mu.Lock()
	mismatch = true
	mu.Unlock()
	input = declaration("false")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	_, diagnostic := run([]string{"apply"}, "", true)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint)
	if !strings.Contains(diagnostic, "repository.settings.copilot.code_review.use_custom_instructions") || !strings.Contains(diagnostic, "may already have applied") {
		t.Fatal(diagnostic)
	}
	mu.Lock()
	mismatch = false
	enabled = false
	mu.Unlock()
	run([]string{"export"}, "", false)
	expect("GET " + endpoint)
	output, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(output), "use_custom_instructions: false") {
		t.Fatal(string(output), err)
	}
	mu.Lock()
	denied = true
	mu.Unlock()
	run([]string{"export"}, "", false)
	expect("GET " + endpoint)
	if err := checkSettingsUnchanged(fixture.work, string(output)); err != nil {
		t.Fatal(err)
	}
	run([]string{"export"}, "", false)
	expect()
	if err := writeSettingsFile(fixture.work, declaration("false")); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(fixture.root, "private-plan.json")
	_, diagnostic = run([]string{"action", "plan", report, "status-only"}, "", true)
	expect("GET " + endpoint)
	if strings.Contains(diagnostic, "custom_instructions") {
		t.Fatal("private failure path exposed", diagnostic)
	}
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if json.Unmarshal(raw, &wire) != nil || wire["resource"] != "repository.settings.copilot.code_review.use_custom_instructions" || wire["outcome"] != "read-failure" {
		t.Fatal(string(raw))
	}
	summary := filepath.Join(fixture.root, "private-summary.md")
	run([]string{"action", "summary", report, summary, "status-only"}, "", true)
	expect()
	content, err := os.ReadFile(summary)
	if err != nil || strings.Contains(string(content), "custom_instructions") || !strings.Contains(string(content), "read-failure") {
		t.Fatal(string(content), err)
	}
	if err := writeSettingsFile(fixture.work, declaration("null")); err != nil {
		t.Fatal(err)
	}
	run([]string{"apply"}, "", true)
	mu.Lock()
	if len(all) != 0 {
		t.Fatal("invalid config contacted API", all)
	}
	denied = false
	redirect = true
	mu.Unlock()
	expect()
	if err := writeSettingsFile(fixture.work, declaration("false")); err != nil {
		t.Fatal(err)
	}
	run([]string{"plan"}, "", true)
	expect("GET " + endpoint)
}
