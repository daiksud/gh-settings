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

	"go.yaml.in/yaml/v4"
)

func TestCreationPolicyNormalizationExportVisibility(t *testing.T) {
	prWarning := `Warning: repository.settings.general.features.pull_request_permissions.creation_allowed_by: observed "FUTURE_PR" is outside UI choices; using "all_users" (projection only).` + "\n"
	issueWarning := `Warning: repository.settings.general.features.issue_permissions.creation_allowed_by: observed "FUTURE_ISSUE" is outside UI choices; using "all_users" (projection only).` + "\n"
	for _, test := range []struct {
		name, prRaw, issueRaw             string
		prs, issues                       bool
		diagnostic, prChoice, issueChoice string
	}{
		{"visible", "FUTURE_PR", "FUTURE_ISSUE", true, true, prWarning + issueWarning, "all_users", "all_users"},
		{"PR hidden", "FUTURE_PR", "FUTURE_ISSUE", false, true, issueWarning, "", "all_users"},
		{"Issue hidden", "FUTURE_PR", "FUTURE_ISSUE", true, false, prWarning, "all_users", ""},
		{"hidden", "FUTURE_PR", "FUTURE_ISSUE", false, false, "", "", ""},
		{"canonical", "ALL", "COLLABORATORS_ONLY", true, true, "", "all_users", "collaborators_only"},
		{"empty whitespace", " \t\n ", "", true, true, `Warning: repository.settings.general.features.pull_request_permissions.creation_allowed_by: observed " \t\n " is outside UI choices; using "all_users" (projection only).` + "\n" + `Warning: repository.settings.general.features.issue_permissions.creation_allowed_by: observed "" is outside UI choices; using "all_users" (projection only).` + "\n", "all_users", "all_users"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if isPRAccessFixtureRead(r) {
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": map[string]any{"hasPullRequestsEnabled": test.prs, "pullRequestCreationPolicy": test.prRaw}}})
					return
				}
				if isIssueCreationPolicyFixtureRead(r) {
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": map[string]any{"issueCreationPolicy": test.issueRaw}}})
					return
				}
				if serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					body := strings.Replace(planResponse, `"has_issues":true`, fmt.Sprintf(`"has_issues":%t`, test.issues), 1)
					fmt.Fprint(w, body)
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
			if err != nil || result.runErr != nil || result.stdout.Len() != 0 || result.stderr.String() != test.diagnostic {
				t.Fatal("export warning scope/provenance failed", err, result.runErr, result.stdout.String(), result.stderr.String(), test.diagnostic)
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			var document map[string]any
			if err != nil || yaml.Unmarshal(data, &document) != nil || strings.Contains(string(data), "FUTURE") || strings.Contains(string(data), "raw_value") {
				t.Fatal("raw provenance leaked into YAML", string(data), err)
			}
			features := document["repository"].(map[string]any)["settings"].(map[string]any)["general"].(map[string]any)["features"].(map[string]any)
			for _, control := range []struct{ key, choice string }{{"pull_request_permissions", test.prChoice}, {"issue_permissions", test.issueChoice}} {
				value, present := features[control.key]
				if control.choice == "" {
					if present {
						t.Fatal("hidden policy exported", control.key, value)
					}
					continue
				}
				if !present || value.(map[string]any)["creation_allowed_by"] != control.choice {
					t.Fatal("effective UI choice lost", control, features)
				}
			}
			if err := fixture.checkRequests(false); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreationPolicyNormalizationAcrossExecutableCommands(t *testing.T) {
	for _, test := range []struct{ control, graphqlField, nativeField string }{
		{"issue_permissions", "issueCreationPolicy", "repository.issue_creation_policy"},
		{"pull_request_permissions", "pullRequestCreationPolicy", "repository.pull_request_creation_policy"},
	} {
		t.Run(test.control, func(t *testing.T) {
			binary := acceptanceBinary(t)
			var mu sync.Mutex
			var rawPolicy any = "FUTURE_POLICY"
			keepUnknown, projects := true, true
			var operations []string
			var writes []map[string]any
			var wantMutation map[string]any
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "GET" && r.URL.Path == "/repos/owner/current" {
					operations = append(operations, "REST read")
					fmt.Fprint(w, strings.Replace(planResponse, `"has_projects":true`, fmt.Sprintf(`"has_projects":%t`, projects), 1))
					return
				}
				if r.Method == "PATCH" && r.URL.Path == "/repos/owner/current" {
					operations = append(operations, "REST write")
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || !reflect.DeepEqual(payload, map[string]any{"has_projects": false}) {
						t.Error("undeclared REST input", payload, err)
						w.WriteHeader(400)
						return
					}
					projects = false
					fmt.Fprint(w, `{}`)
					return
				}
				if r.Method != "POST" || r.URL.Path != "/graphql" {
					t.Error("unexpected operation", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				publicPR, publicIssue := isPRAccessFixtureRead(r), isIssueCreationPolicyFixtureRead(r)
				var request struct {
					Query     string
					Variables map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				if strings.Contains(request.Query, "mutation") {
					operations = append(operations, "mutation")
					input, ok := request.Variables["input"].(map[string]any)
					if !ok || !reflect.DeepEqual(input, wantMutation) {
						t.Error("undeclared mutation", input, wantMutation)
						w.WriteHeader(400)
						return
					}
					writes = append(writes, input)
					if !keepUnknown {
						rawPolicy = input[test.graphqlField]
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"updateRepository": map[string]any{"repository": map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current", test.graphqlField: input[test.graphqlField]}}}})
					return
				}
				operations = append(operations, "query")
				if !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
					t.Error("wrong acquisition target", request.Variables)
					w.WriteHeader(400)
					return
				}
				fields := map[string]any{test.graphqlField: rawPolicy}
				if publicPR {
					if test.control != "pull_request_permissions" {
						t.Error("unmanaged PR read")
						w.WriteHeader(400)
						return
					}
					fields["hasPullRequestsEnabled"] = true
				} else if publicIssue {
					if test.control != "issue_permissions" {
						t.Error("unmanaged Issue read")
						w.WriteHeader(400)
						return
					}
				} else {
					other := "issueCreationPolicy"
					if test.control == "issue_permissions" {
						other = "pullRequestCreationPolicy"
					}
					if !strings.Contains(request.Query, "id nameWithOwner") || !strings.Contains(request.Query, test.graphqlField) || strings.Contains(request.Query, other) {
						t.Error("unmanaged prerequisite read", request.Query)
						w.WriteHeader(400)
						return
					}
					fields["id"], fields["nameWithOwner"] = "opaque-id", "owner/current"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": fields}})
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
				operations = nil
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
			checkOperations := func(want ...string) {
				t.Helper()
				mu.Lock()
				defer mu.Unlock()
				if !reflect.DeepEqual(operations, want) {
					t.Fatal("unexpected operation scope/order", operations, want)
				}
			}
			intent := func(features string) string {
				t.Helper()
				input := generalInput("features: " + features + "\n")
				if err := writeSettingsFile(fixture.work, input); err != nil {
					t.Fatal(err)
				}
				return input
			}
			path := "repository.settings.general.features." + test.control + ".creation_allowed_by"
			var input string
			for _, choice := range []string{"all_users", "collaborators_only"} {
				input = intent("{" + test.control + ": {creation_allowed_by: " + choice + "}}")
				output, diagnostic, status := run("plan")
				if status != 0 || diagnostic != "" || strings.Count(output, "Warning:") != 1 || !strings.Contains(output, path) || !strings.Contains(output, `"FUTURE_POLICY"`) || !strings.Contains(output, `"all_users" (normalized observation) → "`+choice+`" (write required)`) || strings.Contains(output, "No differences") || strings.Contains(output, "settings."+test.nativeField) {
					t.Fatal("fallback matched as exact native state", status, output, diagnostic)
				}
				checkOperations("query")
				if err := checkSettingsUnchanged(fixture.work, input); err != nil {
					t.Fatal(err)
				}
			}
			report, summary := filepath.Join(fixture.root, "policy.json"), filepath.Join(fixture.root, "policy-summary.md")
			output, diagnostic, status := run("action", "plan", report)
			if status != 0 || output != "" || diagnostic != "" {
				t.Fatal("private plan disclosed policy", status, output, diagnostic)
			}
			checkOperations("query")
			data, err := os.ReadFile(report)
			var saved struct {
				Outcome string
				Managed int
				Changes []struct {
					Field, Before, After string
					Normalization        struct {
						Path     string
						Raw      string `json:"raw_value"`
						Fallback string
					}
				}
			}
			if err != nil || json.Unmarshal(data, &saved) != nil || saved.Outcome != "changed" || saved.Managed != 1 || len(saved.Changes) != 1 || saved.Changes[0].Field != test.nativeField || saved.Changes[0].Before != "ALL" || saved.Changes[0].After != "COLLABORATORS_ONLY" || saved.Changes[0].Normalization.Path != path || saved.Changes[0].Normalization.Raw != "FUTURE_POLICY" || saved.Changes[0].Normalization.Fallback != "all_users" {
				t.Fatal("private report lost provenance/ownership", string(data), err)
			}
			output, diagnostic, status = run("action", "summary", report, summary)
			data, err = os.ReadFile(summary)
			if err != nil || status != 0 || output != "" || diagnostic != "" || !strings.Contains(string(data), path) || !strings.Contains(string(data), "FUTURE_POLICY") || !strings.Contains(string(data), "collaborators_only") || !strings.Contains(string(data), "write required") {
				t.Fatal("detailed Summary disagreed", string(data), status, output, diagnostic, err)
			}
			summary = filepath.Join(fixture.root, "status-summary.md")
			output, diagnostic, status = run("action", "summary", report, summary, "status-only")
			data, err = os.ReadFile(summary)
			if err != nil || status != 0 || output != "plan-outcome=changed\n" || diagnostic != "" || strings.Contains(string(data), "FUTURE_POLICY") || strings.Contains(string(data), "creation_allowed_by") || strings.Contains(string(data), "all_users") || strings.Contains(string(data), "collaborators_only") {
				t.Fatal("status-only disclosed policy", string(data), status, output, diagnostic, err)
			}
			mu.Lock()
			noWrites := len(writes) == 0
			mu.Unlock()
			if !noWrites {
				t.Fatal("planning changed policy")
			}
			input = intent("{projects: false}")
			output, diagnostic, status = run("plan")
			if status != 0 || diagnostic != "" || strings.Contains(output, "Warning:") || strings.Contains(output, "creation_allowed_by") {
				t.Fatal("unmanaged policy displayed", status, output, diagnostic)
			}
			checkOperations("REST read")
			output, diagnostic, status = run("apply")
			if status != 0 || output != "Applied repository settings.\n" || diagnostic != "" {
				t.Fatal("scalar operation acquired policy", status, output, diagnostic)
			}
			checkOperations("REST write")
			mu.Lock()
			preserved := rawPolicy == "FUTURE_POLICY" && len(writes) == 0
			projects = true
			wantMutation = map[string]any{"repositoryId": "opaque-id", test.graphqlField: "ALL"}
			mu.Unlock()
			if !preserved {
				t.Fatal("omitted policy changed")
			}
			input = intent("{projects: false, " + test.control + ": {creation_allowed_by: all_users}}")
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Count(diagnostic, "Warning:") != 2 || !strings.Contains(diagnostic, "may already have applied") || !strings.Contains(diagnostic, "earlier repository writes remain applied") {
				t.Fatal("raw failure hid write effects", status, output, diagnostic)
			}
			checkOperations("REST write", "query", "mutation", "query")
			mu.Lock()
			retained := !projects && rawPolicy == "FUTURE_POLICY" && len(writes) == 1
			mu.Unlock()
			if !retained {
				t.Fatal("failed verification implied rollback")
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			input = intent("{" + test.control + ": {creation_allowed_by: collaborators_only}}")
			mu.Lock()
			keepUnknown = false
			wantMutation = map[string]any{"repositoryId": "opaque-id", test.graphqlField: "COLLABORATORS_ONLY"}
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 0 || output != "Applied repository settings.\n" || strings.Count(diagnostic, "Warning:") != 1 {
				t.Fatal("canonical fresh readback failed", status, output, diagnostic)
			}
			checkOperations("query", "mutation", "query")
			output, diagnostic, status = run("plan")
			if status != 0 || output != "No differences for managed repository settings.\n" || diagnostic != "" {
				t.Fatal("canonical observation retained fallback", status, output, diagnostic)
			}
			checkOperations("query")
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			rawPolicy = nil
			before := len(writes)
			mu.Unlock()
			output, diagnostic, status = run("apply")
			if status != 1 || output != "" || strings.Contains(diagnostic, "Warning:") || strings.Contains(diagnostic, "may already have applied") || !strings.Contains(diagnostic, "missing/null or invalid") {
				t.Fatal("failed preflight became default/write uncertainty", status, output, diagnostic)
			}
			checkOperations("query")
			mu.Lock()
			after := len(writes)
			mu.Unlock()
			if before != after {
				t.Fatal("failed preflight mutated policy")
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
		})
	}
}
