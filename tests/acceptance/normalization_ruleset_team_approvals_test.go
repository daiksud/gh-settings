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

func acceptanceTeamReviewer(id int, count any, patterns []any) map[string]any {
	return map[string]any{"reviewer": map[string]any{"type": "Team", "id": float64(id)}, "minimum_approvals": count, "file_patterns": patterns}
}

func acceptanceTeamApprovalsIntent(mode string) map[string]any {
	rules := []any{}
	if mode != "PR removed" {
		teams := []any{acceptanceTeamReviewer(2, float64(0), []any{"docs/**"}), acceptanceTeamReviewer(5, float64(3), []any{})}
		if mode != "reviewer removed" {
			id, count := 10, float64(0)
			if mode == "different" {
				count = 10
			}
			if mode == "replacement" {
				id, count = 99, 4
			}
			teams = append(teams, acceptanceTeamReviewer(id, count, []any{"**", "!generated/**", "**"}))
		}
		rule := acceptanceApprovalsRule(0)
		rule["parameters"].(map[string]any)["required_reviewers"] = teams
		rules = append(rules, rule)
	}
	return map[string]any{"name": "basic", "target": "branch", "enforcement": "active", "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": rules}
}

func TestRulesetTeamApprovalsNormalizationAcrossExecutableCommands(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var state map[string]any
	json.Unmarshal([]byte(acceptanceRuleset), &state)
	reset := func(count any) {
		rule := acceptanceApprovalsRule(0)
		rule["parameters"].(map[string]any)["required_reviewers"] = []any{acceptanceTeamReviewer(10, count, []any{"**", "!generated/**", "**"}), acceptanceTeamReviewer(2, -1, []any{"docs/**"}), acceptanceTeamReviewer(5, 3, []any{})}
		state["rules"] = []any{rule}
	}
	reset(11)
	desired := "equal"
	ignored, retained, includeOther := true, false, false
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
				t.Error("lifecycle write")
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
				t.Error("unselected write")
				w.WriteHeader(400)
				return
			}
			other := acceptanceTeamApprovalsIntent("equal")
			other["id"], other["name"], other["source_type"], other["source"] = 9, "UNMANAGED_TEAM_POLICY", "Repository", "owner/current"
			other["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["required_reviewers"] = []any{acceptanceTeamReviewer(999, 111, []any{})}
			json.NewEncoder(w).Encode(other)
		case "/repos/owner/current/rulesets/7":
			if r.Method == "GET" {
				reads++
				json.NewEncoder(w).Encode(state)
				return
			}
			if r.Method != "PUT" {
				t.Error("lifecycle write")
				w.WriteHeader(400)
				return
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			want := acceptanceTeamApprovalsIntent(desired)
			if !reflect.DeepEqual(payload, want) {
				t.Error("literal whole-policy request differs", payload, want)
			}
			writes++
			if !ignored {
				for k, v := range payload {
					state[k] = v
				}
			}
			if retained {
				state["rules"] = acceptanceTeamApprovalsIntent("equal")["rules"]
			}
			want["id"], want["source_type"], want["source"] = 7, "Repository", "owner/current"
			json.NewEncoder(w).Encode(want)
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
	intent := func(mode string) string {
		t.Helper()
		mu.Lock()
		desired = mode
		mu.Unlock()
		teams := "[{reviewer: 2, approvals: 0, file_patterns: ['docs/**']}, {reviewer: 5, approvals: 3, file_patterns: []}"
		if mode != "reviewer removed" {
			id, count := 10, 0
			if mode == "different" {
				count = 10
			}
			if mode == "replacement" {
				id, count = 99, 4
			}
			teams += fmt.Sprintf(", {reviewer: %d, approvals: %d, file_patterns: ['**', '!generated/**', '**']}", id, count)
		}
		teams += "]"
		rules := "{require_a_pull_request_before_merging: {require_review_from_specific_teams: " + teams + "}}"
		if mode == "PR removed" {
			rules = "{}"
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
	path := "repository.settings.rulesets.rules.require_a_pull_request_before_merging.require_review_from_specific_teams.approvals"
	warning2 := `Warning: ` + path + `: observed -1 is outside UI choices; using 0 (projection only). Ruleset 7 "basic", reviewer=2.`
	warning10 := `Warning: ` + path + `: observed 11 is outside UI choices; using 0 (projection only). Ruleset 7 "basic", reviewer=10.`
	warnings := warning2 + "\n" + warning10 + "\n"
	output, diagnostic, status := run("export", "--rulesets")
	if status != 0 || output != "" || diagnostic != warnings {
		t.Fatal("export normalization", status, output, diagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var exported struct {
		Repository struct {
			Settings struct {
				Rulesets []struct {
					Rules struct {
						PR struct {
							Teams []struct {
								Reviewer  int
								Approvals *int
								Patterns  *[]string `yaml:"file_patterns"`
							} `yaml:"require_review_from_specific_teams"`
						} `yaml:"require_a_pull_request_before_merging"`
					} `yaml:"rules"`
				} `yaml:"rulesets"`
			} `yaml:"settings"`
		} `yaml:"repository"`
	}
	if err := yaml.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported.Repository.Settings.Rulesets) != 1 {
		t.Fatal(string(data))
	}
	teams := exported.Repository.Settings.Rulesets[0].Rules.PR.Teams
	if len(teams) != 3 || teams[0].Reviewer != 2 || teams[1].Reviewer != 5 || teams[2].Reviewer != 10 || teams[0].Approvals == nil || *teams[0].Approvals != 0 || teams[1].Approvals == nil || *teams[1].Approvals != 3 || teams[2].Approvals == nil || *teams[2].Approvals != 0 || teams[1].Patterns == nil || len(*teams[1].Patterns) != 0 || teams[2].Patterns == nil || !reflect.DeepEqual(*teams[2].Patterns, []string{"**", "!generated/**", "**"}) {
		t.Fatal("effective UI or patterns differ", string(data))
	}
	report, summary := filepath.Join(fixture.root, "teams.json"), filepath.Join(fixture.root, "teams.md")
	for _, mode := range []string{"equal", "different", "reviewer removed", "replacement", "PR removed"} {
		input := intent(mode)
		output, diagnostic, status = run("plan")
		after := "0 (write required)"
		switch mode {
		case "different":
			after = "10 (write required)"
		case "reviewer removed", "replacement":
			after = "inactive (reviewer removed; write required)"
		case "PR removed":
			after = "inactive (rule removed; write required)"
		}
		decision := path + ` (Ruleset 7 "basic", reviewer=10): 0 (normalized observation) → ` + after
		if status != 0 || diagnostic != "" || strings.Count(output, warning2) != 1 || strings.Count(output, warning10) != 1 || !strings.Contains(output, decision) || strings.Count(output, "Effective policy:") != 1 {
			t.Fatal("plan evidence", mode, status, output, diagnostic)
		}
		output, diagnostic, status = run("action", "plan", report)
		data, err = os.ReadFile(report)
		var saved struct {
			Outcome  string
			Managed  int
			Rulesets []struct {
				Records []struct {
					ReviewerID *int `json:"reviewer_id"`
					Path       string
					Raw        *int `json:"raw_value"`
					Fallback   *int
				} `json:"team_approvals_normalizations"`
			}
		}
		decodeErr := json.Unmarshal(data, &saved)
		if err != nil || decodeErr != nil || status != 0 || output != "" || diagnostic != "" || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Rulesets) != 1 || len(saved.Rulesets[0].Records) != 2 {
			t.Fatal("private report", status, string(data), err, decodeErr, diagnostic)
		}
		for _, n := range saved.Rulesets[0].Records {
			if n.ReviewerID == nil || n.Raw == nil || n.Fallback == nil || *n.Fallback != 0 || n.Path != path || (*n.ReviewerID == 10 && *n.Raw != 11) || (*n.ReviewerID == 2 && *n.Raw != -1) || (*n.ReviewerID != 2 && *n.ReviewerID != 10) {
				t.Fatal(n)
			}
		}
		checkSource(input)
	}
	mu.Lock()
	beforeSummary := requests
	mu.Unlock()
	output, diagnostic, status = run("action", "summary", report, summary)
	data, err = os.ReadFile(summary)
	visible := html.UnescapeString(string(data))
	if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(visible, warning2) != 1 || strings.Count(visible, warning10) != 1 || !strings.Contains(visible, "inactive (rule removed; write required)") {
		t.Fatal("detailed Summary", status, visible, diagnostic, err)
	}
	private := filepath.Join(fixture.root, "private.md")
	output, diagnostic, status = run("action", "summary", report, private, "status-only")
	data, err = os.ReadFile(private)
	if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), "reviewer=") || strings.Contains(string(data), path) || strings.Contains(string(data), "Ruleset 7") {
		t.Fatal("status-only privacy", status, string(data), diagnostic, err)
	}
	mu.Lock()
	if writes != 0 || requests != beforeSummary {
		t.Error("plan/Summary side effects", writes, requests, beforeSummary)
	}
	beforeReads := reads
	mu.Unlock()
	input := intent("equal")
	output, diagnostic, status = run("apply")
	if status != 1 || output != "" || strings.Count(diagnostic, warning2) != 2 || strings.Count(diagnostic, warning10) != 2 || !strings.Contains(diagnostic, "writes may already have applied") {
		t.Fatal("raw counts confirmed default", status, output, diagnostic)
	}
	checkSource(input)
	mu.Lock()
	if reads-beforeReads != 2 || writes != 1 {
		t.Error("fresh GET or single PUT lost", reads-beforeReads, writes)
	}
	ignored = false
	mu.Unlock()
	for _, mode := range []string{"equal", "different", "reviewer removed", "replacement", "PR removed"} {
		mu.Lock()
		reset(11)
		mu.Unlock()
		input = intent(mode)
		output, diagnostic, status = run("apply")
		if status != 0 || diagnostic != warnings {
			t.Fatal("confirmed Team policy", mode, status, output, diagnostic)
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
		changed := writes != beforeWrites
		mu.Unlock()
		if changed {
			t.Fatal("canonical no-op wrote")
		}
	}
	for _, mode := range []string{"reviewer removed", "PR removed"} {
		mu.Lock()
		reset(11)
		ignored, retained = true, true
		mu.Unlock()
		input = intent(mode)
		output, diagnostic, status = run("apply")
		if status != 1 || output != "" || !strings.Contains(diagnostic, "writes may already have applied") {
			t.Fatal("retained canonical Team confirmed removal", mode, status, output, diagnostic)
		}
		checkSource(input)
	}
	for _, invalid := range []any{nil, "11", 1.5} {
		mu.Lock()
		reset(invalid)
		beforeWrites := writes
		mu.Unlock()
		input = intent("equal")
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
			t.Fatal("malformed count wrote")
		}
	}
}
