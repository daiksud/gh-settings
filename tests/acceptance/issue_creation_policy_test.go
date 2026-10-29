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

func TestIssuePolicyDisabledExportAndSharedApply(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	policy := "ALL"
	issues := false
	var writes []map[string]any
	var operations []string
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if isIssueCreationPolicyFixtureRead(r) {
			operations = append(operations, "read Issue policy")
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": map[string]any{"issueCreationPolicy": policy}}})
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if servePRAccessFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if r.Method == "GET" {
			switch r.URL.Path {
			case "/repos/owner/current":
				fmt.Fprint(w, strings.Replace(planResponse, `"has_issues":true`, `"has_issues":false`, 1))
			case "/repos/owner/current/topics":
				fmt.Fprint(w, `{"names":[]}`)
			case "/repos/owner/current/labels":
				fmt.Fprint(w, `[]`)
			default:
				t.Error("unexpected read", r.URL)
			}
			return
		}
		if r.Method == "PATCH" && r.URL.Path == "/repos/owner/current" {
			operations = append(operations, "REST write")
			var input map[string]any
			json.NewDecoder(r.Body).Decode(&input)
			if !reflect.DeepEqual(input, map[string]any{"has_projects": false}) {
				t.Error("extra REST values", input)
			}
			// This unrelated feature write must not enable Issues.
			fmt.Fprint(w, `{}`)
			return
		}
		if r.Method != "POST" || r.URL.Path != "/graphql" {
			t.Error("unexpected later write", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		var request struct {
			Query     string
			Variables map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if strings.HasPrefix(request.Query, "query") {
			operations = append(operations, "lookup")
			fields := map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current"}
			if strings.Contains(request.Query, "issueCreationPolicy") {
				fields["issueCreationPolicy"] = policy
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": fields}})
			return
		}
		operations = append(operations, "mutation")
		input := request.Variables["input"].(map[string]any)
		result := map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current"}
		for key, value := range input {
			switch key {
			case "repositoryId":
				if value != "opaque-id" {
					t.Error(input)
				}
			case "issueCreationPolicy":
				policy = value.(string)
				result[key] = policy
			case "hasPullRequestsEnabled":
				if value != false {
					t.Error(input)
				}
				result[key] = false
			default:
				t.Error("undeclared companion", key)
			}
		}
		writes = append(writes, input)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"updateRepository": map[string]any{"repository": result}}})
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
	run("export")
	run("validate")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(data), "issues: false") || strings.Contains(string(data), "issue_permissions:") {
		t.Fatal(string(data), err)
	}
	input := generalInput("features: {issue_permissions: {creation_allowed_by: collaborators_only}}\n")
	writeSettingsFile(fixture.work, input)
	run("validate")
	want := `settings.repository.issue_creation_policy: "ALL" → "COLLABORATORS_ONLY"` + "\n"
	if got := run("plan"); got != want {
		t.Fatal(got)
	}
	report := filepath.Join(fixture.root, "issue-plan.json")
	run("action", "plan", report)
	data, _ = os.ReadFile(report)
	var typed map[string]any
	json.Unmarshal(data, &typed)
	expected := map[string]any{"outcome": "changed", "target": "owner/current", "managed": float64(1), "changes": []any{map[string]any{"field": "repository.issue_creation_policy", "before": "ALL", "after": "COLLABORATORS_ONLY"}}}
	if !reflect.DeepEqual(typed, expected) {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "issue-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	reads := fixture.reads.Load()
	r, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || r.runErr != nil || reads != fixture.reads.Load() {
		t.Fatal(err, r.stderr.String())
	}
	data, _ = os.ReadFile(summary)
	if !strings.Contains(html.UnescapeString(string(data)), strings.TrimSpace(want)) {
		t.Fatal(string(data))
	}
	run("apply")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	run("apply")
	writeSettingsFile(fixture.work, generalInput("features: {issue_permissions: {creation_allowed_by: all_users}}\n"))
	run("apply")
	writeSettingsFile(fixture.work, generalInput("features: {pull_requests: false, issue_permissions: {creation_allowed_by: collaborators_only}}\n"))
	run("apply")
	mixed := generalInput("features: {projects: false, issue_permissions: {creation_allowed_by: all_users}}\n")
	writeSettingsFile(fixture.work, mixed)
	run("apply")
	if err := checkSettingsUnchanged(fixture.work, mixed); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if issues || policy != "ALL" || len(writes) != 5 {
		t.Fatal("availability changed or extra mutation", issues, policy, writes)
	}
	for i, input := range writes {
		want := map[string]any{"repositoryId": "opaque-id", "issueCreationPolicy": "COLLABORATORS_ONLY"}
		if i == 2 || i == 4 {
			want["issueCreationPolicy"] = "ALL"
		}
		if i == 3 {
			want["hasPullRequestsEnabled"] = false
		}
		if !reflect.DeepEqual(input, want) {
			t.Fatal("over-broad mutation", i, input, want)
		}
	}
	if !reflect.DeepEqual(operations[len(operations)-4:], []string{"REST write", "lookup", "mutation", "lookup"}) {
		t.Fatal(operations)
	}
}
func TestIssuePolicyFailuresPreserveStateAndStopLaterResources(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, tt := range []struct {
		name, command, input, want string
		requests                   int
	}{
		{"partial read", "plan", "features: {projects: false, issue_permissions: {creation_allowed_by: all_users}}\n", "Issue policy hidden", 2},
		{"malformed export", "export", "", "invalid issueCreationPolicy policy string", 4},
		{"ignored mutation", "apply", "features: {issue_permissions: {creation_allowed_by: collaborators_only}}\n", "did not retain requested issueCreationPolicy", 2},
		{"partial mutation", "apply", "features: {issue_permissions: {creation_allowed_by: all_users}}\n", "partial Issue mutation denied", 2},
		{"redirect", "apply", "features: {issue_permissions: {creation_allowed_by: all_users}}\n", "307", 2},
		{"mixed write denied", "apply", "features: {projects: false, issue_permissions: {creation_allowed_by: all_users}}\n", "earlier repository writes remain applied", 3},
		{"invalid local", "apply", "features: {issue_permissions: {creation_allowed_by: unknown}}\n", `unsupported general.features.issue_permissions.creation_allowed_by "unknown"`, 0},
		{"stale origin", "plan", "features: {issue_permissions: {creation_allowed_by: all_users}}\n", "stale checkout origin", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var methods []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				methods = append(methods, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if isIssueCreationPolicyFixtureRead(r) {
					switch tt.name {
					case "malformed export":
						fmt.Fprint(w, `{"data":{"repository":{"issueCreationPolicy":false}}}`)
					case "stale origin":
						fmt.Fprint(w, `{"data":{"repository":null},"errors":[{"message":"stale checkout origin"}]}`)
					default:
						fmt.Fprint(w, `{"data":{"repository":{"issueCreationPolicy":"ALL"}},"errors":[{"message":"Issue policy hidden"}]}`)
					}
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if servePRAccessFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if r.Method == "GET" && r.URL.Path == "/repos/owner/current" {
					fmt.Fprint(w, planResponse)
					return
				}
				if r.Method == "PATCH" && r.URL.Path == "/repos/owner/current" {
					var patch map[string]any
					json.NewDecoder(r.Body).Decode(&patch)
					if !reflect.DeepEqual(patch, map[string]any{"has_projects": false}) {
						t.Error(patch)
					}
					fmt.Fprint(w, `{}`)
					return
				}
				if r.Method == "POST" && r.URL.Path == "/graphql" {
					var request struct{ Query string }
					json.NewDecoder(r.Body).Decode(&request)
					if strings.HasPrefix(request.Query, "query") {
						fmt.Fprint(w, `{"data":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","issueCreationPolicy":"ALL"}}}`)
						return
					}
					switch tt.name {
					case "redirect":
						w.Header().Set("Location", "https://api.github.com/unexpected")
						w.WriteHeader(307)
						fmt.Fprint(w, `{}`)
					case "mixed write denied":
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"Issue mutation denied"}`)
					case "partial mutation":
						fmt.Fprint(w, `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","issueCreationPolicy":"ALL"}}},"errors":[{"message":"partial Issue mutation denied"}]}`)
					default:
						fmt.Fprint(w, `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","issueCreationPolicy":"ALL"}}}}`)
					}
					return
				}
				t.Error("unexpected later resource or followed redirect", r.Method, r.URL)
				w.WriteHeader(400)
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			input := repositoryInput("about: {topics: []}\nsettings:\n  general:\n" + indentInput(tt.input, 4))
			if tt.command == "export" {
				input = "previous configuration must survive incomplete Issue policy\n"
			}
			writeSettingsFile(fixture.work, input)
			r, err := executeCLI(binary, fixture.work, fixture.root, env, []string{tt.command}, false)
			if err != nil || r.runErr == nil || r.stdout.Len() != 0 || !strings.Contains(r.stderr.String(), tt.want) {
				t.Fatal(err, r.stderr.String(), r.stdout.String())
			}
			if err := checkSettingsUnchanged(fixture.work, input); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(methods) != tt.requests {
				t.Fatal(methods)
			}
		})
	}
}
