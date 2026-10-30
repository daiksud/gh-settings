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

func TestRulesetRequiredApprovalsNormalizationAcrossExecutableCommands(t *testing.T) {
	for _, both := range []bool{false, true} {
		t.Run(fmt.Sprint("both=", both), func(t *testing.T) {
			binary := acceptanceBinary(t)
			var mu sync.Mutex
			var state map[string]any
			json.Unmarshal([]byte(acceptanceRuleset), &state)
			enforcement := "active"
			if both {
				enforcement = "disabled"
			}
			reset := func(count any) {
				state["enforcement"] = "active"
				if both {
					state["enforcement"] = "evaluate"
				}
				state["rules"] = []any{acceptanceApprovalsRule(count)}
				if both {
					state["rules"] = append([]any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "FUTURE_SEVERITY"}}}, state["rules"].([]any)...)
				}
			}
			reset(11)
			ignored, includeOther := true, false
			description, desiredNative := "current", 0
			reads, writes, generalWrites := 0, 0, 0
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
						generalWrites++
						description = "updated"
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
						t.Error("unmanaged write")
						w.WriteHeader(400)
						return
					}
					var other map[string]any
					json.Unmarshal([]byte(acceptanceRuleset), &other)
					other["id"], other["name"], other["enforcement"] = 9, "UNMANAGED_OBJECT", "UNMANAGED_MODE"
					other["rules"] = []any{acceptanceApprovalsRule(99), map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "UNMANAGED_SEVERITY"}}}
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
					rules := []any{}
					if both {
						rules = append(rules, map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "errors"}})
					}
					if desiredNative != -1 {
						rules = append(rules, acceptanceApprovalsRule(desiredNative))
					}
					want := map[string]any{"name": "basic", "target": "branch", "enforcement": enforcement, "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": rules}
					if !reflect.DeepEqual(payload, want) {
						t.Error("independent intended payload", payload, want)
					}
					writes++
					if !ignored {
						for k, v := range payload {
							state[k] = v
						}
					}
					// The acknowledgment may claim success while the independent GET retains the old rule.
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
			intent := func(choice, about string) string {
				t.Helper()
				native, valid := map[string]int{"0": 0, "10": 10, "removed": -1}[choice]
				if !valid {
					t.Fatal(choice)
				}
				mu.Lock()
				desiredNative = native
				mu.Unlock()
				rules := "{}"
				if choice != "removed" {
					rules = "{require_a_pull_request_before_merging: {required_approvals: " + choice + "}}"
				}
				if both {
					if choice == "removed" {
						rules = "{require_code_quality_results: {severity: errors}}"
					} else {
						rules = "{require_code_quality_results: {severity: errors}, require_a_pull_request_before_merging: {required_approvals: " + choice + "}}"
					}
				}
				input := repositoryInput(about + "settings: {rulesets: [{id: 7, ruleset_name: basic, enforcement_status: " + enforcement + ", target_branches: {include_default_branch: true}, rules: " + rules + "}]}\n")
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
			path := "repository.settings.rulesets.rules.require_a_pull_request_before_merging.required_approvals"
			warning := `Warning: repository.settings.rulesets.rules.require_a_pull_request_before_merging.required_approvals: observed 11 is outside UI choices; using 0 (projection only). Ruleset 7 "basic".`
			qualityWarning := `Warning: repository.settings.rulesets.rules.require_code_quality_results.severity: observed "FUTURE_SEVERITY" is outside UI choices; using "errors" (projection only). Ruleset 7 "basic".`
			enfWarning := `Warning: repository.settings.rulesets.enforcement_status: observed "evaluate" is outside UI choices; using "disabled" (projection only). Ruleset 7 "basic".`
			orderedWarnings := warning + "\n"
			if both {
				orderedWarnings = enfWarning + "\n" + orderedWarnings + qualityWarning + "\n"
			}
			output, diagnostic, status := run("export", "--rulesets")
			if status != 0 || output != "" || diagnostic != orderedWarnings {
				t.Fatal("export", status, output, diagnostic)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || !strings.Contains(string(data), "required_approvals: 0") || strings.Contains(string(data), "FUTURE") {
				t.Fatal(string(data), err)
			}
			report, summary := filepath.Join(fixture.root, "approvals.json"), filepath.Join(fixture.root, "approvals.md")
			for _, choice := range []string{"0", "10", "removed"} {
				input := intent(choice, "")
				output, diagnostic, status = run("plan")
				after := choice + ` (write required)`
				if choice == "removed" {
					after = "inactive (rule removed; write required)"
				}
				decision := path + ` (Ruleset 7 "basic"): 0 (normalized observation) → ` + after
				if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) || strings.Count(output, "Effective policy:") != 1 || (both && (strings.Count(output, enfWarning) != 1 || strings.Index(output, enfWarning) > strings.Index(output, warning) || strings.Count(output, qualityWarning) != 1 || strings.Index(output, warning) > strings.Index(output, qualityWarning))) {
					t.Fatal("decision", status, output, diagnostic)
				}
				output, diagnostic, status = run("action", "plan", report)
				data, err = os.ReadFile(report)
				var saved struct {
					Outcome  string
					Managed  int
					Rulesets []struct {
						Operation string
						Approvals struct {
							Path     string
							Raw      int `json:"raw_value"`
							Fallback int
						} `json:"required_approvals_normalization"`
						Enforcement *json.RawMessage `json:"enforcement_normalization"`
						Quality     *json.RawMessage `json:"quality_severity_normalization"`
					}
				}
				decodeErr := json.Unmarshal(data, &saved)
				if err != nil || decodeErr != nil || status != 0 || output != "" || diagnostic != "" || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Rulesets) != 1 {
					t.Fatal("private report", string(data), err, decodeErr, diagnostic)
				}
				item := saved.Rulesets[0]
				if item.Operation != "update" || item.Approvals.Path != path || item.Approvals.Raw != 11 || item.Approvals.Fallback != 0 || (item.Enforcement != nil) != both || (item.Quality != nil) != both {
					t.Fatal("private evidence", string(data))
				}
				checkSource(input)
			}
			output, diagnostic, status = run("action", "summary", report, summary)
			data, err = os.ReadFile(summary)
			visible := html.UnescapeString(string(data))
			if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(visible, warning) != 1 || !strings.Contains(visible, "inactive (rule removed; write required)") {
				t.Fatal("summary", visible, diagnostic, err)
			}
			private := filepath.Join(fixture.root, "private.md")
			output, diagnostic, status = run("action", "summary", report, private, "status-only")
			data, err = os.ReadFile(private)
			if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), path) || strings.Contains(string(data), "FUTURE") || strings.Contains(string(data), "Ruleset 7") {
				t.Fatal("status privacy", string(data), diagnostic, err)
			}
			mu.Lock()
			beforeWrites := writes + generalWrites
			mu.Unlock()
			if beforeWrites != 0 {
				t.Fatal("read-only wrote")
			}
			// Same identity/name, normalized-equal zero: a canonical PUT must not hide out-of-range GET count.
			input := intent("0", "")
			mu.Lock()
			beforeReads := reads
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Count(diagnostic, warning) != 2 || (both && (strings.Count(diagnostic, enfWarning) != 2 || !strings.HasPrefix(diagnostic, orderedWarnings))) || !strings.Contains(diagnostic, "writes may already have applied") {
				t.Fatal("raw confirmation", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			observedReads := reads - beforeReads
			ignored = false
			mu.Unlock()
			if observedReads != 2 {
				t.Fatal("fresh readback", observedReads)
			}
			for _, choice := range []string{"0", "10", "removed"} {
				mu.Lock()
				reset(11)
				mu.Unlock()
				input = intent(choice, "")
				output, diagnostic, status = run("apply")
				if status != 0 || diagnostic != orderedWarnings {
					t.Fatal("canonical write or actual removal", choice, status, output, diagnostic)
				}
				checkSource(input)
				mu.Lock()
				before := writes
				includeOther = true
				mu.Unlock()
				output, diagnostic, status = run("plan")
				if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
					t.Fatal("canonical or absent no-op", choice, status, output, diagnostic)
				}
				output, diagnostic, status = run("apply")
				if status != 0 || diagnostic != "" || strings.Contains(output, "UNMANAGED") {
					t.Fatal("unlisted apply", status, output, diagnostic)
				}
				mu.Lock()
				unchanged := writes == before
				mu.Unlock()
				if !unchanged {
					t.Fatal("canonical no-op wrote")
				}
			}
			// Retaining even a canonical rule must fail when the requested whole-object rule is absent.
			mu.Lock()
			reset(0)
			state["enforcement"] = enforcement
			if both {
				state["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["severity"] = "errors"
			}
			ignored = true
			mu.Unlock()
			input = intent("removed", "")
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || !strings.Contains(diagnostic, "writes may already have applied") {
				t.Fatal("retained canonical rule", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			reset(11)
			mu.Unlock()
			input = intent("0", "about: {description: updated}\n")
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "earlier successful writes remain applied") || strings.Contains(diagnostic, "UNMANAGED") {
				t.Fatal("partial REST", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			progressed := description == "updated" && generalWrites == 1
			reset(nil)
			beforeInvalid := writes
			mu.Unlock()
			if !progressed {
				t.Fatal("lost REST progress")
			}
			input = intent("0", "")
			for _, args := range [][]string{{"export", "--rulesets"}, {"apply"}} {
				output, diagnostic, status = run(args...)
				if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
					t.Fatal("null approvals", args, status, output, diagnostic)
				}
				checkSource(input)
			}
			mu.Lock()
			malformedWrote := writes != beforeInvalid
			state["rules"] = []any{}
			state["enforcement"] = "active"
			includeOther = false
			mu.Unlock()
			if malformedWrote {
				t.Fatal("invalid wrote")
			}
			output, diagnostic, status = run("export", "--rulesets")
			data, err = os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Contains(string(data), "require_a_pull_request_before_merging") || strings.Contains(string(data), "required_approvals") {
				t.Fatal("absent invented rule", status, string(data), diagnostic, err)
			}
		})
	}
}

// This literal policy fixture is independent of constructed changes and request bodies.
func acceptanceApprovalsRule(value any) map[string]any {
	if count, ok := value.(int); ok {
		value = float64(count)
	}
	return map[string]any{"type": "pull_request", "parameters": map[string]any{"allowed_merge_methods": []any{"merge", "rebase", "squash"}, "dismiss_stale_reviews_on_push": false, "require_code_owner_review": false, "require_extra_approval_for_unattributed_changes": false, "require_last_push_approval": false, "required_approving_review_count": value, "required_review_thread_resolution": false, "required_reviewers": []any{}}}
}
