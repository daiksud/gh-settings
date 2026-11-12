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

func TestCopilotMCPProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	endpoint := "/repos/owner/current/copilot/code-review/settings"
	var mu sync.Mutex
	enabled, denied, redirect, mismatch := true, false, false, false
	responseMode := "complete"
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
				value, ok := body["mcp_enabled"].(bool)
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
			response := map[string]any{"mcp_enabled": enabled, "custom_instructions_enabled": false, "severity_filter": "no_filter", "review_effort": map[string]any{"configured": nil, "effective": "balanced"}, "available_review_efforts": []string{"lite", "balanced", "max"}}
			if responseMode == "missing" {
				delete(response, "mcp_enabled")
			} else if responseMode == "malformed" {
				response["mcp_enabled"] = "false"
			}
			json.NewEncoder(w).Encode(response)
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
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current",
		"SETTINGS_ACTOR=owner", "SETTINGS_TRIGGERING_ACTOR=owner", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1",
		"SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40),
		"SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=v0.6.1", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	declaration := func(value string) string {
		return "version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: " + value + "}}}}\n"
	}
	run := func(args []string, want string, fail bool) (string, string) {
		t.Helper()
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (result.runErr != nil) != fail || !fail && (result.stdout.String() != want || hasNonInfoDiagnostic(result.stderr.String())) || fail && result.stdout.Len() != 0 {
			t.Fatalf("args=%v exit=%v stdout=%q stderr=%q err=%v", args, result.runErr, result.stdout.String(), result.stderr.String(), err)
		}
		return result.stdout.String(), withoutExportInfo(result.stderr.String())
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
	run([]string{"plan"}, "repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: true → false\n", false)
	expect("GET " + endpoint)
	report := filepath.Join(fixture.root, "mcp-plan.json")
	run([]string{"action", "plan", report}, "", false)
	expect("GET " + endpoint)
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if json.Unmarshal(raw, &wire) != nil || wire["managed"] != float64(1) || wire["outcome"] != "changed" {
		t.Fatal(string(raw))
	}
	changes, ok := wire["copilot_mcp"].([]any)
	if !ok || len(changes) != 1 {
		t.Fatal(string(raw))
	}
	change, ok := changes[0].(map[string]any)
	if !ok || change["field"] != "repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests" || change["before"] != true || change["after"] != false {
		t.Fatal(string(raw))
	}
	summary := filepath.Join(fixture.root, "mcp-summary.md")
	run([]string{"action", "summary", report, summary}, "", false)
	expect()
	content, err := os.ReadFile(summary)
	if err != nil || !strings.Contains(string(content), "allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests") {
		t.Fatal(string(content), err)
	}
	// Each control remains independently managed even though all use one endpoint.
	combined := "version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {use_custom_instructions: false, review_effort_level: default, allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false}}}}\n"
	if err := writeSettingsFile(fixture.work, combined); err != nil {
		t.Fatal(err)
	}
	run([]string{"action", "plan", report}, "", false)
	expect("GET "+endpoint, "GET "+endpoint, "GET "+endpoint)
	raw, err = os.ReadFile(report)
	wire = nil
	if err != nil || json.Unmarshal(raw, &wire) != nil || wire["managed"] != float64(3) || wire["outcome"] != "changed" || wire["copilot_mcp"] == nil || wire["copilot_custom_instructions"] != nil || wire["copilot_review_effort"] != nil {
		t.Fatal(string(raw), err)
	}
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
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
	if !strings.Contains(diagnostic, "repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests") || !strings.Contains(diagnostic, "may already have applied") {
		t.Fatal(diagnostic)
	}
	mu.Lock()
	mismatch = false
	enabled = false
	mu.Unlock()
	run([]string{"export", "--copilot-mcp"}, "", false)
	expect("GET " + endpoint)
	output, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(output), "allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false") || strings.Contains(string(output), "use_custom_instructions") || strings.Contains(string(output), "review_effort_level") {
		t.Fatal(string(output), err)
	}
	for _, mode := range []string{"missing", "malformed"} {
		mu.Lock()
		responseMode = mode
		mu.Unlock()
		run([]string{"export", "--copilot-mcp"}, "", true)
		expect("GET " + endpoint)
		if err := checkSettingsUnchanged(fixture.work, string(output)); err != nil {
			t.Fatal(err)
		}
	}
	mu.Lock()
	responseMode = "complete"
	mu.Unlock()
	run([]string{"export", "--copilot-mcp", "--copilot-custom-instructions", "--copilot-review-effort"}, "", false)
	expect("GET "+endpoint, "GET "+endpoint, "GET "+endpoint)
	output, err = os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(output), "allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false") || !strings.Contains(string(output), "use_custom_instructions: false") || !strings.Contains(string(output), "review_effort_level: default") {
		t.Fatal(string(output), err)
	}
	mu.Lock()
	denied = true
	mu.Unlock()
	run([]string{"export", "--copilot-mcp"}, "", true)
	expect("GET " + endpoint)
	if err := checkSettingsUnchanged(fixture.work, string(output)); err != nil {
		t.Fatal(err)
	}
	run([]string{"export"}, "", false)
	expect()
	if err := writeSettingsFile(fixture.work, declaration("false")); err != nil {
		t.Fatal(err)
	}
	report = filepath.Join(fixture.root, "private-plan.json")
	_, diagnostic = run([]string{"action", "plan", report, "status-only"}, "", true)
	expect("GET " + endpoint)
	if strings.Contains(diagnostic, "mcp") {
		t.Fatal("private failure path exposed", diagnostic)
	}
	raw, err = os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	wire = nil
	if json.Unmarshal(raw, &wire) != nil || wire["resource"] != "repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests" || wire["outcome"] != "read-failure" || wire["copilot_mcp"] != nil || wire["managed"] != float64(0) {
		t.Fatal(string(raw))
	}
	summary = filepath.Join(fixture.root, "private-summary.md")
	run([]string{"action", "summary", report, summary, "status-only"}, "", true)
	expect()
	content, err = os.ReadFile(summary)
	if err != nil || strings.Contains(string(content), "mcp") || !strings.Contains(string(content), "read-failure") {
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
