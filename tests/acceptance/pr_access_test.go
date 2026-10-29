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
	"sync/atomic"
	"testing"
)

const prIdentityResponse = `{"data":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current"}}}`

func TestPRAccessExportPlanApplyAndNativeReports(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	enabled, policy := true, "ALL"
	var writes []map[string]any
	var operations []string
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			operations = append(operations, "read merge")
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		if isPRAccessFixtureRead(r) {
			operations = append(operations, "read PR policy")
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"repository": map[string]any{"hasPullRequestsEnabled": enabled, "pullRequestCreationPolicy": policy}}})
			return
		}
		if r.Method == "GET" {
			operations = append(operations, r.Method+" "+r.URL.Path)
			switch r.URL.Path {
			case "/repos/owner/current":
				fmt.Fprint(w, planResponse)
			case "/repos/owner/current/topics":
				fmt.Fprint(w, `{"names":[]}`)
			case "/repos/owner/current/labels":
				fmt.Fprint(w, `[]`)
			default:
				t.Error("unexpected read", r.URL)
			}
			return
		}
		if r.Method != "POST" || r.URL.Path != "/graphql" {
			t.Error("unexpected mutation", r.Method, r.URL)
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
			operations = append(operations, "lookup identity")
			if !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
				t.Error(request)
			}
			fmt.Fprint(w, prIdentityResponse)
			return
		}
		operations = append(operations, "mutate PR policy")
		input := request.Variables["input"].(map[string]any)
		if input["repositoryId"] != "opaque-id" {
			t.Error(input)
		}
		result := map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current"}
		for key, value := range input {
			switch key {
			case "repositoryId":
			case "hasPullRequestsEnabled":
				enabled = value.(bool)
				result[key] = enabled
			case "pullRequestCreationPolicy":
				policy = value.(string)
				result[key] = policy
			default:
				t.Error("undeclared input", key)
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
		result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || result.runErr != nil || result.stderr.Len() != 0 {
			t.Fatalf("%v %v %s", args, err, result.stderr.String())
		}
		return result.stdout.String()
	}
	run("export")
	run("validate")
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	exported, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil || !strings.Contains(string(exported), "pull_requests: true") || !strings.Contains(string(exported), "pull_request_permissions:\n                    creation_allowed_by: all_users\n") {
		t.Fatal(string(exported), err)
	}
	input := generalInput("features: {pull_requests: true, pull_request_permissions: {creation_allowed_by: collaborators_only}}\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	beforeValidate := fixture.reads.Load()
	run("validate")
	if fixture.reads.Load() != beforeValidate {
		t.Fatal("validate requested API")
	}
	wantPlan := "settings.repository.pull_request_creation_policy: \"ALL\" → \"COLLABORATORS_ONLY\"\n"
	if got := run("plan"); got != wantPlan {
		t.Fatal(got)
	}
	report := filepath.Join(fixture.root, "pr-plan.json")
	run("action", "plan", report)
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var typed map[string]any
	if err := json.Unmarshal(data, &typed); err != nil {
		t.Fatal(err)
	}
	wantReport := map[string]any{"outcome": "changed", "target": "owner/current", "managed": float64(2), "changes": []any{map[string]any{"field": "repository.pull_request_creation_policy", "before": "ALL", "after": "COLLABORATORS_ONLY"}}}
	if !reflect.DeepEqual(typed, wantReport) {
		t.Fatal(string(data))
	}
	summary := filepath.Join(fixture.root, "pr-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	result, err := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", report, summary}, false)
	if err != nil || result.runErr != nil {
		t.Fatal(err, result.stderr.String())
	}
	data, _ = os.ReadFile(summary)
	visible := html.UnescapeString(string(data))
	for _, line := range strings.Split(strings.TrimSpace(wantPlan), "\n") {
		if !strings.Contains(visible, line) {
			t.Fatal("missing native Summary", line, visible)
		}
	}
	run("apply")
	mu.Lock()
	if len(writes) != 1 || !reflect.DeepEqual(writes[0], map[string]any{"repositoryId": "opaque-id", "hasPullRequestsEnabled": true, "pullRequestCreationPolicy": "COLLABORATORS_ONLY"}) {
		t.Fatal(writes)
	}
	mu.Unlock()
	if got := run("plan"); got != "No differences for managed repository settings.\n" {
		t.Fatal(got)
	}
	// Scalars retain direct-apply semantics: matching declarations are still sent.
	run("apply")
	writeSettingsFile(fixture.work, generalInput("features: {pull_requests: false}\n"))
	if got := run("plan"); got != "repository.settings.general.features.pull_requests: true → false\n" {
		t.Fatal(got)
	}
	run("apply")
	writeSettingsFile(fixture.work, generalInput("features: {pull_request_permissions: {creation_allowed_by: all_users}}\n"))
	run("apply")
	mu.Lock()
	defer mu.Unlock()
	if len(writes) != 4 || !reflect.DeepEqual(writes[2], map[string]any{"repositoryId": "opaque-id", "hasPullRequestsEnabled": false}) || !reflect.DeepEqual(writes[3], map[string]any{"repositoryId": "opaque-id", "pullRequestCreationPolicy": "ALL"}) || enabled || policy != "ALL" {
		t.Fatal(writes, enabled, policy)
	}
	suffix := operations[len(operations)-7:]
	if !reflect.DeepEqual(suffix, []string{"lookup identity", "mutate PR policy", "read PR policy", "lookup identity", "mutate PR policy", "lookup identity", "mutate PR policy"}) {
		t.Fatal("direct apply compared current policy", suffix)
	}
}
func TestPRAccessFailuresStopWritesAndPreserveFiles(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, tt := range []struct {
		name, command, input, want string
		requests                   int
	}{
		{"partial read", "plan", "features: {issues: false, pull_requests: false}\n", "partial read denied", 2},
		{"null export", "export", "", "pullRequestCreationPolicy", 3},
		{"lookup wrong target", "apply", "features: {pull_requests: false}\n", "target does not match", 1},
		{"mutation ignored", "apply", "features: {pull_requests: false}\n", "did not retain", 2},
		{"mutation partial errors", "apply", "features: {pull_requests: false}\n", "writes may already have applied", 2},
		{"mutation redirect", "apply", "features: {pull_requests: false}\n", "307", 2},
		{"REST then lookup denied", "apply", "features: {issues: false, pull_requests: false}\n", "earlier repository writes remain applied", 2},
		{"REST then mutation denied", "apply", "features: {issues: false, pull_requests: false}\n", "earlier repository writes remain applied", 3},
		{"invalid local enum", "apply", "features: {pull_request_permissions: {creation_allowed_by: unknown}}\n", `unsupported general.features.pull_request_permissions.creation_allowed_by "unknown"`, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var methods []string
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				methods = append(methods, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if isPRAccessFixtureRead(r) {
					if tt.command == "export" {
						fmt.Fprint(w, `{"data":{"repository":{"hasPullRequestsEnabled":false,"pullRequestCreationPolicy":null}}}`)
					} else {
						fmt.Fprint(w, `{"data":{"repository":{"hasPullRequestsEnabled":false,"pullRequestCreationPolicy":"ALL"}},"errors":[{"message":"partial read denied"}]}`)
					}
					return
				}
				if r.Method == "GET" && r.URL.Path == "/repos/owner/current" {
					fmt.Fprint(w, planResponse)
					return
				}
				if r.Method == "PATCH" && r.URL.Path == "/repos/owner/current" {
					var patch map[string]any
					json.NewDecoder(r.Body).Decode(&patch)
					if !reflect.DeepEqual(patch, map[string]any{"has_issues": false}) {
						t.Error("excess REST fields", patch)
					}
					fmt.Fprint(w, `{}`)
					return
				}
				if r.Method == "POST" && r.URL.Path == "/graphql" {
					var request struct{ Query string }
					json.NewDecoder(r.Body).Decode(&request)
					if strings.HasPrefix(request.Query, "query") {
						if tt.name == "REST then lookup denied" {
							w.WriteHeader(403)
							fmt.Fprint(w, `{"message":"lookup denied"}`)
						} else if tt.name == "lookup wrong target" {
							fmt.Fprint(w, strings.Replace(prIdentityResponse, "owner/current", "other/repo", 1))
						} else {
							fmt.Fprint(w, prIdentityResponse)
						}
						return
					}
					switch tt.name {
					case "mutation redirect":
						w.Header().Set("Location", "https://api.github.com/unexpected")
						w.WriteHeader(307)
						fmt.Fprint(w, `{}`)
					case "REST then mutation denied":
						w.WriteHeader(403)
						fmt.Fprint(w, `{"message":"mutation denied"}`)
					case "mutation partial errors":
						fmt.Fprint(w, `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","hasPullRequestsEnabled":false}}},"errors":[{"message":"partial write denied"}]}`)
					default:
						fmt.Fprint(w, `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","hasPullRequestsEnabled":true}}}}`)
					}
					return
				}
				t.Error("unexpected later operation or redirect", r.Method, r.URL)
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
			original := repositoryInput("about: {topics: []}\nsettings:\n  general:\n" + indentInput(tt.input, 4))
			if tt.command == "export" {
				original = "previous file must survive incomplete PR policy\n"
			}
			writeSettingsFile(fixture.work, original)
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{tt.command}, false)
			if err != nil || result.runErr == nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), tt.want) {
				t.Fatal(err, result.stderr.String(), result.stdout.String())
			}
			if err := checkSettingsUnchanged(fixture.work, original); err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(methods) != tt.requests {
				t.Fatal("unexpected write/read or followed redirect", methods)
			}
		})
	}
}

func TestPRAccessPlanRejectsStaleOrigin(t *testing.T) {
	binary := acceptanceBinary(t)
	var calls atomic.Int32
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		var request struct{ Query string }
		if r.Method != "POST" || r.URL.Path != "/graphql" || json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("unexpected target request", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		if strings.Contains(request.Query, "followRenames: false") {
			fmt.Fprint(w, `{"data":{"repository":null},"errors":[{"message":"stale checkout origin"}]}`)
			return
		}
		fmt.Fprint(w, `{"data":{"repository":{"hasPullRequestsEnabled":false,"pullRequestCreationPolicy":"ALL"}}}`)
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := generalInput("features: {pull_requests: false}\n")
	writeSettingsFile(fixture.work, input)
	result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{"plan"}, false)
	if err != nil || result.runErr == nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "stale checkout origin") || calls.Load() != 1 {
		t.Fatal("followed stale origin", err, result.stdout.String(), result.stderr.String(), calls.Load())
	}
	if err := checkSettingsUnchanged(fixture.work, input); err != nil {
		t.Fatal(err)
	}
}
