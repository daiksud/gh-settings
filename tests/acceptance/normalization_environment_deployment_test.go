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
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestEnvironmentDeploymentNormalizationAcrossExecutableCommands(t *testing.T) {
	for _, raw := range []bool{false, true} {
		t.Run(strconv.FormatBool(raw), func(t *testing.T) {
			binary := acceptanceBinary(t)
			var mu sync.Mutex
			state := map[string]any{"id": 5, "name": "production", "protection_rules": []any{}, "deployment_branch_policy": map[string]any{"protected_branches": raw, "custom_branch_policies": raw}, "can_admins_bypass": false}
			patterns := map[int]map[string]any{7: {"id": 7, "type": "branch", "name": "LATENT_PATTERN"}}
			reset := func() {
				state["protection_rules"], state["can_admins_bypass"], state["deployment_branch_policy"] = []any{}, false, map[string]any{"protected_branches": raw, "custom_branch_policies": raw}
				patterns = map[int]map[string]any{7: {"id": 7, "type": "branch", "name": "LATENT_PATTERN"}}
			}
			ignored, includeOther := true, false
			mode := "no_restriction"
			description := "current"
			generalWrites, envWrites, postDetails := 0, 0, 0
			awaitingReadback := false
			var history []string
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
					base["id"], base["full_name"], base["archived"], base["description"] = 71, "owner/current", false, description
					json.NewEncoder(w).Encode(base)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/current/environments":
					if r.Method != "GET" {
						t.Error("inventory write")
						w.WriteHeader(400)
						return
					}
					entries := []any{map[string]any{"id": 5, "name": "production"}}
					if includeOther {
						entries = append(entries, map[string]any{"id": 9, "name": "UNMANAGED"})
					}
					json.NewEncoder(w).Encode(map[string]any{"total_count": len(entries), "environments": entries})
				case "/repos/owner/current/environments/UNMANAGED":
					t.Error("unlisted environment detail acquired")
					w.WriteHeader(400)
				case "/repos/owner/current/environments/production":
					if r.Method == "GET" {
						if awaitingReadback {
							postDetails++
						}
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
					want := map[string]any{"wait_timer": float64(0), "prevent_self_review": false, "can_admins_bypass": false, "reviewers": []any{}}
					if mode == "name_only" {
						want = map[string]any{}
					}
					var desiredMode any
					if mode == "protected_branches_only" {
						desiredMode = map[string]any{"protected_branches": true, "custom_branch_policies": false}
					}
					if mode == "selected_branches_and_tags" {
						desiredMode = map[string]any{"protected_branches": false, "custom_branch_policies": true}
					}
					if mode != "omit" && mode != "name_only" {
						want["deployment_branch_policy"] = desiredMode
					}
					if !reflect.DeepEqual(payload, want) {
						t.Error("PUT differs from literal requested fields", payload, want)
					}
					envWrites++
					awaitingReadback = true
					history = append(history, "PUT environment")
					if mode == "omit" || mode == "name_only" {
						state["deployment_branch_policy"] = map[string]any{"protected_branches": !raw, "custom_branch_policies": !raw}
					} else if !ignored {
						state["deployment_branch_policy"] = desiredMode
					}
					state["protection_rules"], state["can_admins_bypass"] = []any{}, false
					if mode == "name_only" {
						state["protection_rules"] = []any{map[string]any{"type": "wait_timer", "wait_timer": 30}}
						state["can_admins_bypass"] = true
					}
					ack := map[string]any{}
					for k, v := range state {
						ack[k] = v
					}
					if mode != "omit" && mode != "name_only" {
						ack["deployment_branch_policy"] = desiredMode
					}
					json.NewEncoder(w).Encode(ack)
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					fmt.Fprint(w, `{"total_count":0,"custom_deployment_protection_rules":[]}`)
				case "/repos/owner/current/environments/production/deployment-branch-policies":
					if r.Method == "GET" {
						policy, ok := state["deployment_branch_policy"].(map[string]any)
						if !ok || policy["custom_branch_policies"] != true {
							t.Error("pattern applicability guessed", state)
						}
						entries := []any{}
						ids := []int{}
						for id := range patterns {
							ids = append(ids, id)
						}
						sort.Ints(ids)
						for _, id := range ids {
							entries = append(entries, patterns[id])
						}
						json.NewEncoder(w).Encode(map[string]any{"total_count": len(entries), "branch_policies": entries})
						return
					}
					if r.Method != "POST" {
						t.Error("unexpected pattern write", r.Method)
						w.WriteHeader(400)
						return
					}
					var payload map[string]any
					json.NewDecoder(r.Body).Decode(&payload)
					if !reflect.DeepEqual(payload, map[string]any{"type": "branch", "name": "wanted"}) {
						t.Error("pattern differs from literal intent", payload)
					}
					history = append(history, "POST pattern")
					patterns[8] = map[string]any{"id": 8, "type": "branch", "name": "wanted"}
					json.NewEncoder(w).Encode(patterns[8])
				case "/repos/owner/current/environments/production/deployment-branch-policies/7":
					if r.Method == "GET" {
						json.NewEncoder(w).Encode(map[string]any{"id": 7, "type": "branch", "name": "LATENT_PATTERN"})
						return
					}
					if r.Method != "DELETE" {
						t.Error(r.Method)
						w.WriteHeader(400)
						return
					}
					history = append(history, "DELETE pattern")
					delete(patterns, 7)
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
				mu.Lock()
				awaitingReadback = false
				mu.Unlock()
				r, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
				if err != nil {
					t.Fatal(err)
				}
				status := 0
				if r.runErr != nil {
					var failed *exec.ExitError
					if !errors.As(r.runErr, &failed) {
						t.Fatal(r.runErr)
					}
					status = failed.ExitCode()
				}
				return r.stdout.String(), r.stderr.String(), status
			}
			intent := func(choice string, withPatterns bool, about string) string {
				t.Helper()
				mu.Lock()
				mode = choice
				mu.Unlock()
				entry := "{name: production}"
				if choice != "name_only" {
					entry = "{name: production, deployment_protection_rules: {wait_timer: 0, prevent_self_review: false, allow_administrators_to_bypass_configured_protection_rules: false, required_reviewers: []}"
					if choice != "omit" {
						entry += ", deployment_branches_and_tags: " + choice
					}
					if withPatterns {
						entry += ", deployment_branch_or_tag_rules: [{ref_type: branch, name_pattern: wanted}]"
					}
					entry += "}"
				}
				input := repositoryInput(about + "settings: {environments: [" + entry + "]}\n")
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
			path := "repository.settings.environments.deployment_branches_and_tags"
			warning := `Warning: repository.settings.environments.deployment_branches_and_tags: observed {"protected_branches":` + strconv.FormatBool(raw) + `,"custom_branch_policies":` + strconv.FormatBool(raw) + `} is outside UI choices; using "no_restriction" (projection only). Environment "production".`
			output, diagnostic, status := run("export", "--environments")
			if status != 0 || output != "" || diagnostic != warning+"\n" {
				t.Fatal("export", status, output, diagnostic)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || !strings.Contains(string(data), "deployment_branches_and_tags: no_restriction") || strings.Contains(string(data), "LATENT_PATTERN") || strings.Contains(string(data), "protected_branches:") {
				t.Fatal(string(data), err)
			}
			report, summary := filepath.Join(fixture.root, "environment.json"), filepath.Join(fixture.root, "environment.md")
			for _, choice := range []string{"no_restriction", "protected_branches_only", "selected_branches_and_tags"} {
				input := intent(choice, false, "")
				output, diagnostic, status = run("plan")
				decision := path + ` (Environment "production"): "no_restriction" (normalized observation) → "` + choice + `" (write required)`
				if status != 0 || diagnostic != "" || strings.Count(output, warning) != 1 || !strings.Contains(output, decision) || !strings.Contains(output, "effective before policy") || strings.Contains(output, "LATENT_PATTERN") {
					t.Fatal("mode plan", status, output, diagnostic)
				}
				output, diagnostic, status = run("action", "plan", report)
				data, err = os.ReadFile(report)
				var saved struct {
					Outcome      string
					Managed      int
					Environments []struct {
						Operation     string
						Normalization *struct {
							Path      string
							RawPolicy struct {
								Protected *bool `json:"protected_branches"`
								Custom    *bool `json:"custom_branch_policies"`
							} `json:"raw_policy"`
							Fallback string
						} `json:"deployment_mode_normalization"`
					}
				}
				decodeErr := json.Unmarshal(data, &saved)
				if err != nil || decodeErr != nil || status != 0 || output != "" || diagnostic != "" || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Environments) != 1 {
					t.Fatal("private report", string(data), diagnostic, err, decodeErr)
				}
				record := saved.Environments[0].Normalization
				if saved.Environments[0].Operation != "update" || record == nil || record.Path != path || record.RawPolicy.Protected == nil || record.RawPolicy.Custom == nil || *record.RawPolicy.Protected != raw || *record.RawPolicy.Custom != raw || record.Fallback != "no_restriction" || strings.Contains(string(data), "LATENT_PATTERN") {
					t.Fatal("closed bool-pair provenance", string(data))
				}
				checkSource(input)
			}
			output, diagnostic, status = run("action", "summary", report, summary)
			data, err = os.ReadFile(summary)
			if err != nil || status != 0 || output != "" || diagnostic != "" || strings.Count(html.UnescapeString(string(data)), warning) != 1 || !strings.Contains(string(data), "effective before policy") {
				t.Fatal("detailed summary", string(data), diagnostic, err)
			}
			private := filepath.Join(fixture.root, "private.md")
			output, diagnostic, status = run("action", "summary", report, private, "status-only")
			data, err = os.ReadFile(private)
			if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), path) || strings.Contains(string(data), "production") || strings.Contains(string(data), "protected_branches") {
				t.Fatal("status privacy", string(data), diagnostic, err)
			}
			mu.Lock()
			readOnlyWrites := envWrites + generalWrites + len(history)
			includeOther = true
			mu.Unlock()
			if readOnlyWrites != 0 {
				t.Fatal("read-only wrote")
			}
			input := intent("omit", false, "")
			output, diagnostic, status = run("plan")
			if status != 0 || diagnostic != "" || !strings.Contains(output, "API-controlled") || !strings.Contains(output, "effective before policy") || strings.Contains(output, "Warning:") || strings.Contains(output, "LATENT_PATTERN") || strings.Contains(output, "UNMANAGED") || strings.Contains(output, "protected_branches") || strings.Contains(output, "custom_branch_policies") {
				t.Fatal("omitted mode plan", status, output, diagnostic)
			}
			output, diagnostic, status = run("apply")
			if status != 0 || diagnostic != "" || strings.Contains(output, "LATENT_PATTERN") {
				t.Fatal("omitted mode outcome", status, output, diagnostic)
			}
			checkSource(input)
			input = intent("name_only", false, "")
			output, diagnostic, status = run("apply")
			if status != 0 || diagnostic != "" {
				t.Fatal("API-controlled name-only outcome", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			reset()
			beforeFresh := postDetails
			mu.Unlock()
			input = intent("no_restriction", false, "")
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "write may have applied") || strings.Contains(diagnostic, "LATENT_PATTERN") || strings.Contains(diagnostic, "UNMANAGED") {
				t.Fatal("same-name raw pair confirmed fallback", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			freshReads := postDetails - beforeFresh
			ignored = false
			mu.Unlock()
			if freshReads != 1 {
				t.Fatal("no independent fresh detail read")
			}
			output, diagnostic, status = run("apply")
			if status != 0 || diagnostic != warning+"\n" {
				t.Fatal("canonical confirmation", status, output, diagnostic)
			}
			checkSource(input)
			for _, choice := range []string{"no_restriction", "protected_branches_only", "selected_branches_and_tags"} {
				if choice != "no_restriction" {
					mu.Lock()
					reset()
					mu.Unlock()
					input = intent(choice, false, "")
					output, diagnostic, status = run("apply")
					if status != 0 || diagnostic != warning+"\n" {
						t.Fatal("different mode", choice, status, output, diagnostic)
					}
					checkSource(input)
				}
				mu.Lock()
				before := envWrites
				mu.Unlock()
				output, diagnostic, status = run("plan")
				if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
					t.Fatal("canonical no-op/unlisted privacy", choice, status, output, diagnostic)
				}
				output, diagnostic, status = run("apply")
				if status != 0 || diagnostic != "" || strings.Contains(output, "UNMANAGED") {
					t.Fatal("repeat apply", status, output, diagnostic)
				}
				mu.Lock()
				unchanged := envWrites == before
				mu.Unlock()
				if !unchanged {
					t.Fatal("canonical equality wrote")
				}
			}
			mu.Lock()
			reset()
			beforePatterns := len(history)
			beforeBlocked := envWrites
			mu.Unlock()
			input = intent("omit", true, "")
			output, diagnostic, status = run("apply")
			if status != 1 || strings.Contains(diagnostic, "Warning:") || strings.Contains(diagnostic, "LATENT_PATTERN") {
				t.Fatal("fallback proved pattern applicability", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			noWrite := envWrites == beforeBlocked && len(history) == beforePatterns
			ignored = true
			mu.Unlock()
			if !noWrite {
				t.Fatal("patterns-only wrote under unknown mode")
			}
			input = intent("selected_branches_and_tags", true, "")
			output, diagnostic, status = run("apply")
			if status != 1 || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "not in custom mode") {
				t.Fatal("raw custom flag proved active mode", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			if len(history) != beforePatterns+1 {
				t.Fatal("pattern CRUD used unknown applicability", history)
			}
			ignored = false
			historyStart := len(history)
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 0 || diagnostic != warning+"\n" {
				t.Fatal("explicit selected mode/CRUD", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			actualHistory := append([]string{}, history[historyStart:]...)
			reset()
			ignored = true
			mu.Unlock()
			if !reflect.DeepEqual(actualHistory, []string{"PUT environment", "POST pattern", "DELETE pattern"}) {
				t.Fatal("pattern write order", actualHistory)
			}
			input = intent("no_restriction", false, "about: {description: updated}\n")
			output, diagnostic, status = run("apply")
			if status != 1 || strings.Count(diagnostic, warning) != 2 || !strings.Contains(diagnostic, "earlier successful writes remain applied") || strings.Contains(diagnostic, "LATENT_PATTERN") {
				t.Fatal("prior resource effects", status, output, diagnostic)
			}
			checkSource(input)
			mu.Lock()
			progressed := description == "updated" && generalWrites == 1
			state["deployment_branch_policy"] = map[string]any{"protected_branches": nil, "custom_branch_policies": true}
			beforeInvalid := envWrites
			mu.Unlock()
			if !progressed {
				t.Fatal("REST effects lost")
			}
			input = intent("no_restriction", false, "")
			for _, args := range [][]string{{"export", "--environments"}, {"apply"}} {
				output, diagnostic, status = run(args...)
				if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") {
					t.Fatal("malformed no-save/no-write", args, status, output, diagnostic)
				}
				checkSource(input)
			}
			mu.Lock()
			malformedWrote := envWrites != beforeInvalid
			state["deployment_branch_policy"] = nil
			includeOther = false
			mu.Unlock()
			if malformedWrote {
				t.Fatal("malformed preflight wrote")
			}
			output, diagnostic, status = run("export", "--environments")
			data, err = os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if err != nil || status != 0 || output != "" || diagnostic != "" || !strings.Contains(string(data), "deployment_branches_and_tags: no_restriction") || strings.Contains(string(data), "LATENT_PATTERN") || strings.Contains(string(data), "deployment_branch_or_tag_rules") {
				t.Fatal("known null export/latent privacy", status, string(data), diagnostic, err)
			}
		})
	}
}
