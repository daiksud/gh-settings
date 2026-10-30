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

	"go.yaml.in/yaml/v4"
)

func TestRulesetMergeMethodsNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var state map[string]any
	json.Unmarshal([]byte(acceptanceRuleset), &state)
	reset := func(methods any) {
		rule := acceptanceApprovalsRule(0)
		rule["parameters"].(map[string]any)["allowed_merge_methods"] = methods
		state["rules"] = []any{rule}
	}
	reset([]any{"squash", "FUTURE_METHOD"})
	ignored, retainedRule, includeOther := true, false, false
	desired := []any{"merge", "rebase", "squash"}
	reads, writes, requests := 0, 0, 0
	intended := func() map[string]any {
		rules := []any{}
		if desired != nil {
			rule := acceptanceApprovalsRule(0)
			rule["parameters"].(map[string]any)["allowed_merge_methods"] = desired
			rules = append(rules, rule)
		}
		return map[string]any{"name": "basic", "target": "branch", "enforcement": "active", "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": rules}
	}
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
		case "/repos/owner/current", "/repos/owner/current/topics", "/repos/owner/current/labels":
			if r.Method != "GET" {
				t.Error("unexpected unrelated write", r.Method, r.URL)
				w.WriteHeader(400)
				return
			}
			switch r.URL.Path {
			case "/repos/owner/current":
				fmt.Fprint(w, planResponse)
			case "/repos/owner/current/topics":
				fmt.Fprint(w, `{"names":[]}`)
			default:
				fmt.Fprint(w, `[]`)
			}
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
				t.Error("unlisted write")
				w.WriteHeader(400)
				return
			}
			var other map[string]any
			json.Unmarshal([]byte(acceptanceRuleset), &other)
			other["id"], other["name"] = 9, "UNMANAGED_OBJECT"
			rule := acceptanceApprovalsRule(0)
			rule["parameters"].(map[string]any)["allowed_merge_methods"] = []any{"UNMANAGED_METHOD"}
			other["rules"] = []any{rule}
			json.NewEncoder(w).Encode(other)
		case "/repos/owner/current/rulesets/7":
			if r.Method == "GET" {
				reads++
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
			// Literal expectation is independent of the request and decoded declaration.
			want := intended()
			if !reflect.DeepEqual(payload, want) {
				t.Error("unexpected whole-policy payload", payload, want)
			}
			writes++
			if !ignored {
				for k, v := range payload {
					state[k] = v
				}
			}
			if retainedRule {
				reset([]any{"merge", "rebase", "squash"})
			}
			response := map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}
			for k, v := range want {
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
		rule := "{required_approvals: 0, allowed_merge_methods: [merge, squash, rebase]}"
		mu.Lock()
		desired = []any{"merge", "rebase", "squash"}
		switch choice {
		case "subset":
			rule = "{required_approvals: 0, allowed_merge_methods: [squash]}"
			desired = []any{"squash"}
		case "removed":
			rule = ""
			desired = nil
		}
		mu.Unlock()
		rules := "{}"
		if rule != "" {
			rules = "{require_a_pull_request_before_merging: " + rule + "}"
		}
		input := repositoryInput("settings: {rulesets: [{id: 7, ruleset_name: basic, enforcement_status: active, target_branches: {include_default_branch: true}, rules: " + rules + "}]}\n")
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
	warning := `Warning: repository.settings.rulesets.rules.require_a_pull_request_before_merging.allowed_merge_methods: observed ["squash","FUTURE_METHOD"] is outside UI choices; using ["merge","squash","rebase"] (projection only). Ruleset 7 "basic".`
	output, diagnostic, status := run("export", "--rulesets")
	if status != 0 || output != "" || diagnostic != warning+"\n" {
		t.Fatal("export normalization", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	var exported struct {
		Repository struct {
			Settings struct {
				Rulesets []struct {
					Rules struct {
						PR struct {
							Methods []string `yaml:"allowed_merge_methods"`
						} `yaml:"require_a_pull_request_before_merging"`
					} `yaml:"rules"`
				} `yaml:"rulesets"`
			} `yaml:"settings"`
		} `yaml:"repository"`
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported.Repository.Settings.Rulesets) != 1 || !reflect.DeepEqual(exported.Repository.Settings.Rulesets[0].Rules.PR.Methods, []string{"merge", "rebase", "squash"}) || strings.Contains(string(data), "FUTURE_METHOD") {
		t.Fatal("effective UI export", string(data))
	}
	report, summary := filepath.Join(fixture.root, "methods.json"), filepath.Join(fixture.root, "methods.md")
	for _, choice := range []string{"fallback", "subset", "removed"} {
		input := intent(choice)
		after := `["merge","squash","rebase"] (write required)`
		if choice == "subset" {
			after = `["squash"] (write required)`
		}
		if choice == "removed" {
			after = "inactive (rule removed; write required)"
		}
		output, diagnostic, status = run("plan")
		decision := `repository.settings.rulesets.rules.require_a_pull_request_before_merging.allowed_merge_methods (Ruleset 7 "basic"): ["merge","squash","rebase"] (normalized observation) → ` + after
		if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) || strings.Count(output, "Effective policy:") != 1 {
			t.Fatal("plan evidence", status, output, diagnostic)
		}
		output, diagnostic, status = run("action", "plan", report)
		data, err = os.ReadFile(report)
		var saved struct {
			Outcome  string
			Managed  int
			Rulesets []struct {
				Record struct {
					Path     string
					Raw      []string `json:"raw_value"`
					Fallback []string
				} `json:"merge_methods_normalization"`
			}
		}
		decodeErr := json.Unmarshal(data, &saved)
		if err != nil || decodeErr != nil || status != 0 || output != "" || diagnostic != "" || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Rulesets) != 1 {
			t.Fatal("private report", status, string(data), err, decodeErr, diagnostic)
		}
		record := saved.Rulesets[0].Record
		if record.Path != "repository.settings.rulesets.rules.require_a_pull_request_before_merging.allowed_merge_methods" || !reflect.DeepEqual(record.Raw, []string{"squash", "FUTURE_METHOD"}) || !reflect.DeepEqual(record.Fallback, []string{"merge", "squash", "rebase"}) {
			t.Fatal(record)
		}
		checkSource(input)
	}
	mu.Lock()
	beforeSummary := requests
	mu.Unlock()
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	visible := html.UnescapeString(string(data))
	if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(visible, warning) != 1 || !strings.Contains(visible, "inactive (rule removed; write required)") {
		t.Fatal("detailed Summary", visible, diagnostic, err)
	}
	private := filepath.Join(fixture.root, "private.md")
	output, diagnostic, status = run("action", "summary", report, private, "status-only")
	data, err = os.ReadFile(private)
	if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), "FUTURE_METHOD") || strings.Contains(string(data), "allowed_merge_methods") || strings.Contains(string(data), "Ruleset 7") {
		t.Fatal("private Summary", status, string(data), diagnostic, err)
	}
	mu.Lock()
	if writes != 0 || requests != beforeSummary {
		t.Error("plan/Summary side effects", writes, requests, beforeSummary)
	}
	beforeReads := reads
	mu.Unlock()
	input := intent("fallback")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "writes may already have applied") {
		t.Fatal("raw drift confirmed", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if reads-beforeReads != 2 || writes != 1 {
		t.Error("fresh GET or single PUT lost", reads-beforeReads, writes)
	}
	ignored = false
	mu.Unlock()
	for _, choice := range []string{"fallback", "subset", "removed"} {
		mu.Lock()
		reset([]any{"squash", "FUTURE_METHOD"})
		mu.Unlock()
		input = intent(choice)
		output, diagnostic, status = run("apply")
		if status != 0 || diagnostic != warning+"\n" {
			t.Fatal("confirmed write", choice, status, output, diagnostic)
		}
		checkSource(input)
		mu.Lock()
		beforeWrites := writes
		includeOther = true
		mu.Unlock()
		output, diagnostic, status = run("plan")
		if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
			t.Fatal("canonical repeat", status, output, diagnostic)
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
	reset([]any{"squash", "FUTURE_METHOD"})
	ignored, retainedRule = true, true
	mu.Unlock()
	input = intent("removed")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || !strings.Contains(diagnostic, "writes may already have applied") {
		t.Fatal("retained canonical rule confirmed removal", status, output, diagnostic)
	}
	checkSource(input)
	for _, invalid := range []any{nil, []any{"future", nil}, []any{}, []any{"future", "future"}} {
		mu.Lock()
		reset(invalid)
		beforeWrites := writes
		mu.Unlock()
		input = intent("fallback")
		for _, args := range [][]string{{"export", "--rulesets"}, {"apply"}} {
			output, diagnostic, status = run(args...)
			if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
				t.Fatal("malformed acquisition", args, status, output, diagnostic)
			}
			checkSource(input)
		}
		mu.Lock()
		changed := writes != beforeWrites
		mu.Unlock()
		if changed {
			t.Fatal("incomplete acquisition wrote")
		}
	}
}
