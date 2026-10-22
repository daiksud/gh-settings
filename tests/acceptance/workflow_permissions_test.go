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

func TestWorkflowPermissionsExportPlanApplyAndSummary(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	policy := map[string]any{"default_workflow_permissions": "read", "can_approve_pull_request_reviews": false}
	execution := map[string]any{"enabled": false, "allowed_actions": "all", "sha_pinning_required": true}
	var writes []map[string]any
	var operations []string
	executionReads := 0
	workflowReads := 0
	deny := false
	incomplete := false
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/repos/owner/current/actions/permissions") {
			operations = append(operations, r.Method+" "+r.URL.Path)
			workflow := r.URL.Path == "/repos/owner/current/actions/permissions/workflow"
			if !workflow && r.URL.Path != "/repos/owner/current/actions/permissions" {
				t.Error("unexpected endpoint", r.URL)
				w.WriteHeader(400)
				return
			}
			target := execution
			if workflow {
				target = policy
			}
			switch r.Method {
			case "GET":
				if workflow {
					workflowReads++
					if incomplete {
						fmt.Fprint(w, `{"default_workflow_permissions":"read"}`)
						return
					}
				} else {
					executionReads++
				}
				json.NewEncoder(w).Encode(target)
			case "PUT":
				if workflow && deny {
					w.WriteHeader(409)
					fmt.Fprint(w, `{"message":"prevented by owning organization"}`)
					return
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if workflow {
					writes = append(writes, body)
				}
				for key, value := range body {
					target[key] = value
				}
				w.WriteHeader(204)
			default:
				t.Error("unexpected method", r.Method)
				w.WriteHeader(400)
			}
			return
		}
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if r.Method != "GET" {
			t.Error("unexpected mutation", r.Method, r.URL)
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
			t.Error("unexpected request", r.URL)
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
	run("export")
	run("export", "--actions")
	mu.Lock()
	if workflowReads != 0 {
		t.Fatal("old export scope acquired workflow policy")
	}
	beforeExecution := executionReads
	mu.Unlock()
	run("export", "--workflow-permissions")
	run("validate")
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "can_approve_pull_request_reviews: false") || strings.Contains(string(data), "sha_pinning_required:") {
		t.Fatal(string(data), err)
	}
	mu.Lock()
	if executionReads != beforeExecution {
		t.Fatal("workflow export acquired execution policy")
	}
	mu.Unlock()
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	input := actionsPolicyInput(`{default_workflow_permissions: write, can_approve_pull_request_reviews: true}`)
	writeSettingsFile(fixture.work, input)
	plan := run("plan")
	if !strings.Contains(plan, `settings.actions.default_workflow_permissions: "read" → "write"`) || !strings.Contains(plan, `settings.actions.can_approve_pull_request_reviews: false → true`) {
		t.Fatal(plan)
	}
	report := filepath.Join(fixture.root, "workflow-plan.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var typed map[string]any
	if err := json.Unmarshal(data, &typed); err != nil {
		t.Fatal(err)
	}
	if typed["managed"] != float64(2) || len(typed["actions"].([]any)) != 2 || len(typed["changes"].([]any)) != 0 {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "workflow-summary.md")
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
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	run("apply")
	writeSettingsFile(fixture.work, actionsPolicyInput(`{default_workflow_permissions: read}`))
	run("apply")
	writeSettingsFile(fixture.work, actionsPolicyInput(`{can_approve_pull_request_reviews: false}`))
	run("apply")
	mu.Lock()
	if executionReads != beforeExecution || len(writes) != 4 || !reflect.DeepEqual(writes[2], map[string]any{"default_workflow_permissions": "read"}) || !reflect.DeepEqual(writes[3], map[string]any{"can_approve_pull_request_reviews": false}) || policy["default_workflow_permissions"] != "read" || policy["can_approve_pull_request_reviews"] != false {
		t.Fatal(executionReads, writes, policy)
	}
	mu.Unlock()
	run("export", "--actions", "--workflow-permissions")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	// A later organization refusal retains the completed execution-policy write.
	input = actionsPolicyInput(`{enabled: true, default_workflow_permissions: write}`)
	writeSettingsFile(fixture.work, input)
	mu.Lock()
	deny = true
	operations = nil
	mu.Unlock()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
	if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "earlier resource writes remain applied") {
		t.Fatal(err, r.stderr.String())
	}
	mu.Lock()
	if !reflect.DeepEqual(operations, []string{"GET /repos/owner/current/actions/permissions", "PUT /repos/owner/current/actions/permissions", "GET /repos/owner/current/actions/permissions", "GET /repos/owner/current/actions/permissions/workflow", "PUT /repos/owner/current/actions/permissions/workflow"}) {
		t.Fatal(operations)
	}
	if execution["enabled"] != true || policy["default_workflow_permissions"] != "read" {
		t.Fatal("failure claimed rollback", execution, policy)
	}
	deny = false
	incomplete = true
	mu.Unlock()
	r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"export", "--workflow-permissions"}, false)
	if err != nil || r.runErr == nil {
		t.Fatal(err, r.stderr.String())
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	// Invalid local values never reach the endpoint.
	before := fixture.reads.Load()
	for _, body := range []string{`{default_workflow_permissions: unknown}`, `{default_workflow_permissions: null}`, `{can_approve_pull_request_reviews: null}`} {
		writeSettingsFile(fixture.work, actionsPolicyInput(body))
		for _, command := range []string{"validate", "plan", "apply"} {
			r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || r.runErr == nil {
				t.Fatal(body, command, err, r.stderr.String())
			}
		}
	}
	if fixture.reads.Load() != before {
		t.Fatal("invalid declaration contacted API")
	}
}
