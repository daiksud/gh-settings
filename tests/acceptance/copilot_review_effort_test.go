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

func TestCopilotReviewEffortProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	endpoint := "/repos/owner/current/copilot/code-review/settings"
	var mu sync.Mutex
	var configured any
	denied, mismatch := false, false
	available := []string{"lite", "balanced", "max"}
	var requests, all []string
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		all = append(all, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == endpoint {
			requests = append(requests, r.Method+" "+r.URL.Path)
			if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
				t.Error("wrong API version")
			}
			if denied {
				w.WriteHeader(403)
				fmt.Fprint(w, `{}`)
				return
			}
			if r.Method == "PATCH" {
				var payload map[string]any
				if json.NewDecoder(r.Body).Decode(&payload) != nil || len(payload) != 1 {
					t.Error("wrong payload")
					w.WriteHeader(400)
					return
				}
				value, ok := payload["review_effort"].(string)
				if !ok || value != "inherit" && value != "lite" && value != "balanced" {
					t.Error("wrong effort", payload)
					w.WriteHeader(400)
					return
				}
				if !mismatch {
					if value == "inherit" {
						configured = nil
					} else {
						configured = value
					}
				}
			} else if r.Method != "GET" {
				t.Error("wrong method")
				w.WriteHeader(400)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"review_effort": map[string]any{"configured": configured, "effective": "balanced"}, "available_review_efforts": available, "severity_filter": "no_filter", "custom_instructions_enabled": false, "mcp_enabled": true})
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
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current")
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	declaration := func(value string) string {
		return "version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {review_effort_level: " + value + "}}}}\n"
	}
	run := func(args []string, want string, fail bool) string {
		t.Helper()
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (result.runErr != nil) != fail || !fail && (result.stdout.String() != want || result.stderr.Len() != 0) || fail && result.stdout.Len() != 0 {
			t.Fatalf("args=%v exit=%v stdout=%q stderr=%q err=%v", args, result.runErr, result.stdout.String(), result.stderr.String(), err)
		}
		return result.stderr.String()
	}
	expect := func(want ...string) {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if !reflect.DeepEqual(requests, want) {
			t.Fatalf("requests=%v want=%v", requests, want)
		}
		requests = nil
		all = nil
	}
	write := func(value string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, declaration(value)); err != nil {
			t.Fatal(err)
		}
	}
	write("balanced")
	run([]string{"plan"}, "repository.settings.copilot.code_review.review_effort_level: default → balanced\n", false)
	expect("GET " + endpoint)
	report := filepath.Join(fixture.root, "effort-plan.json")
	run([]string{"action", "plan", report, "status-only"}, "", false)
	expect("GET " + endpoint)
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if json.Unmarshal(raw, &wire) != nil || wire["managed"] != float64(1) || wire["copilot_review_effort"] == nil {
		t.Fatal(string(raw))
	}
	summary := filepath.Join(fixture.root, "effort-summary.md")
	run([]string{"action", "summary", report, summary, "status-only"}, "plan-outcome=changed\n", false)
	expect()
	content, err := os.ReadFile(summary)
	if err != nil || strings.Contains(string(content), "review_effort_level") || !strings.Contains(string(content), "changed") {
		t.Fatal(string(content), err)
	}
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint)
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET " + endpoint)
	write("default")
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint)
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET " + endpoint)
	write("lite")
	mu.Lock()
	available = []string{"balanced"}
	mu.Unlock()
	run([]string{"apply"}, "", true)
	expect("GET " + endpoint)
	mu.Lock()
	available = []string{"lite", "balanced", "max"}
	mismatch = true
	mu.Unlock()
	diagnostic := run([]string{"apply"}, "", true)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint)
	if !strings.Contains(diagnostic, "may already have applied") || !strings.Contains(diagnostic, "review_effort_level") {
		t.Fatal(diagnostic)
	}
	mu.Lock()
	mismatch = false
	mu.Unlock()
	run([]string{"export", "--copilot-review-effort", "--copilot-custom-instructions"}, "", false)
	expect("GET "+endpoint, "GET "+endpoint)
	output, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(output), "review_effort_level: default") || !strings.Contains(string(output), "use_custom_instructions: false") {
		t.Fatal(string(output), err)
	}
	mu.Lock()
	configured = "max"
	mu.Unlock()
	run([]string{"export", "--copilot-review-effort"}, "", true)
	expect("GET " + endpoint)
	if err := checkSettingsUnchanged(fixture.work, string(output)); err != nil {
		t.Fatal(err)
	}
	// Instructions remain independently selectable even when effort is unsupported.
	run([]string{"export", "--copilot-custom-instructions"}, "", false)
	expect("GET " + endpoint)
	output, err = os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || strings.Contains(string(output), "review_effort_level") {
		t.Fatal(string(output), err)
	}
	mu.Lock()
	denied = true
	mu.Unlock()
	run([]string{"export"}, "", false)
	expect()
	write("balanced")
	run([]string{"action", "plan", report, "status-only"}, "", true)
	expect("GET " + endpoint)
	raw, err = os.ReadFile(report)
	if err != nil || json.Unmarshal(raw, &wire) != nil || wire["resource"] != "repository.settings.copilot.code_review.review_effort_level" || wire["outcome"] != "read-failure" {
		t.Fatal(string(raw), err)
	}
	for _, invalid := range []string{"null", "max", "inherit", "false"} {
		write(invalid)
		run([]string{"apply"}, "", true)
		mu.Lock()
		if len(all) != 0 {
			t.Fatal("invalid declaration contacted API", all)
		}
		mu.Unlock()
	}
}
