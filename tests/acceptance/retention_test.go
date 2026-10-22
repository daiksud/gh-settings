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

func TestRetentionExportPlanApplyAndSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	days, maximum := 90, 90
	workflow := "read"
	mode := ""
	retentionReads, executionReads, workflowReads, redirected := 0, 0, 0, 0
	writes := []map[string]any{}
	events := []string{}
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
		case "/repos/owner/current/actions/permissions/artifact-and-log-retention":
			if r.Method == "GET" {
				retentionReads++
				switch mode {
				case "hidden":
					fmt.Fprintf(w, `{"days":%d}`, days)
					return
				case "denied":
					w.WriteHeader(403)
					fmt.Fprint(w, `{"message":"unavailable"}`)
					return
				case "get-redirect":
					w.Header().Set("Location", "/repos/other/redirected")
					w.WriteHeader(301)
					return
				}
				fmt.Fprintf(w, `{"days":%d,"maximum_allowed_days":%d}`, days, maximum)
				return
			}
			if r.Method != "PUT" {
				t.Error("unexpected retention mutation", r.Method)
				w.WriteHeader(400)
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body) != 1 || body["days"] == nil {
				t.Error("readonly/unrelated fields in retention payload", body)
			}
			writes = append(writes, body)
			events = append(events, "PUT retention")
			if mode == "put-redirect" {
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
				days = int(body["days"].(float64))
			}
			w.WriteHeader(204)
			return
		case "/repos/owner/current/actions/permissions/workflow":
			if r.Method == "GET" {
				workflowReads++
				fmt.Fprintf(w, `{"default_workflow_permissions":%q,"can_approve_pull_request_reviews":false}`, workflow)
				return
			}
			if r.Method != "PUT" {
				t.Error(r.Method)
				w.WriteHeader(400)
				return
			}
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if !reflect.DeepEqual(body, map[string]any{"default_workflow_permissions": "write"}) {
				t.Error(body)
			}
			workflow = "write"
			events = append(events, "PUT workflow")
			w.WriteHeader(204)
			return
		case "/repos/owner/current/actions/permissions":
			if r.Method != "GET" {
				t.Error("execution policy mutated", r.Method)
				w.WriteHeader(400)
				return
			}
			executionReads++
			fmt.Fprint(w, `{"enabled":false,"allowed_actions":"all","sha_pinning_required":true}`)
			return
		}
		if r.Method != "GET" {
			t.Error("artifact/run/status/unrelated mutation", r.Method, r.URL)
			w.WriteHeader(400)
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
			t.Error("unrelated endpoint", r.URL)
			w.WriteHeader(400)
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
		if err := writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings:\n  actions: "+value+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(fixture.work, ".github/settings.yml")
	run("export")
	run("export", "--actions")
	run("export", "--workflow-permissions")
	mu.Lock()
	if retentionReads != 0 {
		t.Fatal("existing export choices read retention")
	}
	beforeExecution, beforeWorkflow := executionReads, workflowReads
	mu.Unlock()
	run("export", "--retention")
	run("validate")
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "artifact_and_log_retention_days: 90") || strings.Contains(string(data), "maximum_allowed_days") || strings.Contains(string(data), "default_workflow_permissions:") {
		t.Fatal(string(data), err)
	}
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	write("{artifact_and_log_retention_days: 14}")
	plan := run("plan")
	if !strings.Contains(plan, "settings.actions.artifact_and_log_retention_days: 90 → 14") {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "retention-plan.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var typed map[string]any
	if err := json.Unmarshal(data, &typed); err != nil {
		t.Fatal(err)
	}
	changes := typed["actions"].([]any)
	entry := changes[0].(map[string]any)
	if typed["managed"] != float64(1) || len(changes) != 1 || entry["before"] != float64(90) || entry["after"] != float64(14) {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "retention-summary.md")
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
	mu.Lock()
	if days != 14 || len(writes) != 1 || !reflect.DeepEqual(writes[0], map[string]any{"days": float64(14)}) || executionReads != beforeExecution || workflowReads != beforeWorkflow {
		t.Fatal(days, writes, executionReads, workflowReads)
	}
	mu.Unlock()
	// A fresh lower parent cap fails before PUT, even when the old days match.
	mu.Lock()
	maximum = 7
	before := len(writes)
	mu.Unlock()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "exceeds repository maximum_allowed_days 7") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	if len(writes) != before || days != 14 {
		t.Fatal("cap failure wrote/clamped", days, writes)
	}
	mu.Unlock()
	write("{artifact_and_log_retention_days: 7}")
	run("apply")
	mu.Lock()
	if days != 7 {
		t.Fatal("known over-cap policy could not be repaired", days)
	}
	maximum = 90
	mu.Unlock()
	// Incomplete/denied/redirected reads preserve the existing file.
	for _, failure := range []string{"hidden", "denied", "get-redirect"} {
		mu.Lock()
		mode = failure
		before = len(writes)
		mu.Unlock()
		original, _ := os.ReadFile(path)
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--retention"}, false)
		after, _ := os.ReadFile(path)
		if err != nil || r.runErr == nil || r.stdout.Len() != 0 || string(after) != string(original) {
			t.Fatal(failure, err, r.stderr.String(), string(after))
		}
		mu.Lock()
		if len(writes) != before || redirected != 0 {
			t.Fatal("read failure wrote/followed redirect", failure, writes, redirected)
		}
		mu.Unlock()
	}
	for _, failure := range []string{"put-redirect", "enforced"} {
		mu.Lock()
		mode = failure
		before = len(writes)
		mu.Unlock()
		write("{artifact_and_log_retention_days: 14}")
		r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
		if err != nil || r.runErr == nil {
			t.Fatal(failure, err, r.stderr.String())
		}
		mu.Lock()
		if days != 7 || len(writes) != before+1 || redirected != 0 {
			t.Fatal("failed write changed/followed/retried", failure, days, writes, redirected)
		}
		mu.Unlock()
	}
	mu.Lock()
	mode = ""
	mu.Unlock()
	run("export", "--retention", "--actions", "--workflow-permissions")
	data, _ = os.ReadFile(path)
	for _, field := range []string{"artifact_and_log_retention_days: 7", "enabled: false", "default_workflow_permissions: read"} {
		if !strings.Contains(string(data), field) {
			t.Fatal("composed resource lost", field, string(data))
		}
	}
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	write("{default_workflow_permissions: write, artifact_and_log_retention_days: 30}")
	mu.Lock()
	mode = "deny-write"
	events = nil
	mu.Unlock()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "earlier successful writes remain applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	if workflow != "write" || days != 7 || !reflect.DeepEqual(events, []string{"PUT workflow", "PUT retention"}) {
		t.Fatal("partial progress lost/retried", workflow, days, events)
	}
	mu.Unlock()
}
