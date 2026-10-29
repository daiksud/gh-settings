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
	"strings"
	"sync"
	"testing"
)

func TestActionsExecutionNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	base := map[string]any{"enabled": true, "allowed_actions": "FUTURE_POLICY", "sha_pinning_required": true}
	selected := map[string]any{"github_owned_allowed": false, "verified_allowed": false, "patterns_allowed": []any{}}
	ignored := true
	description := "current"
	generalWrites := 0
	var writes []map[string]any
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			if r.Method == "PATCH" {
				var payload map[string]any
				json.NewDecoder(r.Body).Decode(&payload)
				if !reflect.DeepEqual(payload, map[string]any{"description": "updated"}) {
					t.Error(payload)
				}
				description = "updated"
				generalWrites++
				fmt.Fprint(w, `{}`)
				return
			}
			var state map[string]any
			json.Unmarshal([]byte(planResponse), &state)
			state["description"] = description
			json.NewEncoder(w).Encode(state)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/actions/permissions", "/repos/owner/current/actions/permissions/selected-actions":
			target := base
			isSelected := strings.HasSuffix(r.URL.Path, "/selected-actions")
			if isSelected {
				if base["allowed_actions"] != "selected" {
					t.Error("inactive selected request")
					w.WriteHeader(409)
					return
				}
				target = selected
			}
			if r.Method == "GET" {
				json.NewEncoder(w).Encode(target)
				return
			}
			if r.Method != "PUT" {
				t.Error(r.Method)
				w.WriteHeader(400)
				return
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if isSelected && base["enabled"] != false {
				t.Error("selected configured after enabling")
			}
			writes = append(writes, payload)
			if !ignored {
				for key, value := range payload {
					target[key] = value
				}
			}
			w.WriteHeader(204)
		default:
			t.Error("unexpected request", r.Method, r.URL)
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
		r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil {
			t.Fatal(err)
		}
		status := 0
		if r.runErr != nil {
			var failure *exec.ExitError
			if !errors.As(r.runErr, &failure) {
				t.Fatal(r.runErr)
			}
			status = failure.ExitCode()
		}
		return r.stdout.String(), r.stderr.String(), status
	}
	intent := func(general, about string) string {
		t.Helper()
		input := repositoryInput(about + "settings: {actions: {general: " + general + "}}\n")
		if err := writeSettingsFile(fixture.work, input); err != nil {
			t.Fatal(err)
		}
		return input
	}
	checkSource := func(input string) {
		t.Helper()
		if err := checkSettingsUnchanged(fixture.work, input); err != nil {
			t.Fatal(err)
		}
	}
	path := "repository.settings.actions.general.actions_permissions"
	fallback := "allow_all_actions_and_reusable_workflows"
	warning := "Warning: " + path + ": observed \"FUTURE_POLICY\" is outside UI choices; using \"" + fallback + "\" (projection only)."
	output, diagnostic, status := run("export", "--actions")
	if status != 0 || output != "" || diagnostic != warning+"\n" {
		t.Fatal("export", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "actions_permissions: "+fallback) || strings.Contains(string(data), "FUTURE_POLICY") {
		t.Fatal(string(data), err)
	}
	report, summary := filepath.Join(fixture.root, "execution.json"), filepath.Join(fixture.root, "execution.md")
	var saved struct {
		Outcome          string
		Managed          int
		ActionsExecution []struct {
			Before, After string
			Normalization struct {
				Path     string
				Raw      string `json:"raw_value"`
				Fallback string
			}
		} `json:"actions_execution"`
	}
	for _, choice := range []string{fallback, "disable_actions", "allow_owner_actions_and_reusable_workflows"} {
		input := intent("{actions_permissions: "+choice+"}", "")
		output, diagnostic, status = run("plan")
		decision := path + ": \"" + fallback + "\" (normalized observation) → \"" + choice + "\" (write required)"
		if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) {
			t.Fatal("radio decision", status, output, diagnostic)
		}
		output, diagnostic, status = run("action", "plan", report)
		data, err = os.ReadFile(report)
		wantManaged := 2
		if choice == "disable_actions" {
			wantManaged = 1
		}
		if err != nil || status != 0 || output != "" || diagnostic != "" || json.Unmarshal(data, &saved) != nil || saved.Outcome != "changed" || saved.Managed != wantManaged || len(saved.ActionsExecution) != 1 || saved.ActionsExecution[0].Normalization.Raw != "FUTURE_POLICY" || saved.ActionsExecution[0].Normalization.Path != path || saved.ActionsExecution[0].Before != fallback || saved.ActionsExecution[0].After != choice {
			t.Fatal("private radio evidence", string(data), err, diagnostic)
		}
		checkSource(input)
	}
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(html.UnescapeString(string(data)), warning) != 1 {
		t.Fatal("summary", string(data), diagnostic, err)
	}
	privateSummary := filepath.Join(fixture.root, "execution-private.md")
	output, diagnostic, status = run("action", "summary", report, privateSummary, "status-only")
	data, err = os.ReadFile(privateSummary)
	if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), path) || strings.Contains(string(data), "FUTURE_POLICY") {
		t.Fatal("private summary", string(data), diagnostic, err)
	}
	mu.Lock()
	readonlyWrites := len(writes) + generalWrites
	mu.Unlock()
	if readonlyWrites != 0 {
		t.Fatal("read-only wrote")
	}
	input := intent("{actions_permissions: "+fallback+"}", "")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "writes may already have applied") {
		t.Fatal("retained unknown", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	ignored = false
	mu.Unlock()
	output, diagnostic, status = run("apply")
	if status != 0 || strings.Count(diagnostic, warning) != 1 {
		t.Fatal("canonical confirmation", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	base["allowed_actions"] = "FUTURE_POLICY"
	mu.Unlock()
	input = intent("{actions_permissions: disable_actions}", "")
	output, diagnostic, status = run("apply")
	if status != 0 || strings.Count(diagnostic, warning) != 1 {
		t.Fatal("disable preserve", status, output, diagnostic)
	}
	checkSource(input)
	output, diagnostic, status = run("export", "--actions")
	data, err = os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || status != 0 || diagnostic != "" || !strings.Contains(string(data), "actions_permissions: disable_actions") || strings.Contains(string(data), "FUTURE_POLICY") {
		t.Fatal("hidden export", status, string(data), diagnostic, err)
	}
	input = intent("{actions_permissions: disable_actions}", "")
	output, diagnostic, status = run("plan")
	if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
		t.Fatal("known disabled", status, output, diagnostic)
	}
	input = intent("{actions_permissions: allow_owner_and_select_non_owner_actions_and_reusable_workflows, allow_actions_created_by_github: true, allow_actions_by_marketplace_verified_creators: false, allow_or_block_specified_actions_and_reusable_workflows: ['org/*', '!org/bad@*', 'org/*']}", "")
	output, diagnostic, status = run("plan")
	if status != 0 || diagnostic != "" || strings.Contains(output, "FUTURE_POLICY") || strings.Contains(output, "normalized observation") || !strings.Contains(output, `"disable_actions" → "allow_owner_and_select_non_owner_actions_and_reusable_workflows" (write required)`) {
		t.Fatal("known disabled enabling", status, output, diagnostic)
	}
	output, diagnostic, status = run("action", "plan", report)
	data, err = os.ReadFile(report)
	if err != nil || status != 0 || diagnostic != "" || json.Unmarshal(data, &saved) != nil || saved.Managed != 5 || len(saved.ActionsExecution) != 1 || saved.ActionsExecution[0].Before != "disable_actions" {
		t.Fatal("selected ownership", string(data), diagnostic, err)
	}
	mu.Lock()
	beforeSelection := len(writes)
	mu.Unlock()
	output, diagnostic, status = run("apply")
	if status != 0 || diagnostic != "" {
		t.Fatal("hidden selected enable", status, output, diagnostic)
	}
	mu.Lock()
	selectionWrites := append([]map[string]any{}, writes[beforeSelection:]...)
	base["allowed_actions"] = "FUTURE_POLICY"
	base["enabled"] = true
	mu.Unlock()
	wantSelection := []map[string]any{{"enabled": false, "allowed_actions": "selected"}, {"github_owned_allowed": true, "verified_allowed": false, "patterns_allowed": []any{"org/*", "!org/bad@*", "org/*"}}, {"enabled": true}}
	if !reflect.DeepEqual(selectionWrites, wantSelection) {
		t.Fatal(selectionWrites, wantSelection)
	}
	checkSource(input)
	input = intent("{require_actions_to_be_pinned_to_a_full_length_commit_sha: false}", "")
	output, diagnostic, status = run("apply")
	if status != 0 || diagnostic != "" || strings.Contains(output, "FUTURE_POLICY") {
		t.Fatal("silent SHA update", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if base["allowed_actions"] != "FUTURE_POLICY" {
		t.Fatal("raw policy replayed")
	}
	ignored = true
	mu.Unlock()
	input = intent("{actions_permissions: "+fallback+"}", "about: {description: updated}\n")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "earlier resource writes remain applied") {
		t.Fatal("earlier REST effects", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if description != "updated" || generalWrites != 1 {
		t.Fatal(description, generalWrites)
	}
	base["allowed_actions"] = nil
	beforeInvalid := len(writes)
	mu.Unlock()
	output, diagnostic, status = run("export", "--actions")
	if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
		t.Fatal("null acquisition", status, output, diagnostic)
	}
	checkSource(input)
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
		t.Fatal("null preflight", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	defer mu.Unlock()
	if len(writes) != beforeInvalid {
		t.Fatal("invalid observation wrote")
	}
}
