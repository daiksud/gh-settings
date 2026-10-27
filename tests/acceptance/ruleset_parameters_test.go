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

	"go.yaml.in/yaml/v4"
)

const acceptanceReviewRuleset = `{"id":7,"name":"default","source_type":"Repository","source":"owner/current","current_user_can_bypass":"never","target":"branch","enforcement":"active","bypass_actors":[],"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"rules":[{"type":"deletion"},{"type":"non_fast_forward"},{"type":"required_linear_history"},{"type":"pull_request","parameters":{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":true,"require_code_owner_review":true,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":1,"required_review_thread_resolution":true,"dismissal_restriction":{"enabled":true,"allowed_actors":[{"type":"Team","id":21}]},"required_reviewers":[{"reviewer":{"type":"Team","id":22},"minimum_approvals":1,"file_patterns":["**","!docs/**","docs/README.md","**"]}]}},{"type":"required_status_checks","parameters":{"do_not_enforce_on_create":false,"required_status_checks":[{"context":"pass","integration_id":15368},{"context":"gh-settings","integration_id":15368}],"strict_required_status_checks_policy":true}}]}`

func acceptanceRuleParams(t *testing.T, policy map[string]any, kind string) map[string]any {
	t.Helper()
	for _, value := range policy["rules"].([]any) {
		rule := value.(map[string]any)
		if rule["type"] == kind {
			return rule["parameters"].(map[string]any)
		}
	}
	t.Fatal("missing rule", kind)
	return nil
}
func TestRulesetPRChecksExistingExportEditApply(t *testing.T) {
	binary := acceptanceBinary(t)
	var mu sync.Mutex
	var policy map[string]any
	json.Unmarshal([]byte(acceptanceReviewRuleset), &policy)
	var writes []map[string]any
	var methods []string
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if servePRAccessFixtureRead(w, r) {
			return
		}
		if serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			if r.Method != "GET" {
				t.Error("unexpected scalar mutation")
			}
			fmt.Fprint(w, planResponse)
			return
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
			return
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
			return
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
			t.Error("API contract version", r.Header)
		}
		if r.Method == "GET" {
			if r.URL.Query().Get("includes_parents") != "false" {
				t.Error("includes parents", r.URL)
			}
			switch r.URL.Path {
			case "/repos/owner/current/rulesets":
				fmt.Fprint(w, `[{"id":7,"name":"default","source_type":"Repository","source":"OWNER/CURRENT"},{"id":99,"name":"default","source_type":"Organization","source":"owner"}]`)
			case "/repos/owner/current/rulesets/7":
				json.NewEncoder(w).Encode(policy)
			default:
				t.Error("unexpected read", r.URL)
				w.WriteHeader(404)
			}
			return
		}
		if r.Method != "PUT" || r.URL.Path != "/repos/owner/current/rulesets/7" {
			t.Error("unexpected mutation", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
		var desired map[string]any
		if e := json.NewDecoder(r.Body).Decode(&desired); e != nil {
			t.Error(e)
		}
		if len(desired) != 6 || desired["id"] != nil || desired["current_user_can_bypass"] != nil {
			t.Error("metadata replay", desired)
		}
		writes = append(writes, desired)
		methods = append(methods, r.Method+" "+r.URL.Path)
		desired["id"] = float64(7)
		desired["source_type"] = "Repository"
		desired["source"] = "owner/current"
		desired["current_user_can_bypass"] = "never"
		policy = desired
		json.NewEncoder(w).Encode(policy)
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
		result, e := executeCLI(binary, fixture.work, fixture.root, env, args, false)
		if e != nil || result.runErr != nil {
			t.Fatalf("%v %v %s", args, e, result.stderr.String())
		}
		return result.stdout.String()
	}
	run("export", "--rulesets")
	data, e := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if e != nil {
		t.Fatal(e)
	}
	var exported map[string]any
	if e := yaml.Unmarshal(data, &exported); e != nil {
		t.Fatal(e)
	}
	run("validate")
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal("unchanged export differs", out)
	}
	members := exported["settings"].(map[string]any)["rulesets"].([]any)
	if len(members) != 1 {
		t.Fatal("inherited policy exported", members)
	}
	desired := members[0].(map[string]any)
	pr := acceptanceRuleParams(t, desired, "pull_request")
	reviewer := pr["required_reviewers"].([]any)[0].(map[string]any)
	reviewer["file_patterns"] = []any{"!docs/**", "**", "docs/README.md", "**"}
	document := map[string]any{"schemaVersion": 1, "settings": map[string]any{"rulesets": members}}
	save := func() {
		t.Helper()
		bytes, e := yaml.Marshal(document)
		if e != nil {
			t.Fatal(e)
		}
		if e := writeSettingsFile(fixture.work, string(bytes)); e != nil {
			t.Fatal(e)
		}
	}
	save()
	out := run("plan")
	if strings.Count(out, `update "default"`) != 1 || !strings.Contains(out, `"file_patterns":["!docs/**","**","docs/README.md","**"]`) {
		t.Fatal("order-only diff hidden", out)
	}
	reportPath := filepath.Join(fixture.root, "review-plan.json")
	run("action", "plan", reportPath)
	summaryPath := filepath.Join(fixture.root, "review-summary.md")
	summaryEnv := append(fixtureEnv(fixture.cfg), "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_TARGET=owner/current", "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	summary, e := executeCLI(binary, fixture.root, fixture.root, summaryEnv, []string{"action", "summary", reportPath, summaryPath}, false)
	if e != nil || summary.runErr != nil {
		t.Fatal(e, summary.stderr.String())
	}
	summaryData, _ := os.ReadFile(summaryPath)
	if !strings.Contains(html.UnescapeString(string(summaryData)), `"file_patterns":["!docs/**","**","docs/README.md","**"]`) {
		t.Fatal("summary lost order", string(summaryData))
	}
	run("apply")
	mu.Lock()
	firstCount := len(writes)
	firstPR := acceptanceRuleParams(t, writes[0], "pull_request")
	if firstCount != 1 || firstPR["required_approving_review_count"] != float64(1) || firstPR["require_extra_approval_for_unattributed_changes"] != false {
		t.Fatal("order edit changed siblings", writes)
	}
	mu.Unlock()
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	noWrites := len(writes) == 1
	mu.Unlock()
	if !noWrites {
		t.Fatal("order repeat apply wrote")
	}
	// Edit only selected attributes; retained rules and reviewer identity keep companions.
	partial := `schemaVersion: 1
settings:
  rulesets:
    - id: 7
      name: default
      enforcement: active
      conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}
      rules:
        - type: deletion
        - type: non_fast_forward
        - type: required_linear_history
        - type: pull_request
          parameters:
            required_approving_review_count: 0
            dismiss_stale_reviews_on_push: false
            require_code_owner_review: false
            required_reviewers: [{reviewer: {type: Team, id: 22}, minimum_approvals: 0, file_patterns: ['!docs/**', '**', 'docs/README.md', '**']}]
        - type: required_status_checks
          parameters:
            required_status_checks: [{context: pass}]
            strict_required_status_checks_policy: false
`
	if e := writeSettingsFile(fixture.work, partial); e != nil {
		t.Fatal(e)
	}
	run("apply")
	mu.Lock()
	last := writes[len(writes)-1]
	p := acceptanceRuleParams(t, last, "pull_request")
	checks := acceptanceRuleParams(t, last, "required_status_checks")
	if p["required_approving_review_count"] != float64(0) || p["dismiss_stale_reviews_on_push"] != false || p["require_code_owner_review"] != false || p["require_extra_approval_for_unattributed_changes"] != false || p["require_last_push_approval"] != false || p["required_review_thread_resolution"] != false || len(p["allowed_merge_methods"].([]any)) != 3 {
		t.Fatal("lost review values", p)
	}
	r := p["required_reviewers"].([]any)[0].(map[string]any)
	if r["minimum_approvals"] != float64(0) || !reflect.DeepEqual(r["file_patterns"], []any{"!docs/**", "**", "docs/README.md", "**"}) {
		t.Fatal("lost ordered reviewer attrs", r)
	}
	c := checks["required_status_checks"].([]any)
	if len(c) != 1 || c[0].(map[string]any)["integration_id"] != nil || checks["strict_required_status_checks_policy"] != false || checks["do_not_enforce_on_create"] != false {
		t.Fatal("check identity/companion lost", checks)
	}
	mu.Unlock()
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	run("apply")
	mu.Lock()
	count := len(writes)
	mu.Unlock()
	if count != 2 {
		t.Fatal("repeat parameter apply wrote", count)
	}
	// Empty reviewer/check memberships remove their members; absent scalar flags stay intact.
	clear := strings.Replace(partial, "required_reviewers: [{reviewer: {type: Team, id: 22}, minimum_approvals: 0, file_patterns: ['!docs/**', '**', 'docs/README.md', '**']}]", "required_reviewers: []", 1)
	clear = strings.Replace(clear, "required_status_checks: [{context: pass}]", "required_status_checks: []", 1)
	writeSettingsFile(fixture.work, clear)
	run("apply")
	mu.Lock()
	if len(acceptanceRuleParams(t, policy, "pull_request")["required_reviewers"].([]any)) != 0 || len(acceptanceRuleParams(t, policy, "required_status_checks")["required_status_checks"].([]any)) != 0 {
		t.Fatal("inner removal missing", policy)
	}
	mu.Unlock()
	if out := run("plan"); out != "No differences for managed repository settings.\n" {
		t.Fatal(out)
	}
	// Removing the rule type from declared membership removes that rule, not the Ruleset.
	writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings: {rulesets: [{id: 7, name: default, enforcement: active, conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}, rules: [{type: deletion}]}]}\n")
	run("apply")
	run("apply")
	mu.Lock()
	defer mu.Unlock()
	if len(writes) != 4 || len(policy["rules"].([]any)) != 1 || len(methods) != 4 {
		t.Fatal("rule removal/idempotence", writes, methods)
	}
}
func TestRulesetPRChecksFailuresBeforeWrites(t *testing.T) {
	binary := acceptanceBinary(t)
	for _, tt := range []struct {
		name, detail, desired string
		status                int
	}{{"unknown PR flag", strings.Replace(acceptanceReviewRuleset, `"require_code_owner_review":true`, `"require_code_owner_review":true,"future_flag":false`, 1), `[{id: 7, name: default, enforcement: active, conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}, rules: []}]`, 200}, {"null optional flag", strings.Replace(acceptanceReviewRuleset, `"require_extra_approval_for_unattributed_changes":false`, `"require_extra_approval_for_unattributed_changes":null`, 1), `[{id: 7, name: default, enforcement: active, conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}}]`, 200}, {"new reviewer incomplete", acceptanceReviewRuleset, `[{name: default, rules: [{type: pull_request, parameters: {required_reviewers: [{reviewer: {type: Team, id: 99}}]}}]}]`, 200}, {"write denied", acceptanceReviewRuleset, `[{id: 7, name: default, enforcement: active, conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}, rules: []}]`, 403}, {"write redirect", acceptanceReviewRuleset, `[{id: 7, name: default, enforcement: active, conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}, rules: []}]`, 307}} {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var methods []string
			fixture, e := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				methods = append(methods, r.Method)
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "GET" {
					if tt.status == 200 {
						t.Error("preflight failure wrote")
					}
					w.Header().Set("Location", "https://api.github.com/repos/other/wrong/rulesets/7")
					w.WriteHeader(tt.status)
					fmt.Fprint(w, `{"message":"denied"}`)
					return
				}
				if strings.HasSuffix(r.URL.Path, "/rulesets") {
					fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
				} else {
					fmt.Fprint(w, tt.detail)
				}
			}, false)
			if e != nil {
				t.Fatal(e)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if e := setupOrdinaryOrigin(fixture.work, env); e != nil {
				t.Fatal(e)
			}
			writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings: {rulesets: "+tt.desired+"}\n")
			result, e := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			if e != nil || result.runErr == nil || result.stdout.Len() != 0 {
				t.Fatal(e, result)
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"GET", "GET"}
			if tt.name == "new reviewer incomplete" {
				want = nil
			}
			if tt.status != 200 {
				want = append(want, "PUT")
			}
			if !reflect.DeepEqual(methods, want) {
				t.Fatal("unexpected API activity", methods, want)
			}
		})
	}
}

func TestRulesetPRResponseMismatchStopsLaterWrites(t *testing.T) {
	for _, removeAll := range []bool{false, true} {
		t.Run(fmt.Sprintf("remove-all-%t", removeAll), func(t *testing.T) {
			binary := acceptanceBinary(t)
			var mu sync.Mutex
			var methods []string
			fixture, e := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				methods = append(methods, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/rulesets"):
					fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":8,"source_type":"Repository","source":"owner/current"}]`)
				case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/7"):
					fmt.Fprint(w, acceptanceReviewRuleset)
				case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/8"):
					fmt.Fprint(w, strings.Replace(strings.Replace(acceptanceRuleset, `"id":7`, `"id":8`, 1), `"name":"basic"`, `"name":"old"`, 1))
				case r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "/7"):
					var policy map[string]any
					if e := json.NewDecoder(r.Body).Decode(&policy); e != nil {
						t.Error(e)
					}
					if removeAll {
						fmt.Fprint(w, acceptanceReviewRuleset)
						return
					}
					acceptanceRuleParams(t, policy, "pull_request")["require_extra_approval_for_unattributed_changes"] = true
					policy["id"], policy["source_type"], policy["source"] = 7, "Repository", "owner/current"
					json.NewEncoder(w).Encode(policy)
				default:
					t.Error("unexpected later operation", r.Method, r.URL)
					w.WriteHeader(400)
				}
			}, false)
			if e != nil {
				t.Fatal(e)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if e := setupOrdinaryOrigin(fixture.work, env); e != nil {
				t.Fatal(e)
			}
			writeSettingsFile(fixture.work, `schemaVersion: 1
settings:
  rulesets:
    - id: 7
      name: default
      enforcement: active
      conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}
      rules:
        - type: deletion
        - type: non_fast_forward
        - type: required_linear_history
        - type: pull_request
          parameters: {required_approving_review_count: 0}
        - type: required_status_checks
`)
			if removeAll {
				writeSettingsFile(fixture.work, "schemaVersion: 1\nsettings: {rulesets: [{id: 7, name: default, enforcement: active, conditions: {ref_name: {include: ['~DEFAULT_BRANCH']}}, rules: [{type: deletion}]}]}\n")
			}
			result, e := executeCLI(binary, fixture.work, fixture.root, env, []string{"apply"}, false)
			if e != nil || result.runErr == nil || result.stdout.Len() != 0 || !strings.Contains(result.stderr.String(), "writes may already have applied") {
				t.Fatal(e, result)
			}
			mu.Lock()
			defer mu.Unlock()
			want := []string{"GET /repos/owner/current/rulesets", "GET /repos/owner/current/rulesets/7", "GET /repos/owner/current/rulesets/8", "PUT /repos/owner/current/rulesets/7"}
			if !reflect.DeepEqual(methods, want) {
				t.Fatal("response mismatch did not stop later writes", methods)
			}

		})
	}
}
