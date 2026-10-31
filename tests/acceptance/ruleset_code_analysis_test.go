//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestRulesetCodeAnalysisExportPlanApplyRetainsAllProtections(t *testing.T) {
	binary := acceptanceBinary(t)
	data, err := os.ReadFile("../fixtures/ruleset-code-analysis.json")
	if err != nil {
		t.Fatal(err)
	}
	var state, baseline map[string]any
	if err = json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(data, &baseline)
	var mu sync.Mutex
	writes := 0
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
		if r.Method == "GET" {
			switch r.URL.Path {
			case "/repos/owner/current":
				fmt.Fprint(w, planResponse)
			case "/repos/owner/current/topics":
				fmt.Fprint(w, `{"names":[]}`)
			case "/repos/owner/current/labels":
				fmt.Fprint(w, `[]`)
			case "/repos/owner/current/rulesets":
				fmt.Fprint(w, `[{"id":26024001,"source_type":"Repository","source":"owner/current"}]`)
			case "/repos/owner/current/rulesets/26024001":
				json.NewEncoder(w).Encode(state)
			default:
				t.Error("unexpected GET", r.URL)
				w.WriteHeader(404)
			}
			return
		}
		if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/rulesets/26024001" {
			t.Error("unexpected mutation", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		writes++
		var payload, expected map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		copyData, _ := json.Marshal(baseline)
		json.Unmarshal(copyData, &expected)
		delete(expected, "id")
		delete(expected, "source")
		delete(expected, "source_type")
		acceptanceRuleParams(t, expected, "code_quality")["severity"] = "warnings"
		acceptanceRuleParams(t, expected, "code_scanning")["code_scanning_tools"].([]any)[0].(map[string]any)["security_alerts_threshold"] = "critical"
		for _, value := range []map[string]any{payload, expected} {
			slices.SortFunc(value["rules"].([]any), func(a, b any) int {
				return strings.Compare(a.(map[string]any)["type"].(string), b.(map[string]any)["type"].(string))
			})
		}
		if !reflect.DeepEqual(payload, expected) {
			t.Error("whole-object protection changed", payload, expected)
		}
		payload["id"], payload["source"], payload["source_type"] = float64(26024001), "owner/current", "Repository"
		state = payload
		json.NewEncoder(w).Encode(state)
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
		out, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if err != nil || out.runErr != nil {
			t.Fatalf("%v: %v %s", args, err, out.stderr.String())
		}
		return out.stdout.String()
	}
	run("export", "--rulesets")
	run("validate")
	path := filepath.Join(fixture.work, ".github/settings.yml")
	exported, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(exported, &doc); err != nil {
		t.Fatal(err)
	}
	actualPolicies := doc["repository"].(map[string]any)["settings"].(map[string]any)["rulesets"].([]any)
	policy := map[string]any{
		"id": 26024001, "ruleset_name": "default", "enforcement_status": "active",
		"target_branches": map[string]any{"include_default_branch": true, "include_all_branches": false, "include_by_pattern": []any{}, "exclude_by_pattern": []any{}},
		"bypass_list":     []any{},
		"rules": map[string]any{
			"restrict_deletions": true, "block_force_pushes": true, "require_linear_history": true, "require_signed_commits": true,
			"require_a_pull_request_before_merging": map[string]any{
				"required_approvals": 1, "allowed_merge_methods": []any{"squash"},
				"dismiss_stale_pull_request_approvals_when_new_commits_are_pushed":      true,
				"require_review_from_code_owners":                                       true,
				"require_an_additional_approval_for_unattributed_copilot_pull_requests": true,
				"require_approval_of_the_most_recent_reviewable_push":                   false,
				"require_conversation_resolution_before_merging":                        true,
			},
			"require_code_scanning_results": map[string]any{"code_scanning_tools": []any{map[string]any{"tool_name": "CodeQL", "security_alerts": "high_or_higher", "alerts": "errors"}}},
			"require_code_quality_results":  map[string]any{"severity": "errors"},
		},
	}
	policies := []any{policy}
	if !reflect.DeepEqual(actualPolicies, policies) {
		t.Fatalf("export=%v want=%v", actualPolicies, policies)
	}
	// Focus the independent UI declaration on Rulesets; ordinary scalar apply is separate.
	document := map[string]any{"version": "daiksud/gh-settings@v0", "repository": map[string]any{"settings": map[string]any{"rulesets": policies}}}
	focused, err := yaml.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsFile(fixture.work, string(focused)); err != nil {
		t.Fatal(err)
	}
	run("apply")
	mu.Lock()
	count := writes
	mu.Unlock()
	if count != 0 {
		t.Fatal("unchanged Ruleset export wrote", count)
	}
	policy["rules"].(map[string]any)["require_code_quality_results"].(map[string]any)["severity"] = "warnings_and_higher"
	policy["rules"].(map[string]any)["require_code_scanning_results"].(map[string]any)["code_scanning_tools"].([]any)[0].(map[string]any)["security_alerts"] = "critical"
	edited, err := yaml.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsFile(fixture.work, string(edited)); err != nil {
		t.Fatal(err)
	}
	if out := run("plan"); !strings.Contains(out, `"severity":"warnings_and_higher"`) || !strings.Contains(out, `"security_alerts":"critical"`) {
		t.Fatal("threshold change absent", out)
	}
	run("apply")
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	count = writes
	mu.Unlock()
	if count != 1 {
		t.Fatal("repeat apply wrote", count)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil || !reflect.DeepEqual(edited, unchanged) {
		t.Fatal("ID-selected update changed declaration", err)
	}
}
