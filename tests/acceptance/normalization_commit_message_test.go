//go:build acceptance

package acceptance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestCommitMessageNormalizationExportDiagnostic(t *testing.T) {
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":true,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"autoMergeAllowed":false,"allowUpdateBranch":false,"mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"BLANK","squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"BLANK"}}}`)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			fmt.Fprint(w, planResponse)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		default:
			t.Error("unexpected request", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"export"}, false)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic := result.stderr.String()
	wantDiagnostic := "Warning: repository.settings.general.pull_requests.squash_merge.default_commit_message: observed title \"COMMIT_OR_PR_TITLE\" and message \"BLANK\" are outside UI choices; using \"default_message\" (projection only).\n" +
		"Warning: repository.settings.general.pull_requests.merge_commit.default_commit_message: observed title \"MERGE_MESSAGE\" and message \"BLANK\" are outside UI choices; using \"default_message\" (projection only).\n"
	if result.runErr != nil || result.stdout.Len() != 0 || diagnostic != wantDiagnostic {
		t.Fatalf("export failed to report both raw pairs: err=%v stdout=%q stderr=%q want=%q", result.runErr, result.stdout.String(), diagnostic, wantDiagnostic)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || strings.Count(string(data), "default_commit_message: default_message") != 2 || strings.Contains(string(data), "BLANK") || strings.Contains(string(data), "MERGE_MESSAGE") || strings.Contains(string(data), "COMMIT_OR_PR_TITLE") {
		t.Fatal("export saved raw protocol values", string(data), err)
	}
	if err := fixture.checkRequests(false); err != nil {
		t.Fatal(err)
	}
}

func TestCommitMessageNormalizationAcrossExecutableCommands(t *testing.T) {
	for _, menu := range []string{"squash_merge", "merge_commit"} {
		t.Run(menu, func(t *testing.T) {
			binary := acceptanceBinary(t)
			titleField, messageField, prefix := "squashMergeCommitTitle", "squashMergeCommitMessage", "squash_merge_commit_"
			fallbackTitle, fallbackMessage := "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES"
			if menu == "merge_commit" {
				titleField, messageField, prefix = "mergeCommitTitle", "mergeCommitMessage", "merge_commit_"
				fallbackTitle, fallbackMessage = "MERGE_MESSAGE", "PR_TITLE"
			}
			var mu sync.Mutex
			state := map[string]any{"mergeCommitAllowed": true, "squashMergeAllowed": true, "rebaseMergeAllowed": false, "deleteBranchOnMerge": false, "autoMergeAllowed": false, "allowUpdateBranch": false, "mergeCommitTitle": "MERGE_MESSAGE", "mergeCommitMessage": "PR_TITLE", "squashMergeCommitTitle": "COMMIT_OR_PR_TITLE", "squashMergeCommitMessage": "COMMIT_MESSAGES"}
			state[titleField], state[messageField] = "FUTURE_TITLE", "FUTURE_MESSAGE"
			keepUnknown, template := true, false
			var requests []string
			var patches []map[string]any
			var wantPatch map[string]any
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch {
				case isMergeMethodsFixtureRead(r), isCommitMessageFixtureRead(r):
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": state}})
				case r.Method == "GET" && r.URL.Path == "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case r.Method == "PATCH" && r.URL.Path == "/repos/owner/current":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || !reflect.DeepEqual(payload, wantPatch) {
						t.Errorf("unexpected PATCH: %v want %v err=%v", payload, wantPatch, err)
						w.WriteHeader(400)
						return
					}
					patches = append(patches, payload)
					if value, present := payload["is_template"]; present {
						template = value.(bool)
					}
					if !keepUnknown {
						if value, present := payload[prefix+"title"]; present {
							state[titleField] = value
						}
						if value, present := payload[prefix+"message"]; present {
							state[messageField] = value
						}
					}
					fmt.Fprint(w, `{}`)
				default:
					t.Error("unexpected operation", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := append(exportFixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=pull_request", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			run := func(args ...string) (string, string, int) {
				t.Helper()
				mu.Lock()
				requests = nil
				mu.Unlock()
				result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
				if err != nil {
					t.Fatal(err)
				}
				status := 0
				if result.runErr != nil {
					var failure *exec.ExitError
					if !errors.As(result.runErr, &failure) {
						t.Fatal(result.runErr)
					}
					status = failure.ExitCode()
				}
				return result.stdout.String(), result.stderr.String(), status
			}
			checkRequests := func(want ...string) {
				t.Helper()
				mu.Lock()
				defer mu.Unlock()
				if !reflect.DeepEqual(requests, want) {
					t.Fatalf("requests=%v want=%v", requests, want)
				}
			}
			intent := func(body string) string {
				t.Helper()
				input := generalInput(body)
				if err := writeSettingsFile(fixture.work, input); err != nil {
					t.Fatal(err)
				}
				return input
			}
			path := "repository.settings.general.pull_requests." + menu + ".default_commit_message"
			var input string
			for _, choice := range []string{"default_message", "pull_request_title"} {
				input = intent("pull_requests: {" + menu + ": {default_commit_message: " + choice + "}}\n")
				output, diagnostic, status := run("plan")
				if status != 0 || diagnostic != "" || strings.Count(output, "Warning:") != 1 || !strings.Contains(output, path) || !strings.Contains(output, `"FUTURE_TITLE"`) || !strings.Contains(output, `"FUTURE_MESSAGE"`) || !strings.Contains(output, `"`+choice+`" (write required)`) || strings.Contains(output, "No differences") {
					t.Fatal("projected pair became exact comparison", status, output, diagnostic)
				}
				checkRequests("POST /graphql")
				if err := checkSettingsUnchanged(fixture.work, input); err != nil {
					t.Fatal(err)
				}
			}
			report, summary := filepath.Join(fixture.root, "menu.json"), filepath.Join(fixture.root, "menu-summary.md")
			output, diagnostic, status := run("action", "plan", report)
			if status != 0 || output != "" || diagnostic != "" {
				t.Fatal("private planning disclosed values", status, output, diagnostic)
			}
			checkRequests("POST /graphql")
			data, err := os.ReadFile(report)
			var saved struct {
				Outcome string
				Managed int
				Changes []any
			}
			// Raw member names belong to the private, same-version report schema.
			var raw map[string]json.RawMessage
			if err != nil || json.Unmarshal(data, &raw) != nil || json.Unmarshal(raw["outcome"], &saved.Outcome) != nil || json.Unmarshal(raw["managed"], &saved.Managed) != nil || json.Unmarshal(raw["changes"], &saved.Changes) != nil {
				t.Fatal("invalid private report", string(data), err)
			}
			var decisions []struct {
				Menu, After   string
				Normalization struct {
					Path       string
					RawTitle   string `json:"raw_title"`
					RawMessage string `json:"raw_message"`
					Fallback   string
				}
			}
			if json.Unmarshal(raw["commit_messages"], &decisions) != nil || saved.Outcome != "changed" || saved.Managed != 2 || len(saved.Changes) != 0 || len(decisions) != 1 || decisions[0].Menu != menu || decisions[0].After != "pull_request_title" || decisions[0].Normalization.Path != path || decisions[0].Normalization.RawTitle != "FUTURE_TITLE" || decisions[0].Normalization.RawMessage != "FUTURE_MESSAGE" || decisions[0].Normalization.Fallback != "default_message" {
				t.Fatal("private report lost paired ownership", string(data))
			}
			output, diagnostic, status = run("action", "summary", report, summary)
			data, err = os.ReadFile(summary)
			if err != nil || status != 0 || diagnostic != "" || output != "" || !strings.Contains(string(data), path) || !strings.Contains(string(data), "FUTURE_TITLE") || !strings.Contains(string(data), "FUTURE_MESSAGE") || !strings.Contains(string(data), "write required") {
				t.Fatal("detailed Summary disagrees", string(data), status, output, diagnostic, err)
			}
			summary = filepath.Join(fixture.root, "status-summary.md")
			output, diagnostic, status = run("action", "summary", report, summary, "status-only")
			data, err = os.ReadFile(summary)
			if err != nil || status != 0 || diagnostic != "" || output != "plan-outcome=changed\n" || strings.Contains(string(data), "FUTURE") || strings.Contains(string(data), "default_commit_message") {
				t.Fatal("status-only leaked menu", string(data), status, output, diagnostic, err)
			}
			mu.Lock()
			noWrites := len(patches) == 0
			mu.Unlock()
			if !noWrites {
				t.Fatal("planning wrote normalized values")
			}
			input = intent("features: {issues: false}\n")
			output, diagnostic, status = run("plan")
			if status != 0 || diagnostic != "" || strings.Contains(output, "Warning:") || strings.Contains(output, "default_commit_message") {
				t.Fatal("unmanaged menu displayed", status, output, diagnostic)
			}
			checkRequests("GET /repos/owner/current")
			mu.Lock()
			wantPatch = map[string]any{"has_issues": false}
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 0 || diagnostic != "" || output != "Applied repository settings.\n" {
				t.Fatal("scalar-only apply acquired menu", status, output, diagnostic)
			}
			checkRequests("PATCH /repos/owner/current")
			mu.Lock()
			preserved := state[titleField] == "FUTURE_TITLE" && state[messageField] == "FUTURE_MESSAGE"
			mu.Unlock()
			if !preserved {
				t.Fatal("unmanaged raw menu changed")
			}
			input = intent("template_repository: true\npull_requests: {" + menu + ": {default_commit_message: default_message}}\n")
			mu.Lock()
			wantPatch = map[string]any{"is_template": true, prefix + "title": fallbackTitle, prefix + "message": fallbackMessage}
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Count(diagnostic, "Warning:") != 2 || !strings.Contains(diagnostic, "may already have applied") {
				t.Fatal("unknown readback claimed success", status, output, diagnostic)
			}
			checkRequests("POST /graphql", "PATCH /repos/owner/current", "POST /graphql")
			mu.Lock()
			retained := template && state[titleField] == "FUTURE_TITLE" && state[messageField] == "FUTURE_MESSAGE"
			mu.Unlock()
			if !retained {
				t.Fatal("failed verification implied rollback")
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			input = intent("pull_requests: {" + menu + ": {default_commit_message: pull_request_title}}\n")
			mu.Lock()
			keepUnknown = false
			wantPatch = map[string]any{prefix + "title": "PR_TITLE", prefix + "message": "BLANK"}
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 0 || output != "Applied repository settings.\n" || strings.Count(diagnostic, "Warning:") != 1 {
				t.Fatal("canonical readback failed", status, output, diagnostic)
			}
			checkRequests("POST /graphql", "PATCH /repos/owner/current", "POST /graphql")
			output, diagnostic, status = run("plan")
			if status != 0 || diagnostic != "" || output != "No differences for managed repository settings.\n" {
				t.Fatal("canonical observation retained fallback", status, output, diagnostic)
			}
			checkRequests("POST /graphql")
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			state[titleField] = nil
			before := len(patches)
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") || strings.Contains(diagnostic, "may already have applied") {
				t.Fatal("missing preflight became normalization or write uncertainty", status, output, diagnostic)
			}
			checkRequests("POST /graphql")
			mu.Lock()
			after := len(patches)
			mu.Unlock()
			if before != after {
				t.Fatal("failed acquisition wrote a menu")
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}
