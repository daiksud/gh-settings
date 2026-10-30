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

func TestRulesetBypassNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var state map[string]any
	json.Unmarshal([]byte(acceptanceRuleset), &state)
	reset := func(mode any) {
		state["bypass_actors"] = []any{
			map[string]any{"actor_type": "Team", "actor_id": 9, "bypass_mode": mode},
			map[string]any{"actor_type": "User", "actor_id": 9, "bypass_mode": "exempt"},
			map[string]any{"actor_type": "DeployKey", "actor_id": nil, "bypass_mode": "KEY_MODE"},
		}
	}
	reset("FUTURE_MODE")
	ignored, retainedCanonical, includeOther := true, false, false
	expectedMode := "always"
	reads, writes, requests := 0, 0, 0
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests++
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
			if r.Method != "GET" {
				t.Error("unexpected repository write", r.Method)
				w.WriteHeader(400)
				return
			}
			fmt.Fprint(w, planResponse)
		case "/repos/owner/current/topics":
			if r.Method != "GET" {
				t.Error("unexpected topics write", r.Method)
				w.WriteHeader(400)
				return
			}
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			if r.Method != "GET" {
				t.Error("unexpected labels write", r.Method)
				w.WriteHeader(400)
				return
			}
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/rulesets":
			if r.Method != "GET" {
				t.Error("lifecycle write", r.Method)
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
				t.Error("unlisted write")
				w.WriteHeader(400)
				return
			}
			var other map[string]any
			json.Unmarshal([]byte(acceptanceRuleset), &other)
			other["id"], other["name"] = 9, "UNMANAGED_OBJECT"
			other["bypass_actors"] = []any{map[string]any{"actor_type": "Team", "actor_id": 19, "bypass_mode": "UNMANAGED_MODE"}}
			json.NewEncoder(w).Encode(other)
		case "/repos/owner/current/rulesets/7":
			if r.Method == "GET" {
				reads++
				json.NewEncoder(w).Encode(state)
				return
			}
			if r.Method != "PUT" {
				t.Error("lifecycle write", r.Method)
				w.WriteHeader(400)
				return
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			actors := []any{map[string]any{"actor_type": "DeployKey", "actor_id": nil, "bypass_mode": "always"}}
			if expectedMode != "removed" {
				actors = append(actors, map[string]any{"actor_type": "Team", "actor_id": float64(9), "bypass_mode": expectedMode})
			}
			actors = append(actors, map[string]any{"actor_type": "User", "actor_id": float64(9), "bypass_mode": "exempt"})
			want := map[string]any{"name": "basic", "target": "branch", "enforcement": "active", "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": actors, "rules": []any{map[string]any{"type": "deletion"}, map[string]any{"type": "non_fast_forward"}}}
			if !reflect.DeepEqual(payload, want) {
				t.Error("literal whole-policy payload differs", payload, want)
			}
			writes++
			if !ignored {
				for k, v := range payload {
					state[k] = v
				}
			}
			if retainedCanonical {
				for _, raw := range state["bypass_actors"].([]any) {
					actor := raw.(map[string]any)
					if actor["actor_type"] != "User" {
						actor["bypass_mode"] = "always"
					}
				}
			}
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
	intent := func(choice string) string {
		t.Helper()
		native, valid := map[string]string{"always": "always", "for_pull_requests_only": "pull_request", "removed": "removed"}[choice]
		if !valid {
			t.Fatal(choice)
		}
		mu.Lock()
		expectedMode = native
		mu.Unlock()
		actors := "[{deploy_keys: true, bypass_mode: always}, "
		if choice != "removed" {
			actors += "{team: 9, bypass_mode: " + choice + "}, "
		}
		actors += "{user: 9, bypass_mode: exempt}]"
		input := repositoryInput("settings: {rulesets: [{id: 7, ruleset_name: basic, enforcement_status: active, target_branches: {include_default_branch: true}, bypass_list: " + actors + ", rules: {restrict_deletions: true, block_force_pushes: true}}]}\n")
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
	keyWarning := `Warning: repository.settings.rulesets.bypass_list.bypass_mode: observed "KEY_MODE" is outside UI choices; using "always" (projection only). Ruleset 7 "basic", actor_type="DeployKey".`
	teamWarning := `Warning: repository.settings.rulesets.bypass_list.bypass_mode: observed "FUTURE_MODE" is outside UI choices; using "always" (projection only). Ruleset 7 "basic", actor_type="Team", actor_id=9.`
	warnings := keyWarning + "\n" + teamWarning + "\n"
	output, diagnostic, status := run("export", "--rulesets")
	if status != 0 || output != "" || diagnostic != warnings {
		t.Fatal("export normalization", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || strings.Count(string(data), "bypass_mode: always") != 2 || !strings.Contains(string(data), "bypass_mode: exempt") || strings.Contains(string(data), "FUTURE_MODE") || strings.Contains(string(data), "KEY_MODE") {
		t.Fatal("export UI", string(data), err)
	}
	report, summary := filepath.Join(fixture.root, "bypass.json"), filepath.Join(fixture.root, "bypass.md")
	for _, choice := range []string{"always", "for_pull_requests_only", "removed"} {
		input := intent(choice)
		output, diagnostic, status = run("plan")
		after := `"` + choice + `" (write required)`
		if choice == "removed" {
			after = "inactive (actor removed; write required)"
		}
		decision := `repository.settings.rulesets.bypass_list.bypass_mode (Ruleset 7 "basic", actor_type="Team", actor_id=9): "always" (normalized observation) → ` + after
		if status != 0 || diagnostic != "" || strings.Count(output, teamWarning) != 1 || strings.Count(output, keyWarning) != 1 || !strings.Contains(output, decision) || strings.Count(output, "Effective policy:") != 1 {
			t.Fatal("plan evidence", status, output, diagnostic)
		}
		output, diagnostic, status = run("action", "plan", report)
		data, err = os.ReadFile(report)
		var saved struct {
			Outcome  string
			Managed  int
			Rulesets []struct {
				Operation string
				Records   []struct {
					ActorType     string `json:"actor_type"`
					ActorID       *int64 `json:"actor_id"`
					Normalization struct {
						Path     string
						Raw      string `json:"raw_value"`
						Fallback string
					}
				} `json:"bypass_mode_normalizations"`
			}
		}
		decodeErr := json.Unmarshal(data, &saved)
		if err != nil || decodeErr != nil || status != 0 || output != "" || diagnostic != "" || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Rulesets) != 1 || len(saved.Rulesets[0].Records) != 2 {
			t.Fatal("private evidence", string(data), err, decodeErr, diagnostic)
		}
		for _, record := range saved.Rulesets[0].Records {
			if record.Normalization.Path != "repository.settings.rulesets.bypass_list.bypass_mode" || record.Normalization.Fallback != "always" {
				t.Fatal(record)
			}
			switch record.ActorType {
			case "Team":
				if record.ActorID == nil || *record.ActorID != 9 || record.Normalization.Raw != "FUTURE_MODE" {
					t.Fatal(record)
				}
			case "DeployKey":
				if record.ActorID != nil || record.Normalization.Raw != "KEY_MODE" {
					t.Fatal(record)
				}
			default:
				t.Fatal(record)
			}
		}
		checkSource(input)
	}
	mu.Lock()
	beforeSummaryRequests := requests
	mu.Unlock()
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	visible := html.UnescapeString(string(data))
	if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(visible, teamWarning) != 1 || !strings.Contains(visible, "inactive (actor removed; write required)") {
		t.Fatal("detailed Summary", visible, diagnostic, err)
	}
	private := filepath.Join(fixture.root, "private.md")
	output, diagnostic, status = run("action", "summary", report, private, "status-only")
	data, err = os.ReadFile(private)
	if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), "actor_type") || strings.Contains(string(data), "FUTURE_MODE") || strings.Contains(string(data), "Ruleset 7") {
		t.Fatal("private Summary", string(data), diagnostic, err)
	}
	mu.Lock()
	if writes != 0 || requests != beforeSummaryRequests {
		t.Error("read-only plan or Summary wrote/requested", writes, requests, beforeSummaryRequests)
	}
	beforeReads := reads
	mu.Unlock()
	input := intent("always")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, teamWarning) != 2 || strings.Count(diagnostic, keyWarning) != 2 || !strings.Contains(diagnostic, "writes may already have applied") {
		t.Fatal("raw GET drift hidden", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if reads-beforeReads != 2 || writes != 1 {
		t.Error("fresh GET or single PUT lost", reads-beforeReads, writes)
	}
	ignored = false
	mu.Unlock()
	for _, choice := range []string{"always", "for_pull_requests_only", "removed"} {
		mu.Lock()
		reset("FUTURE_MODE")
		mu.Unlock()
		input = intent(choice)
		output, diagnostic, status = run("apply")
		if status != 0 || diagnostic != warnings {
			t.Fatal("confirmed update/removal", choice, status, output, diagnostic)
		}
		checkSource(input)
		mu.Lock()
		beforeWrites := writes
		includeOther = true
		mu.Unlock()
		output, diagnostic, status = run("plan")
		if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
			t.Fatal("canonical no-op", status, output, diagnostic)
		}
		output, diagnostic, status = run("apply")
		if status != 0 || diagnostic != "" || strings.Contains(output, "UNMANAGED") {
			t.Fatal("unlisted privacy", status, output, diagnostic)
		}
		mu.Lock()
		unchanged := writes == beforeWrites
		mu.Unlock()
		if !unchanged {
			t.Fatal("canonical no-op wrote")
		}
	}
	mu.Lock()
	reset("FUTURE_MODE")
	ignored, retainedCanonical = true, true
	mu.Unlock()
	input = intent("removed")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || !strings.Contains(diagnostic, "writes may already have applied") {
		t.Fatal("retained canonical actor confirmed removal", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	reset(nil)
	beforeInvalid := writes
	mu.Unlock()
	input = intent("always")
	for _, args := range [][]string{{"export", "--rulesets"}, {"apply"}} {
		output, diagnostic, status = run(args...)
		if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
			t.Fatal("malformed acquisition", args, status, output, diagnostic)
		}
		checkSource(input)
	}
	mu.Lock()
	malformedWrote := writes != beforeInvalid
	mu.Unlock()
	if malformedWrote {
		t.Fatal("malformed acquisition wrote")
	}
}
