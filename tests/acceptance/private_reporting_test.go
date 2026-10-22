//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestPrivateReportingExportPlanApplyAndSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	enabled, scanning := false, false
	showSecret := false
	visibility := "public"
	mode := ""
	writes := []string{}
	statusReads, redirected := 0, 0
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/repos/other/redirected" {
			redirected++
			fmt.Fprint(w, `{}`)
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
			if r.Method == "GET" {
				var base map[string]any
				json.Unmarshal([]byte(planResponse), &base)
				base["visibility"] = visibility
				base["security_and_analysis"] = nil
				if showSecret {
					status := "disabled"
					if scanning {
						status = "enabled"
					}
					base["security_and_analysis"] = map[string]any{"secret_scanning": map[string]string{"status": status}, "secret_scanning_push_protection": map[string]string{"status": "disabled"}}
				}
				json.NewEncoder(w).Encode(base)
				return
			}
			if r.Method != "PATCH" {
				t.Error("unexpected repository mutation", r.Method)
				w.WriteHeader(400)
				return
			}
			var body map[string]map[string]map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body) != 1 || len(body["security_and_analysis"]) != 1 || body["security_and_analysis"]["secret_scanning"]["status"] != "enabled" {
				t.Error("unexpected Secret scanning payload", body)
			}
			scanning = true
			writes = append(writes, "PATCH secret_scanning")
			fmt.Fprint(w, `{}`)
			return
		case "/repos/owner/current/private-vulnerability-reporting":
			if r.Method == "GET" {
				statusReads++
				switch mode {
				case "hidden":
					fmt.Fprint(w, `{"enabled":null}`)
					return
				case "denied":
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"unavailable"}`)
					return
				case "status-redirect":
					w.Header().Set("Location", "/repos/other/redirected")
					w.WriteHeader(301)
					return
				}
				fmt.Fprintf(w, `{"enabled":%t}`, enabled)
				return
			}
			if r.Method != "PUT" && r.Method != "DELETE" {
				t.Error("unexpected reporting method", r.Method)
				w.WriteHeader(400)
				return
			}
			if r.Body != nil {
				body, _ := io.ReadAll(r.Body)
				if len(body) > 0 {
					t.Error("toggle sent unexpected body", string(body))
				}
			}
			writes = append(writes, r.Method+" private-vulnerability-reporting")
			if mode == "write-redirect" {
				w.Header().Set("Location", "/repos/other/redirected")
				w.WriteHeader(307)
				return
			}
			if mode == "deny-write" {
				w.WriteHeader(422)
				fmt.Fprint(w, `{"message":"upper policy refused"}`)
				return
			}
			if mode != "enforced" {
				enabled = r.Method == "PUT"
			}
			w.WriteHeader(204)
			return
		case "/repos/owner/current/topics":
			if r.Method == "GET" {
				fmt.Fprint(w, `{"names":[]}`)
				return
			}
		case "/repos/owner/current/labels":
			if r.Method == "GET" {
				fmt.Fprint(w, `[]`)
				return
			}
		}
		t.Error("advisory/report/notification/unrelated endpoint", r.Method, r.URL)
		w.WriteHeader(400)
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) string {
		t.Helper()
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || r.runErr != nil || r.stderr.Len() != 0 {
			t.Fatal(args, err, r.stderr.String())
		}
		return r.stdout.String()
	}
	write := func(value string) {
		t.Helper()
		if err := writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings:\n  security: "+value+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run("export")
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), "security:") {
		t.Fatal(string(data), err)
	}
	mu.Lock()
	if statusReads != 0 {
		t.Fatal("ordinary export read reporting policy")
	}
	mu.Unlock()
	// Hidden Secret scanning data does not block the independent Metadata reader.
	run("export", "--security", "private-vulnerability-reporting")
	run("validate")
	data, err = os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "private_vulnerability_reporting: false") || strings.Contains(string(data), "secret_scanning:") {
		t.Fatal(string(data), err)
	}
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	write("{private_vulnerability_reporting: true}")
	plan := run("plan")
	if !strings.Contains(plan, "settings.security.private_vulnerability_reporting: false → true") {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "private-reporting-plan.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var typed map[string]any
	if err := json.Unmarshal(data, &typed); err != nil {
		t.Fatal(err)
	}
	if typed["managed"] != float64(1) || len(typed["security"].([]any)) != 1 || len(typed["changes"].([]any)) != 0 {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "private-reporting-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64), "GH_TOKEN=", "GITHUB_TOKEN=")
	reads := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || r.runErr != nil || fixture.reads.Load() != reads {
		t.Fatal(err, r.stderr.String())
	}
	data, _ = os.ReadFile(summary)
	if !strings.Contains(html.UnescapeString(string(data)), strings.TrimSpace(plan)) {
		t.Fatal(string(data))
	}
	run("apply")
	run("apply")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	write("{private_vulnerability_reporting: false}")
	run("apply")
	run("apply")
	mu.Lock()
	if enabled || !reflect.DeepEqual(writes, []string{"PUT private-vulnerability-reporting", "DELETE private-vulnerability-reporting"}) {
		t.Fatal(enabled, writes)
	}
	mu.Unlock()
	for _, failure := range []string{"hidden", "denied", "status-redirect", "private", "internal"} {
		mu.Lock()
		mode = failure
		visibility = "public"
		if failure == "private" || failure == "internal" {
			visibility = failure
		}
		beforeWrites, beforeReads := len(writes), statusReads
		mu.Unlock()
		original, _ := os.ReadFile(path)
		r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--security", "private-vulnerability-reporting"}, false)
		after, _ := os.ReadFile(path)
		if err != nil || r.runErr == nil || r.stdout.Len() != 0 || string(after) != string(original) {
			t.Fatal(failure, err, r.stderr.String(), string(after))
		}
		// Explicit false on a nonpublic target remains inapplicable, not a no-op.
		if failure == "private" || failure == "internal" {
			r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "inapplicable") {
				t.Fatal(failure, err, r.stderr.String())
			}
		}
		mu.Lock()
		if len(writes) != beforeWrites || redirected != 0 || (failure == "private" || failure == "internal") && statusReads != beforeReads {
			t.Fatal("failure wrote/followed redirect/read inapplicable status", failure, writes, redirected, statusReads, beforeReads)
		}
		mu.Unlock()
	}
	mu.Lock()
	visibility = "public"
	mode = "enforced"
	before := len(writes)
	mu.Unlock()
	write("{private_vulnerability_reporting: true}")
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "did not retain requested") {
		t.Fatal("enforced no-change accepted", err, r.stderr.String())
	}
	mu.Lock()
	if enabled || len(writes) != before+1 {
		t.Fatal(enabled, writes)
	}
	mu.Unlock()
	// Check actual production PUT and DELETE redirect handling with no forwarding.
	for _, current := range []bool{false, true} {
		mu.Lock()
		enabled = current
		mode = "write-redirect"
		before = len(writes)
		mu.Unlock()
		write(fmt.Sprintf("{private_vulnerability_reporting: %t}", !current))
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
		if err != nil || r.runErr == nil {
			t.Fatal(current, err, r.stderr.String())
		}
		mu.Lock()
		if enabled != current || redirected != 0 || len(writes) != before+1 {
			t.Fatal("redirect changed state/followed/retried", enabled, redirected, writes)
		}
		mu.Unlock()
	}
	mu.Lock()
	enabled = false
	mode = ""
	showSecret = true
	mu.Unlock()
	for _, args := range [][]string{{"export", "--security", "private-vulnerability-reporting", "--security", "secret-scanning"}, {"export", "--security", "secret-scanning", "--security", "private-vulnerability-reporting"}} {
		run(args...)
		data, _ = os.ReadFile(path)
		for _, field := range []string{"secret_scanning: false", "secret_scanning_push_protection: false", "private_vulnerability_reporting: false"} {
			if !strings.Contains(string(data), field) {
				t.Fatal("selected family erased", field, string(data))
			}
		}
		if got := run("plan"); got != "No differences for managed repository settings.\n" {
			t.Fatal(got)
		}
	}
	write("{secret_scanning: true, private_vulnerability_reporting: true}")
	mu.Lock()
	mode = "deny-write"
	before = len(writes)
	mu.Unlock()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "earlier successful writes remain applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	if !scanning || enabled || !reflect.DeepEqual(writes[before:], []string{"PATCH secret_scanning", "PUT private-vulnerability-reporting"}) {
		t.Fatal("prior security progress lost or retried", scanning, enabled, writes)
	}
	mu.Unlock()
}
