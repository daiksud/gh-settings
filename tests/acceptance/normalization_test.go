//go:build acceptance

package acceptance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestWorkflowNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var permission any = "FUTURE"
	approval, keepUnknown := false, true
	execution := map[string]any{"enabled": false, "allowed_actions": "all", "sha_pinning_required": false}
	var workflowWrites, executionWrites []map[string]any
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/repos/owner/current/actions/permissions/workflow" {
			switch req.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{"default_workflow_permissions": permission, "can_approve_pull_request_reviews": approval})
			case "PUT":
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				workflowWrites = append(workflowWrites, payload)
				if value, present := payload["default_workflow_permissions"]; present && !keepUnknown {
					permission = value
				}
				if value, present := payload["can_approve_pull_request_reviews"]; present {
					approval = value.(bool)
				}
				w.WriteHeader(204)
			default:
				t.Error("unexpected workflow method", req.Method)
				w.WriteHeader(400)
			}
			return
		}
		if req.URL.Path == "/repos/owner/current/actions/permissions" {
			switch req.Method {
			case "GET":
				json.NewEncoder(w).Encode(execution)
			case "PUT":
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				executionWrites = append(executionWrites, payload)
				for key, value := range payload {
					execution[key] = value
				}
				w.WriteHeader(204)
			default:
				t.Error("unexpected execution method", req.Method)
				w.WriteHeader(400)
			}
			return
		}
		if servePRAccessFixtureRead(w, req) || serveIssueCreationPolicyFixtureRead(w, req) || serveSponsorshipsFixtureRead(w, req) {
			return
		}
		if isMergeMethodsFixtureRead(req) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if req.Method != "GET" {
			t.Error("unexpected remote mutation", req.Method, req.URL)
			w.WriteHeader(400)
			return
		}
		switch req.URL.Path {
		case "/repos/owner/current":
			fmt.Fprint(w, planResponse)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		default:
			t.Error("unexpected acquisition", req.URL)
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
	run := func(args ...string) (string, string, int) {
		t.Helper()
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil {
			t.Fatal("process fixture failure", err)
		}
		status := 0
		if result.runErr != nil {
			var failure *exec.ExitError
			if !errors.As(result.runErr, &failure) {
				t.Fatal("process launch failure", result.runErr)
			}
			status = failure.ExitCode()
		}
		return result.stdout.String(), result.stderr.String(), status
	}
	intent := func(body string) string {
		t.Helper()
		input := actionsPolicyInput(body)
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		return input
	}
	output, diagnostic, status := run("export", "--workflow-permissions")
	if status != 0 || output != "" || !strings.Contains(diagnostic, "Warning:") || !strings.Contains(diagnostic, "FUTURE") || !strings.Contains(diagnostic, "repository.settings.actions.general.workflow_permissions") || !strings.Contains(diagnostic, "read_repository_contents_and_packages_permissions") {
		t.Fatal("export normalization was not visible", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "workflow_permissions: read_repository_contents_and_packages_permissions") || strings.Contains(string(data), "FUTURE") {
		t.Fatal("exported YAML did not contain only UI fallback", string(data), err)
	}
	for _, choice := range []string{"read_repository_contents_and_packages_permissions", "read_and_write_permissions"} {
		input := intent("{workflow_permissions: " + choice + "}")
		output, diagnostic, status = run("plan")
		if status != 0 || diagnostic != "" || !strings.Contains(output, "Warning:") || !strings.Contains(output, "FUTURE") || !strings.Contains(output, choice) || !strings.Contains(output, "write required") || strings.Contains(output, "No differences") {
			t.Fatal("plan confused a projection with observation", choice, status, output, diagnostic)
		}
		if err := checkSettingsUnchanged(fixture.work, input); err != nil {
			t.Fatal(err)
		}
	}
	report, summary := filepath.Join(fixture.root, "normalization.json"), filepath.Join(fixture.root, "private-summary.md")
	output, diagnostic, status = run("action", "plan", report, "status-only")
	if status != 0 || output != "" || diagnostic != "" {
		t.Fatal("private planning disclosed a diagnostic", status, output, diagnostic)
	}
	output, diagnostic, status = run("action", "summary", report, summary, "status-only")
	data, err = os.ReadFile(summary)
	if err != nil || status != 0 || diagnostic != "" || output != "plan-outcome=changed\n" || strings.Contains(string(data), "FUTURE") || strings.Contains(string(data), "workflow_permissions") {
		t.Fatal("private summary disclosed normalized policy", status, output, diagnostic, string(data), err)
	}
	mu.Lock()
	readOnlyWrites := len(workflowWrites) + len(executionWrites)
	mu.Unlock()
	if readOnlyWrites != 0 {
		t.Fatal("export/plan normalized remote state", readOnlyWrites)
	}
	intent("{allow_github_actions_to_create_and_approve_pull_requests: false}")
	output, diagnostic, status = run("plan")
	if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
		t.Fatal("unmanaged permission was printed", status, output, diagnostic)
	}
	output, diagnostic, status = run("apply")
	if status != 0 || diagnostic != "" {
		t.Fatal("approval-only apply replayed an unmanaged fallback", status, output, diagnostic)
	}
	mu.Lock()
	preserved := permission == "FUTURE" && len(workflowWrites) == 1 && reflect.DeepEqual(workflowWrites[0], map[string]any{"can_approve_pull_request_reviews": false})
	mu.Unlock()
	if !preserved {
		t.Fatal("approval-only payload or raw companion changed")
	}
	input := intent("{workflow_permissions: read_repository_contents_and_packages_permissions}")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || !strings.Contains(diagnostic, "Warning:") || !strings.Contains(diagnostic, "may already have applied") {
		t.Fatal("normalized fallback falsely verified a write", status, output, diagnostic)
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	keepUnknown = false
	mu.Unlock()
	intent("{workflow_permissions: read_and_write_permissions}")
	output, diagnostic, status = run("apply")
	if status != 0 || !strings.Contains(diagnostic, "Warning:") || !strings.Contains(output, "Applied repository settings") {
		t.Fatal("canonical readback did not complete apply", status, output, diagnostic)
	}
	output, diagnostic, status = run("plan")
	if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
		t.Fatal("exact acquired choice retained a fallback warning", status, output, diagnostic)
	}
	mu.Lock()
	permission, keepUnknown = "FUTURE", true
	mu.Unlock()
	input = intent("{actions_permissions: allow_all_actions_and_reusable_workflows, workflow_permissions: read_repository_contents_and_packages_permissions}")
	output, diagnostic, status = run("apply")
	if status != 1 || !strings.Contains(diagnostic, "earlier resource writes remain applied") || !strings.Contains(diagnostic, "FUTURE") {
		t.Fatal("unverified readback lost earlier progress", status, output, diagnostic)
	}
	mu.Lock()
	retained := execution["enabled"] == true && len(executionWrites) == 1 && permission == "FUTURE" && reflect.DeepEqual(workflowWrites[len(workflowWrites)-1], map[string]any{"default_workflow_permissions": "read"})
	mu.Unlock()
	if !retained {
		t.Fatal("partial failure implied rollback or hid requested write")
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	permission = nil
	beforeWrites := len(workflowWrites) + len(executionWrites)
	mu.Unlock()
	input = intent("{workflow_permissions: read_repository_contents_and_packages_permissions}")
	output, diagnostic, status = run("apply")
	if status != 1 || strings.Contains(diagnostic, "Warning:") || !strings.Contains(diagnostic, "missing/null or invalid") {
		t.Fatal("missing acquisition became a fallback", status, output, diagnostic)
	}
	mu.Lock()
	afterWrites := len(workflowWrites) + len(executionWrites)
	mu.Unlock()
	if afterWrites != beforeWrites {
		t.Fatal("failed preflight wrote a fallback")
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
}
