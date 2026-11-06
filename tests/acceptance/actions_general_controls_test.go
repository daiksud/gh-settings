//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestActionsGeneralControlsExportPlanApply(t *testing.T) {
	for _, tc := range []struct {
		name, flag, endpoint, key, declaration, exported, visibility, ownerType string
		before, after                                                           any
	}{
		{"cache retention", "--cache-retention", "cache/retention-limit", "max_cache_retention_days", "{cache: {cache_retention: 14}}", "cache_retention: 7", "public", "User", float64(7), float64(14)},
		{"cache storage", "--cache-storage", "cache/storage-limit", "max_cache_size_gb", "{cache: {cache_size_eviction_limit: 20}}", "cache_size_eviction_limit: 10", "public", "User", float64(10), float64(20)},
		{"user access", "--actions-access", "permissions/access", "access_level", "{access: accessible_from_repositories_owned_by_user}", "access: not_accessible", "private", "User", "none", "user"},
		{"organization access", "--actions-access", "permissions/access", "access_level", "{access: accessible_from_repositories_in_organization}", "access: not_accessible", "private", "Organization", "none", "organization"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary := acceptanceBinary(t)
			var mu sync.Mutex
			value := tc.before
			mode := ""
			var operations []string
			var writes []map[string]any
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/repos/owner/current/actions/"+tc.endpoint {
					operations = append(operations, r.Method)
					switch r.Method {
					case "GET":
						if mode == "denied" {
							w.WriteHeader(403)
							fmt.Fprint(w, `{}`)
							return
						}
						if mode == "incomplete" {
							fmt.Fprint(w, `{}`)
							return
						}
						json.NewEncoder(w).Encode(map[string]any{tc.key: value})
					case "PUT":
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Error(err)
						}
						writes = append(writes, body)
						if !reflect.DeepEqual(body, map[string]any{tc.key: tc.after}) {
							t.Error("overbroad or incorrect native request", body)
						}
						if mode == "write-denied" {
							w.WriteHeader(422)
							fmt.Fprint(w, `{"message":"upper policy refused"}`)
							return
						}
						if mode != "unchanged" {
							value = body[tc.key]
						}
						w.WriteHeader(204)
					default:
						t.Error("unexpected method", r.Method)
						w.WriteHeader(400)
					}
					return
				}
				if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if r.Method != "GET" {
					t.Error("unrelated mutation", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					var body map[string]any
					json.Unmarshal([]byte(planResponse), &body)
					body["id"] = float64(123)
					body["full_name"] = "owner/current"
					body["visibility"] = tc.visibility
					body["private"] = tc.visibility == "private"
					body["owner"] = map[string]any{"type": tc.ownerType, "login": "owner"}
					json.NewEncoder(w).Encode(body)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/current/actions/permissions":
					fmt.Fprint(w, `{"enabled":true,"allowed_actions":"all","sha_pinning_required":false}`)
				case "/repos/owner/current/actions/permissions/artifact-and-log-retention":
					fmt.Fprint(w, `{"days":30,"maximum_allowed_days":90}`)
				default:
					t.Error("unselected endpoint", r.URL)
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
			write := func(s string) {
				t.Helper()
				if err := writeSettingsFile(fixture.work, actionsPolicyInput(s)); err != nil {
					t.Fatal(err)
				}
			}
			run("export")
			run("export", "--actions")
			run("export", "--retention")
			mu.Lock()
			if len(operations) != 0 {
				t.Error("old export scopes acquired new control", operations)
			}
			mu.Unlock()
			run("export", tc.flag)
			run("validate")
			path := filepath.Join(fixture.work, ".github/settings.yml")
			data, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(data), tc.exported) || strings.Contains(string(data), tc.key+":") {
				t.Fatal(string(data), err)
			}
			if got := run("plan"); got != "No differences for managed repository settings.\n" {
				t.Fatal(got)
			}
			write(tc.declaration)
			if got := run("plan"); !strings.Contains(got, "repository.settings.actions.general.") {
				t.Fatal(got)
			}
			report := filepath.Join(fixture.root, "actions-general-plan.json")
			run("action", "plan", report)
			reportData, err := os.ReadFile(report)
			var typed map[string]any
			if err != nil || json.Unmarshal(reportData, &typed) != nil || typed["managed"] != float64(1) {
				t.Fatal(string(reportData), err)
			}
			summary := filepath.Join(fixture.root, "actions-general-summary.md")
			summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64), "GH_TOKEN=", "GITHUB_TOKEN=")
			reads := fixture.reads.Load()
			r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
			summaryData, readErr := os.ReadFile(summary)
			if err != nil || r.runErr != nil || readErr != nil || fixture.reads.Load() != reads || !strings.Contains(string(summaryData), "repository.settings.actions.general.") {
				t.Fatal(err, readErr, r.stderr.String(), string(summaryData))
			}
			mu.Lock()
			operations = nil
			mu.Unlock()
			run("apply")
			mu.Lock()
			if !reflect.DeepEqual(operations, []string{"GET", "GET", "PUT", "GET"}) || len(writes) != 1 || value != tc.after {
				t.Error(operations, writes, value)
			}
			mu.Unlock()
			run("apply")
			if got := run("plan"); got != "No differences for managed repository settings.\n" {
				t.Fatal(got)
			}
			mu.Lock()
			if len(writes) != 1 {
				t.Error("matching value rewritten", writes)
			}
			operations = nil
			mu.Unlock()
			write("{}")
			run("plan")
			run("apply")
			mu.Lock()
			if len(operations) != 0 {
				t.Error("omitted control acquired", operations)
			}
			mu.Unlock()
			for _, failure := range []string{"denied", "incomplete"} {
				mu.Lock()
				mode = failure
				mu.Unlock()
				before, _ := os.ReadFile(path)
				r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"export", tc.flag}, false)
				after, _ := os.ReadFile(path)
				if err != nil || r.runErr == nil || r.stdout.Len() != 0 || string(before) != string(after) {
					t.Fatal(failure, err, r.stderr.String(), string(after))
				}
			}
			for _, failure := range []string{"write-denied", "unchanged"} {
				write(tc.declaration)
				mu.Lock()
				mode = failure
				value = tc.before
				operations = nil
				previous := len(writes)
				mu.Unlock()
				r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
				if err != nil || r.runErr == nil {
					t.Fatal(failure, err, r.stderr.String())
				}
				mu.Lock()
				if len(writes) != previous+1 || value != tc.before {
					t.Error("failure retried or fabricated success", writes, value)
				}
				if failure == "unchanged" && !reflect.DeepEqual(operations, []string{"GET", "GET", "PUT", "GET"}) {
					t.Error("independent readback missing", operations)
				}
				mu.Unlock()
			}
		})
	}
}
