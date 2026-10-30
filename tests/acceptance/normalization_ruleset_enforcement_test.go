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

func TestRulesetEnforcementNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var state map[string]any
	json.Unmarshal([]byte(acceptanceRuleset), &state)
	state["enforcement"] = "evaluate"
	ignored, includeOther := true, false
	description := "current"
	expectedEnforcement := "disabled"
	generalWrites, reads7 := 0, 0
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
			var base map[string]any
			json.Unmarshal([]byte(planResponse), &base)
			base["description"] = description
			json.NewEncoder(w).Encode(base)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/rulesets":
			if r.Method != "GET" {
				t.Error("unexpected lifecycle write", r.Method)
				w.WriteHeader(400)
				return
			}
			items := []any{map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}}
			if includeOther {
				items = append(items, map[string]any{"id": 9, "source_type": "Repository", "source": "owner/current"})
			}
			json.NewEncoder(w).Encode(items)
		case "/repos/owner/current/rulesets/9":
			if r.Method != "GET" {
				t.Error("unlisted Ruleset write")
				w.WriteHeader(400)
				return
			}
			var other map[string]any
			json.Unmarshal([]byte(acceptanceRuleset), &other)
			other["id"], other["name"], other["enforcement"] = 9, "UNMANAGED_OBJECT", "UNMANAGED_MODE"
			json.NewEncoder(w).Encode(other)
		case "/repos/owner/current/rulesets/7":
			if r.Method == "GET" {
				reads7++
				json.NewEncoder(w).Encode(state)
				return
			}
			if r.Method != "PUT" {
				t.Error("unexpected lifecycle write", r.Method)
				w.WriteHeader(400)
				return
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			want := map[string]any{"name": "basic", "target": "branch", "enforcement": expectedEnforcement, "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": []any{map[string]any{"type": "deletion"}, map[string]any{"type": "non_fast_forward"}}}
			if !reflect.DeepEqual(payload, want) || (payload["enforcement"] != "active" && payload["enforcement"] != "disabled") {
				t.Error("unexpected whole-object payload", payload)
			}
			writes = append(writes, payload)
			if !ignored {
				for k, v := range payload {
					state[k] = v
				}
			}
			// A canonical acknowledgment does not prove that the next GET retained it.
			response := map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}
			for k, v := range payload {
				response[k] = v
			}
			json.NewEncoder(w).Encode(response)
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
	intent := func(status, about string) string {
		t.Helper()
		mu.Lock()
		expectedEnforcement = status
		mu.Unlock()
		input := repositoryInput(about + "settings: {rulesets: [{id: 7, ruleset_name: basic, enforcement_status: " + status + ", target_branches: {include_default_branch: true}, rules: {restrict_deletions: true, block_force_pushes: true}}]}\n")
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
	path := "repository.settings.rulesets.enforcement_status"
	warning := `Warning: repository.settings.rulesets.enforcement_status: observed "evaluate" is outside UI choices; using "disabled" (projection only). Ruleset 7 "basic".`
	output, diagnostic, status := run("export", "--rulesets")
	if status != 0 || output != "" || diagnostic != warning+"\n" {
		t.Fatal("export", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "enforcement_status: disabled") || strings.Contains(string(data), "evaluate") {
		t.Fatal(string(data), err)
	}
	report, summary := filepath.Join(fixture.root, "ruleset.json"), filepath.Join(fixture.root, "ruleset.md")
	var saved struct {
		Outcome  string
		Managed  int
		Rulesets []struct {
			Operation                string
			EnforcementNormalization struct {
				Path     string
				Raw      string `json:"raw_value"`
				Fallback string
			} `json:"enforcement_normalization"`
		}
	}
	for _, desired := range []string{"disabled", "active"} {
		input := intent(desired, "")
		output, diagnostic, status = run("plan")
		decision := path + ` (Ruleset 7 "basic"): "disabled" (normalized observation) → "` + desired + `" (write required)`
		if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) || !strings.Contains(output, "Effective policy:") {
			t.Fatal("normalized decision", status, output, diagnostic)
		}
		output, diagnostic, status = run("action", "plan", report)
		data, err = os.ReadFile(report)
		if err != nil || status != 0 || output != "" || diagnostic != "" || json.Unmarshal(data, &saved) != nil || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Rulesets) != 1 || saved.Rulesets[0].Operation != "update" || saved.Rulesets[0].EnforcementNormalization.Path != path || saved.Rulesets[0].EnforcementNormalization.Raw != "evaluate" || saved.Rulesets[0].EnforcementNormalization.Fallback != "disabled" {
			t.Fatal("private evidence", string(data), diagnostic, err)
		}
		checkSource(input)
	}
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(html.UnescapeString(string(data)), warning) != 1 || !strings.Contains(html.UnescapeString(string(data)), "Effective policy:") {
		t.Fatal("summary", string(data), diagnostic, err)
	}
	private := filepath.Join(fixture.root, "ruleset-private.md")
	output, diagnostic, status = run("action", "summary", report, private, "status-only")
	data, err = os.ReadFile(private)
	if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), path) || strings.Contains(string(data), "evaluate") || strings.Contains(string(data), "Ruleset 7") {
		t.Fatal("private summary", string(data), diagnostic, err)
	}
	mu.Lock()
	readonlyWrites := len(writes) + generalWrites
	mu.Unlock()
	if readonlyWrites != 0 {
		t.Fatal("read-only wrote")
	}
	input := intent("disabled", "")
	mu.Lock()
	beforeReads := reads7
	mu.Unlock()
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "writes may already have applied") {
		t.Fatal("fresh raw mismatch", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	afterReads := reads7
	ignored = false
	mu.Unlock()
	if afterReads-beforeReads != 2 {
		t.Fatal("did not independently read back basic Ruleset", beforeReads, afterReads)
	}
	output, diagnostic, status = run("apply")
	if status != 0 || strings.Count(diagnostic, warning) != 1 {
		t.Fatal("canonical confirmation", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	beforeRepeat := len(writes)
	includeOther = true
	mu.Unlock()
	output, diagnostic, status = run("plan")
	if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
		t.Fatal("unlisted disclosure", status, output, diagnostic)
	}
	output, diagnostic, status = run("apply")
	if status != 0 || diagnostic != "" || strings.Contains(output, "UNMANAGED") {
		t.Fatal("unlisted apply", status, output, diagnostic)
	}
	mu.Lock()
	if len(writes) != beforeRepeat {
		t.Fatal("canonical no-op wrote")
	}
	state["enforcement"] = "evaluate"
	mu.Unlock()
	input = intent("active", "")
	output, diagnostic, status = run("apply")
	if status != 0 || strings.Count(diagnostic, warning) != 1 || strings.Contains(diagnostic, "UNMANAGED") {
		t.Fatal("different choice", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	state["enforcement"] = "evaluate"
	ignored = true
	mu.Unlock()
	input = intent("disabled", "about: {description: updated}\n")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "earlier successful writes remain applied") || strings.Contains(diagnostic, "UNMANAGED") {
		t.Fatal("partial REST progress", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if description != "updated" || generalWrites != 1 {
		t.Fatal(description, generalWrites)
	}
	state["enforcement"] = nil
	beforeInvalid := len(writes)
	mu.Unlock()
	input = intent("disabled", "")
	output, diagnostic, status = run("export", "--rulesets")
	if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
		t.Fatal("null export", status, output, diagnostic)
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
		t.Fatal("malformed observation wrote")
	}
}
