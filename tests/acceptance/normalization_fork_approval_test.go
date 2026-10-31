//go:build acceptance

package acceptance

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestForkApprovalNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var permission any = "FUTURE"
	keepUnknown := true
	execution := map[string]any{"enabled": false, "allowed_actions": "all", "sha_pinning_required": false}
	var workflowWrites, executionWrites []map[string]any
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/repos/owner/current/actions/permissions/fork-pr-contributor-approval" {
			switch req.Method {
			case "GET":
				json.NewEncoder(w).Encode(map[string]any{"approval_policy": permission})
			case "PUT":
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				workflowWrites = append(workflowWrites, payload)
				if value, present := payload["approval_policy"]; present && !keepUnknown {
					permission = value
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
			fmt.Fprint(w, strings.TrimSuffix(planResponse, "}")+`,"visibility":"public"}`)
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
	env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
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
	path := "repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors"
	fallback := "require_approval_for_first_time_contributors"
	warning := "Warning: " + path + ": observed \"FUTURE\" is outside UI choices; using \"" + fallback + "\" (projection only)."
	output, diagnostic, status := run("export", "--fork-approval")
	if diagnostic != warning+"\n" {
		t.Fatalf("exact export warning lost: %q", diagnostic)
	}
	if status != 0 || output != "" || !strings.Contains(diagnostic, "Warning:") || !strings.Contains(diagnostic, "FUTURE") || !strings.Contains(diagnostic, "repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors") || !strings.Contains(diagnostic, "require_approval_for_first_time_contributors") {
		t.Fatal("export normalization was not visible", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "approval_for_running_fork_pull_request_workflows_from_contributors: require_approval_for_first_time_contributors") || strings.Contains(string(data), "FUTURE") {
		t.Fatal("exported YAML did not contain only UI fallback", string(data), err)
	}
	for _, choice := range []string{"require_approval_for_first_time_contributors", "require_approval_for_all_external_contributors", "require_approval_for_first_time_contributors_who_are_new_to_github"} {
		input := intent("{approval_for_running_fork_pull_request_workflows_from_contributors: " + choice + "}")
		decision := path + ": " + strconv.Quote(fallback) + " (normalized observation) → " + strconv.Quote(choice) + " (write required)"
		output, diagnostic, status = run("plan")
		if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) || !strings.Contains(output, "FUTURE") || !strings.Contains(output, choice) || !strings.Contains(output, "write required") || strings.Contains(output, "No differences") {
			t.Fatal("plan confused a projection with observation", choice, status, output, diagnostic)
		}
		if err := checkSettingsUnchanged(fixture.work, input); err != nil {
			t.Fatal(err)
		}
	}
	report, summary := filepath.Join(fixture.root, "fork-normalization.json"), filepath.Join(fixture.root, "private-summary.md")

	output, diagnostic, status = run("action", "plan", report)
	if status != 0 || output != "" || diagnostic != "" {
		t.Fatal(status, output, diagnostic)
	}
	data, err = os.ReadFile(report)
	var saved struct {
		Outcome string
		Managed int
		Actions []struct {
			Field, Before, After string
			Normalization        struct {
				Path     string
				Raw      string `json:"raw_value"`
				Fallback string
			}
		}
	}
	if err != nil || json.Unmarshal(data, &saved) != nil || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Actions) != 1 || saved.Actions[0].Field != "actions.fork_pr_approval_policy" || saved.Actions[0].Before != "first_time_contributors" || saved.Actions[0].After != "first_time_contributors_new_to_github" || saved.Actions[0].Normalization.Path != path || saved.Actions[0].Normalization.Raw != "FUTURE" || saved.Actions[0].Normalization.Fallback != fallback {
		t.Fatal("private fork evidence lost", string(data), err)
	}
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	detailed := html.UnescapeString(string(data))
	if err != nil || status != 0 || output != "" || diagnostic != "" || !strings.Contains(detailed, warning) || !strings.Contains(detailed, path+": "+strconv.Quote(fallback)+" (normalized observation) → \"require_approval_for_first_time_contributors_who_are_new_to_github\" (write required)") {
		t.Fatal("detailed fork Summary lost decision", status, output, diagnostic, string(data), err)
	}
	summary = filepath.Join(fixture.root, "fork-status-summary.md")
	output, diagnostic, status = run("action", "plan", report, "status-only")
	if status != 0 || output != "" || diagnostic != "" {
		t.Fatal("private planning disclosed a diagnostic", status, output, diagnostic)
	}
	output, diagnostic, status = run("action", "summary", report, summary, "status-only")
	data, err = os.ReadFile(summary)
	if err != nil || status != 0 || diagnostic != "" || output != "plan-outcome=changed\n" || strings.Contains(string(data), "FUTURE") || strings.Contains(string(data), "approval_for_running_fork_pull_request_workflows_from_contributors") {
		t.Fatal("private summary disclosed normalized policy", status, output, diagnostic, string(data), err)
	}
	mu.Lock()
	readOnlyWrites := len(workflowWrites) + len(executionWrites)
	mu.Unlock()
	if readOnlyWrites != 0 {
		t.Fatal("export/plan normalized remote state", readOnlyWrites)
	}
	intent("{actions_permissions: {policy: disable_actions}}")
	output, diagnostic, status = run("plan")
	if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
		t.Fatal("unmanaged fork was printed", status, output, diagnostic)
	}
	output, diagnostic, status = run("apply")
	if status != 0 || diagnostic != "" {
		t.Fatal("unmanaged fork was replayed", status, output, diagnostic)
	}
	mu.Lock()
	preserved := permission == "FUTURE" && len(workflowWrites) == 0
	mu.Unlock()
	if !preserved {
		t.Fatal("unmanaged fork mutated")
	}
	input := intent("{approval_for_running_fork_pull_request_workflows_from_contributors: require_approval_for_first_time_contributors}")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, "Warning:") != 3 || !strings.Contains(diagnostic, "may already have applied") {
		t.Fatal("normalized fallback falsely verified a write", status, output, diagnostic)
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	keepUnknown = false
	mu.Unlock()
	intent("{approval_for_running_fork_pull_request_workflows_from_contributors: require_approval_for_all_external_contributors}")
	output, diagnostic, status = run("apply")
	if status != 0 || strings.Count(diagnostic, "Warning:") != 2 || !strings.Contains(output, "Applied repository settings") {
		t.Fatal("canonical readback did not complete apply", status, output, diagnostic)
	}
	output, diagnostic, status = run("plan")
	if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
		t.Fatal("exact acquired choice retained a fallback warning", status, output, diagnostic)
	}
	mu.Lock()
	permission, keepUnknown = "FUTURE", true
	beforeExecutionWrites := len(executionWrites)
	mu.Unlock()
	input = intent("{actions_permissions: {policy: allow_all_actions_and_reusable_workflows}, approval_for_running_fork_pull_request_workflows_from_contributors: require_approval_for_first_time_contributors}")
	output, diagnostic, status = run("apply")
	if status != 1 || !strings.Contains(diagnostic, "earlier successful writes remain applied") || !strings.Contains(diagnostic, "FUTURE") {
		t.Fatal("unverified readback lost earlier progress", status, output, diagnostic)
	}
	mu.Lock()
	retained := execution["enabled"] == true && len(executionWrites) == beforeExecutionWrites+1 && reflect.DeepEqual(executionWrites[len(executionWrites)-1], map[string]any{"enabled": true, "allowed_actions": "all"}) && permission == "FUTURE" && reflect.DeepEqual(workflowWrites[len(workflowWrites)-1], map[string]any{"approval_policy": "first_time_contributors"})
	mu.Unlock()
	if !retained {
		t.Fatal("partial failure implied rollback or hid requested write", execution, executionWrites, workflowWrites, permission)
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	permission = nil
	beforeWrites := len(workflowWrites) + len(executionWrites)
	mu.Unlock()
	input = intent("{approval_for_running_fork_pull_request_workflows_from_contributors: require_approval_for_first_time_contributors}")
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
