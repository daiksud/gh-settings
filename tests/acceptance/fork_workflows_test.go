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

func TestForkWorkflowExportPlanApplyAndSummary(t *testing.T) {
	for _, private := range []bool{false, true} {
		t.Run(fmt.Sprint("private=", private), func(t *testing.T) {
			binary := acceptanceBinary(t)
			var mu sync.Mutex
			visibility := "public"
			flag := "--fork-approval"
			endpoint := "fork-pr-contributor-approval"
			if private {
				visibility = "private"
				flag = "--private-fork-workflows"
				endpoint = "fork-pr-workflows-private-repos"
			}
			approval := "first_time_contributors"
			values := [4]bool{true, false, true, false}
			mode := ""
			reads, writes, redirected := 0, 0, 0
			payloads := []map[string]any{}
			keys := []string{"run_workflows_from_fork_pull_requests", "send_write_tokens_to_workflows", "send_secrets_and_variables", "require_approval_for_fork_pr_workflows"}
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/repos/other/redirected" {
					redirected++
					fmt.Fprint(w, `{}`)
					return
				}
				if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if r.URL.Path == "/repos/owner/current/actions/permissions/"+endpoint {
					if r.Method == "GET" {
						reads++
						switch mode {
						case "hidden":
							fmt.Fprint(w, `{}`)
							return
						case "denied":
							w.WriteHeader(403)
							fmt.Fprint(w, `{}`)
							return
						case "get-redirect":
							w.Header().Set("Location", "/repos/other/redirected")
							w.WriteHeader(301)
							return
						}
						if private {
							fields := map[string]bool{}
							for i, k := range keys {
								fields[k] = values[i]
							}
							json.NewEncoder(w).Encode(fields)
						} else {
							fmt.Fprintf(w, `{"approval_policy":%q}`, approval)
						}
						return
					}
					if r.Method != "PUT" {
						t.Error("unexpected mutation", r.Method)
						w.WriteHeader(400)
						return
					}
					writes++
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					payloads = append(payloads, body)
					if mode == "put-redirect" {
						w.Header().Set("Location", "/repos/other/redirected")
						w.WriteHeader(307)
						return
					}
					if mode == "write-denied" {
						w.WriteHeader(422)
						fmt.Fprint(w, `{"message":"upper policy refused"}`)
						return
					}
					if mode != "enforced" {
						if private {
							for i, k := range keys {
								if v, ok := body[k]; ok {
									values[i] = v.(bool)
								}
							}
						} else {
							approval = body["approval_policy"].(string)
						}
					}
					w.WriteHeader(204)
					return
				}
				if r.Method != "GET" {
					t.Error("unrelated mutation", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					var fields map[string]any
					json.Unmarshal([]byte(planResponse), &fields)
					fields["visibility"] = visibility
					json.NewEncoder(w).Encode(fields)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				default:
					t.Error("unrelated policy endpoint", r.URL)
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
			write := func(value string) {
				t.Helper()
				if err := writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings: {actions: "+value+"}\n"); err != nil {
					t.Fatal(err)
				}
			}
			run("export")
			mu.Lock()
			if reads != 0 {
				t.Fatal("ordinary export requested fork policy")
			}
			mu.Unlock()
			run("export", flag)
			run("validate")
			if got := run("plan"); got != "No differences for managed repository settings.\n" {
				t.Fatal(got)
			}
			config := filepath.Join(fixture.work, ".github/settings.yml")
			data, _ := os.ReadFile(config)
			if private {
				if !strings.Contains(string(data), "send_write_tokens_to_workflows: false") || strings.Contains(string(data), "fork_pr_approval_policy:") {
					t.Fatal(string(data))
				}
			} else if !strings.Contains(string(data), "fork_pr_approval_policy: first_time_contributors") || strings.Contains(string(data), "private_fork_workflows:") {
				t.Fatal(string(data))
			}
			desired := "{fork_pr_approval_policy: all_external_contributors}"
			needle := "fork_pr_approval_policy"
			if private {
				desired = "{private_fork_workflows: {run_workflows_from_fork_pull_requests: true, send_secrets_and_variables: false}}"
				needle = "send_secrets_and_variables: true → false"
			}
			write(desired)
			plan := run("plan")
			if !strings.Contains(plan, needle) {
				t.Fatal(plan)
			}
			report := filepath.Join(fixture.root, "fork-report.json")
			run("action", "plan", report)
			summary := filepath.Join(fixture.root, "fork-summary.md")
			summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64), "GH_TOKEN=", "GITHUB_TOKEN=")
			beforeReads := fixture.reads.Load()
			r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
			if err != nil || r.runErr != nil || fixture.reads.Load() != beforeReads {
				t.Fatal(err, r.stderr.String())
			}
			data, _ = os.ReadFile(summary)
			for _, line := range strings.Split(strings.TrimSpace(plan), "\n") {
				if !strings.Contains(html.UnescapeString(string(data)), line) {
					t.Fatal(string(data), line)
				}
			}
			run("apply")
			run("apply")
			if got := run("plan"); got != "No differences for managed repository settings.\n" {
				t.Fatal(got)
			}
			mu.Lock()
			want := map[string]any{"approval_policy": "all_external_contributors"}
			if private {
				want = map[string]any{"run_workflows_from_fork_pull_requests": true, "send_secrets_and_variables": false}
			}
			if writes != 1 || !reflect.DeepEqual(payloads[0], want) || private && (values[1] || values[3]) {
				t.Fatal(writes, payloads, values)
			}
			mu.Unlock()
			// Partial matching intent needs no mandatory parameter; a differing one may not replay it.
			if private {
				write("{private_fork_workflows: {send_secrets_and_variables: false}}")
				run("plan")
				run("apply")
				write("{private_fork_workflows: {send_secrets_and_variables: true}}")
				for _, cmd := range []string{"plan", "apply"} {
					r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{cmd}, false)
					if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "explicitly declared run_workflows") {
						t.Fatal(cmd, err, r.stderr.String())
					}
				}
			}

			if private {
				mu.Lock()
				values[0] = false
				beforeWrites := writes
				mu.Unlock()
				write("{private_fork_workflows: {run_workflows_from_fork_pull_requests: true}}")
				for _, cmd := range []string{"plan", "apply"} {
					r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{cmd}, false)
					if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "all four values explicitly") {
						t.Fatal(cmd, err, r.stderr.String())
					}
				}
				mu.Lock()
				if writes != beforeWrites {
					t.Fatal("latent enable wrote")
				}
				mu.Unlock()
				write("{private_fork_workflows: {run_workflows_from_fork_pull_requests: true, send_write_tokens_to_workflows: false, send_secrets_and_variables: false, require_approval_for_fork_pr_workflows: false}}")
				run("apply")
				mu.Lock()
				if values != [4]bool{true, false, false, false} || !reflect.DeepEqual(payloads[len(payloads)-1], map[string]any{"run_workflows_from_fork_pull_requests": true, "send_write_tokens_to_workflows": false, "send_secrets_and_variables": false, "require_approval_for_fork_pr_workflows": false}) {
					t.Fatal(values, payloads)
				}
				mu.Unlock()
			}
			for _, failure := range []string{"hidden", "denied", "get-redirect", "wrong visibility", "internal"} {
				mu.Lock()
				mode = failure
				originalVisibility := visibility
				if failure == "wrong visibility" {
					if private {
						visibility = "public"
					} else {
						visibility = "private"
					}
				}
				if failure == "internal" {
					visibility = "internal"
				}
				beforeWrites := writes
				mu.Unlock()
				original, _ := os.ReadFile(config)
				r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"export", flag}, false)
				after, _ := os.ReadFile(config)
				if err != nil || r.runErr == nil || r.stdout.Len() != 0 || string(after) != string(original) {
					t.Fatal(failure, err, r.stderr.String())
				}
				mu.Lock()
				if writes != beforeWrites || redirected != 0 {
					t.Fatal("read failure wrote/followed", failure)
				}
				visibility = originalVisibility
				mu.Unlock()
			}
			// Reverse the declared value so every failure attempts an actual changed policy.
			next := "{fork_pr_approval_policy: first_time_contributors}"
			if private {
				next = "{private_fork_workflows: {run_workflows_from_fork_pull_requests: true, send_secrets_and_variables: true}}"
			}
			write(next)
			for _, failure := range []string{"put-redirect", "write-denied", "enforced"} {
				mu.Lock()
				mode = failure
				beforeWrites := writes
				mu.Unlock()
				r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
				if err != nil || r.runErr == nil {
					t.Fatal(failure, err, r.stderr.String())
				}
				mu.Lock()
				if writes != beforeWrites+1 || redirected != 0 {
					t.Fatal("retried/followed", failure, writes, redirected)
				}
				mu.Unlock()
			}
			// Direct apply retains the valid public policy when the later private endpoint is inapplicable.
			if !private {
				mu.Lock()
				mode = ""
				mu.Unlock()
				write("{fork_pr_approval_policy: first_time_contributors, private_fork_workflows: {run_workflows_from_fork_pull_requests: false}}")
				r, err = executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
				if err != nil || r.runErr == nil || !strings.Contains(r.stderr.String(), "earlier writes remain applied") {
					t.Fatal(err, r.stderr.String())
				}
				mu.Lock()
				if approval != "first_time_contributors" {
					t.Fatal("earlier write rolled back", approval)
				}
				mu.Unlock()
			}
		})
	}
}
