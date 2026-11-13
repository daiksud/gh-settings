//go:build acceptance

package acceptance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A failing stderr must exercise warning delivery, not the already-fatal Info
// path. The controls prove that every other automatic acquisition is silent.
func TestExportWarningOutputFailure(t *testing.T) {
	for _, test := range []struct {
		name                           string
		normalized, readOnly, existing bool
	}{
		{"canonical writable", false, false, false},
		{"canonical unwritable", false, true, false},
		{"warning writable", true, false, false},
		{"warning unwritable first export", true, true, false},
		{"warning unwritable existing file", true, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := silentExportFixture(t, test.normalized)
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			const prior = "# exact prior bytes\nversion: daiksud/gh-settings@v0\nrepository: {}\n"
			if test.existing {
				if err := writeSettingsFile(fixture.work, prior); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, acceptanceBinary(t), "export")
			cmd.Dir, cmd.Env = fixture.work, env
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if test.readOnly {
				path := filepath.Join(fixture.root, "read-only-stderr")
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
				file, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				// Pass a real read-only fd 2 to the child. No parent copy error or
				// broken-pipe signal may stand in for product error propagation.
				cmd.Stderr = file
			}
			err := cmd.Run()
			if ctx.Err() != nil || stdout.Len() != 0 {
				t.Fatal("process timeout or unexpected stdout", ctx.Err(), stdout.String())
			}
			wantFailure := test.normalized && test.readOnly
			if wantFailure {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 {
					t.Fatalf("warning delivery failure must exit 1 before save, got %v", err)
				}
			} else if err != nil {
				t.Fatal("silent/writable control failed", err, stderr.String())
			}
			wantDiagnostic := ""
			if test.normalized && !test.readOnly {
				wantDiagnostic = "Warning: repository.settings.general.features.issue_permissions.creation_allowed_by: observed \"FUTURE\" is outside UI choices; using \"all_users\" (projection only).\n"
			}
			if stderr.String() != wantDiagnostic {
				t.Fatalf("unexpected diagnostic (Info would invalidate this regression): %q", stderr.String())
			}
			data, readErr := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			if wantFailure && !test.existing {
				if !os.IsNotExist(readErr) {
					t.Fatal("failed first export created a file", readErr, string(data))
				}
			} else if wantFailure {
				if readErr != nil || string(data) != prior {
					t.Fatal("failed re-export changed prior bytes", readErr, string(data))
				}
			} else if readErr != nil || !strings.Contains(string(data), "creation_allowed_by: all_users") || strings.Contains(string(data), "FUTURE") {
				t.Fatal("successful control did not save effective intent", readErr, string(data))
			}
			if err := fixture.checkRequests(false); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// All routes are literal complete observations for one personally owned public
// repository. Unknown routes, targets and mutations are test failures. Empty
// collections and known absent Pages/default setup avoid unrelated diagnostics.
func silentExportFixture(t *testing.T, normalized bool) *apiFixture {
	t.Helper()
	responses := map[string]string{
		"":                              strings.TrimSuffix(planResponse, "}") + `,"id":123,"full_name":"owner/current","archived":false,"visibility":"public","private":false,"owner":{"id":2,"login":"owner","type":"User"},"security_and_analysis":{"secret_scanning":{"status":"disabled"},"secret_scanning_push_protection":{"status":"disabled"}}}`,
		"/collaborators":                `[]`,
		"/invitations":                  `[]`,
		"/topics":                       `{"names":[]}`,
		"/labels":                       `[]`,
		"/rulesets":                     `[]`,
		"/actions/policies":             `{"total_count":0,"policies":[]}`,
		"/environments":                 `{"total_count":0,"environments":[]}`,
		"/actions/permissions":          `{"enabled":true,"allowed_actions":"all","sha_pinning_required":false}`,
		"/actions/permissions/workflow": `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`,
		"/actions/permissions/artifact-and-log-retention":   `{"days":90,"maximum_allowed_days":90}`,
		"/actions/cache/retention-limit":                    `{"max_cache_retention_days":7}`,
		"/actions/cache/storage-limit":                      `{"max_cache_size_gb":10}`,
		"/actions/permissions/fork-pr-contributor-approval": `{"approval_policy":"first_time_contributors"}`,
		"/copilot/code-review/settings":                     `{"custom_instructions_enabled":false,"mcp_enabled":false,"severity_filter":"no_filter","review_effort":{"configured":null,"effective":"balanced"},"available_review_efforts":["lite","balanced","max"]}`,
		"/interaction-limits/pulls/bypass-list":             `[]`,
		"/interaction-limits/pulls/creation-cap":            `{"enabled":false,"max_open_pull_requests":10,"include_drafts":false}`,
		"/automated-security-fixes":                         `{"enabled":false,"paused":false}`,
		"/code-scanning/default-setup":                      `{"state":"not-configured"}`,
		"/code-scanning/ai-scan":                            `{"pr_scan":"disabled","repository_scan":"disabled"}`,
		"/private-vulnerability-reporting":                  `{"enabled":false}`,
		"/immutable-releases":                               `{"enabled":false,"enforced_by_owner":false}`,
		"/actions/variables":                                `{"total_count":0,"variables":[]}`,
		"/agents/variables":                                 `{"total_count":0,"variables":[]}`,
	}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Host == "api.github.com" {
			if isIssueCreationPolicyFixtureRead(r) {
				body := issueCreationPolicyFixtureResponse
				if normalized {
					body = strings.Replace(body, "ALL", "FUTURE", 1)
				}
				fmt.Fprint(w, body)
				return
			}
			if servePRAccessFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
				return
			}
			if isMergeMethodsFixtureRead(r) {
				fmt.Fprint(w, mergeMethodsFixtureResponse)
				return
			}
			if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/repos/owner/current") {
				path := strings.TrimPrefix(r.URL.Path, "/repos/owner/current")
				if body, ok := responses[path]; ok {
					fmt.Fprint(w, body)
					return
				}
				if path == "/vulnerability-alerts" {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				if path == "/pages" {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprint(w, `{"message":"Not Found"}`)
					return
				}
			}
		}
		t.Error("unexpected export operation", r.Method, r.Host, r.URL)
		w.WriteHeader(http.StatusBadRequest)
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(fixture.close)
	return fixture
}
