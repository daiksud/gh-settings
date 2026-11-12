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
)

func TestPullRequestLimitsProcess(t *testing.T) {
	binary := acceptanceBinary(t)
	const endpoint = "/repos/owner/current/interaction-limits/pulls/creation-cap"
	const prefix = "repository.settings.interaction_limits.pull_request_limits."
	const enabledField = "limit_open_pull_requests_from_users_without_write_access"
	const maximumField = "maximum_open_pull_requests_per_user"
	const draftsField = "count_draft_pull_requests_toward_the_limit"
	var mu sync.Mutex
	state := map[string]any{"enabled": true, "max_open_pull_requests": float64(10), "include_drafts": true}
	var requests, all []string
	var expectedPatch map[string]any
	responseMode := "complete"
	readbackMode := "complete"
	staleRemaining := 0
	var baseline map[string]any
	patched := false
	clone := func(src map[string]any) map[string]any {
		dst := map[string]any{}
		for k, v := range src {
			dst[k] = v
		}
		return dst
	}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		all = append(all, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == endpoint {
			requests = append(requests, r.Method+" "+r.URL.Path)
			if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
				t.Error("wrong pull request limits API version")
			}
			if r.Method == "PATCH" {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, expectedPatch) {
					t.Errorf("PATCH=%v want=%v err=%v", body, expectedPatch, err)
					w.WriteHeader(400)
					return
				}
				baseline = clone(state)
				for k, v := range body {
					state[k] = v
				}
				patched = true
				fmt.Fprint(w, `{}`)
				return
			}
			if r.Method != "GET" {
				t.Error("unexpected method", r.Method)
				w.WriteHeader(400)
				return
			}
			response := clone(state)
			mode := responseMode
			if patched {
				mode = readbackMode
				if staleRemaining > 0 {
					response = clone(baseline)
					staleRemaining--
				}
				if mode == "drift" {
					response["include_drafts"] = !baseline["include_drafts"].(bool)
				}
			}
			switch mode {
			case "missing":
				delete(response, "include_drafts")
			case "malformed":
				response["enabled"] = "false"
			case "denied":
				w.WriteHeader(403)
				fmt.Fprint(w, `{"message":"denied"}`)
				return
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
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=owner", "SETTINGS_TRIGGERING_ACTOR=owner", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	declaration := func(fields string) string {
		return "version: daiksud/gh-settings@v0\nrepository: {settings: {interaction_limits: {pull_request_limits: {" + fields + "}}}}\n"
	}
	write := func(input string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args []string, want string, fail bool) string {
		t.Helper()
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || (result.runErr != nil) != fail || !fail && (result.stdout.String() != want || hasNonInfoDiagnostic(result.stderr.String())) || fail && result.stdout.Len() != 0 {
			t.Fatalf("args=%v exit=%v stdout=%q stderr=%q err=%v", args, result.runErr, result.stdout.String(), result.stderr.String(), err)
		}
		return withoutExportInfo(result.stderr.String())
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
		patched = false
	}
	// Omission and invalid local input never acquire the cap endpoint.
	write("version: daiksud/gh-settings@v0\nrepository: {}\n")
	run([]string{"plan"}, "No managed repository settings.\n", false)
	run([]string{"apply"}, "No managed repository settings.\n", false)
	mu.Lock()
	if len(all) != 0 {
		t.Error("empty config contacted API", all)
	}
	mu.Unlock()
	expect()
	for _, fields := range []string{enabledField + ": null", maximumField + ": 0", maximumField + ": 1001", draftsField + ": 'false'"} {
		write(declaration(fields))
		run([]string{"validate"}, "", true)
		run([]string{"apply"}, "", true)
		mu.Lock()
		if len(all) != 0 {
			t.Error("invalid config contacted API", all)
		}
		mu.Unlock()
		expect()
	}
	input := declaration(enabledField + ": false, " + maximumField + ": 25, " + draftsField + ": false")
	write(input)
	run([]string{"validate"}, "", false)
	expect()
	wantPlan := prefix + enabledField + ": true → false\n" + prefix + maximumField + ": 10 → 25\n" + prefix + draftsField + ": true → false\n"
	run([]string{"plan"}, wantPlan, false)
	expect("GET " + endpoint)
	report := filepath.Join(fixture.root, "pull-request-limits-plan.json")
	run([]string{"action", "plan", report}, "", false)
	expect("GET " + endpoint)
	raw, err := os.ReadFile(report)
	var wire map[string]any
	if err != nil || json.Unmarshal(raw, &wire) != nil || wire["managed"] != float64(3) || wire["outcome"] != "changed" {
		t.Fatal(string(raw), err)
	}
	changes, ok := wire["pull_request_limits"].([]any)
	wantChanges := []any{
		map[string]any{"field": prefix + enabledField, "before": true, "after": false},
		map[string]any{"field": prefix + maximumField, "before": float64(10), "after": float64(25)},
		map[string]any{"field": prefix + draftsField, "before": true, "after": false},
	}
	if !ok || !reflect.DeepEqual(changes, wantChanges) {
		t.Fatal(string(raw))
	}
	summary := filepath.Join(fixture.root, "pull-request-limits-summary.md")
	run([]string{"action", "summary", report, summary}, "", false)
	mu.Lock()
	if len(all) != 0 {
		t.Error("Summary contacted API", all)
	}
	mu.Unlock()
	expect()
	content, err := os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(wantPlan), "\n") {
		if !strings.Contains(html.UnescapeString(string(content)), line) {
			t.Fatal("missing Summary change", line, string(content))
		}
	}
	mu.Lock()
	expectedPatch = map[string]any{"enabled": false, "max_open_pull_requests": float64(25), "include_drafts": false}
	mu.Unlock()
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint)
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET " + endpoint)
	run([]string{"plan"}, "No differences for managed repository settings.\n", false)
	expect("GET " + endpoint)
	// A maximum-only declaration preserves enabled and drafts. Exact baseline
	// readbacks may be stale, but the write itself must never be retried.
	write(declaration(maximumField + ": 30"))
	mu.Lock()
	expectedPatch = map[string]any{"enabled": false, "max_open_pull_requests": float64(30)}
	staleRemaining = 2
	mu.Unlock()
	run([]string{"apply"}, "Applied repository settings.\n", false)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint, "GET "+endpoint, "GET "+endpoint)
	// Exhausting three exact-baseline observations fails without another PATCH.
	write(declaration(maximumField + ": 35"))
	mu.Lock()
	expectedPatch = map[string]any{"enabled": false, "max_open_pull_requests": float64(35)}
	staleRemaining = 3
	mu.Unlock()
	diagnostic := run([]string{"apply"}, "", true)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint, "GET "+endpoint, "GET "+endpoint)
	if !strings.Contains(diagnostic, "may already have applied") {
		t.Fatal(diagnostic)
	}
	// Changing an unmanaged companion is not a stale observation and fails now.
	write(declaration(maximumField + ": 40"))
	mu.Lock()
	expectedPatch = map[string]any{"enabled": false, "max_open_pull_requests": float64(40)}
	readbackMode = "drift"
	mu.Unlock()
	diagnostic = run([]string{"apply"}, "", true)
	expect("GET "+endpoint, "PATCH "+endpoint, "GET "+endpoint)
	if !strings.Contains(diagnostic, "may already have applied") {
		t.Fatal(diagnostic)
	}
	mu.Lock()
	readbackMode = "complete"
	mu.Unlock()
	// Explicit opt-in exports a complete disabled triple, including native false.
	run([]string{"export", "--pull-request-limits"}, "", false)
	expect("GET " + endpoint)
	exported, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []string{enabledField + ": false", maximumField + ": 40", draftsField + ": false"} {
		if !strings.Contains(string(exported), item) {
			t.Fatal(item, string(exported))
		}
	}
	run([]string{"validate"}, "", false)
	expect()
	for _, mode := range []string{"missing", "malformed", "denied"} {
		mu.Lock()
		responseMode = mode
		mu.Unlock()
		run([]string{"export", "--pull-request-limits"}, "", true)
		expect("GET " + endpoint)
		if err := checkSettingsUnchanged(fixture.work, string(exported)); err != nil {
			t.Fatal(err)
		}
	}
	run([]string{"export"}, "", false)
	expect()
	ordinary, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || strings.Contains(string(ordinary), "pull_request_limits") {
		t.Fatal(string(ordinary), err)
	}
}
