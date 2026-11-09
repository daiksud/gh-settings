//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRemovedRulesetDeclarationsFailBeforeRemoteWork(t *testing.T) {
	binary := acceptanceBinary(t)
	var calls atomic.Int32
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := exportFixtureEnv(fixture.cfg)
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	for _, control := range []string{
		"enforcement_status: active\n        target_pushes: false\n        rules: {}",
		"enforcement_status: active\n        target_pushes: true\n        rules: {}",
		"enforcement_status: active\n        rules: {restrict_file_paths: {}}",
		"enforcement_status: active\n        rules: {restrict_file_path_length: {}}",
		"enforcement_status: active\n        rules: {restrict_file_extensions: {}}",
		"enforcement_status: active\n        rules: {restrict_file_size: {}}",
		"enforcement_status: active\n        rules: {require_code_quality_results: {}}",
		"enforcement_status: active\n        rules: {restrict_code_coverage: {minimum_line_coverage_percentage: 0, maximum_line_coverage_drop: 0}}",
		"enforcement_status: active\n        rules: {require_a_pull_request_before_merging: {require_review_from_specific_teams: []}}",
		"enforcement_status: active\n        rules: {require_a_pull_request_before_merging: {require_an_additional_approval_for_unattributed_copilot_pull_requests: false}}",
		"enforcement_status: active\n        rules: {require_a_pull_request_before_merging: {require_an_additional_approval_for_unattributed_copilot_pull_requests: true}}",
		"enforcement_status: evaluate\n        rules: {}",
		"enforcement_status: active\n        rules: {require_merge_queue: {}}",
		"enforcement_status: active\n        rules: {restrict_commit_message: {requirement: must_contain_matching_pattern, pattern: issue}}",
		"enforcement_status: active\n        rules: {restrict_commit_author_email: {requirement: must_contain_matching_pattern, pattern: example}}",
		"enforcement_status: active\n        rules: {restrict_committer_email: {requirement: must_contain_matching_pattern, pattern: example}}",
		"enforcement_status: active\n        rules: {restrict_branch_names: {requirement: must_contain_matching_pattern, pattern: main}}",
		"enforcement_status: active\n        rules: {restrict_tag_names: {requirement: must_contain_matching_pattern, pattern: v}}",
		"enforcement_status: active\n        rules: {require_license_compliance_results_before_merging: false}",
		"enforcement_status: active\n        rules: {require_a_pull_request_before_merging: {restrict_who_can_dismiss_pull_request_reviews: false}}",
		"enforcement_status: active\n        rules: {require_a_pull_request_before_merging: {people_teams_or_apps_that_can_dismiss_reviews: []}}",
	} {
		content := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    rulesets:\n      - ruleset_name: basic\n        target_branches: {}\n        " + control + "\n"
		if err := writeSettingsFile(fixture.work, content); err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"validate", "plan", "apply"} {
			result, err := executeCLI(binary, fixture.work, fixture.root, env, []string{command}, false)
			if err != nil || result.runErr == nil {
				t.Fatalf("%s accepted %s: %v %s", command, control, err, result.stderr.String())
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid declaration made %d remote requests", calls.Load())
	}
}

func TestUnsupportedRulesetReadsBlockExportUpdateAndDeletion(t *testing.T) {
	binary := acceptanceBinary(t)
	const pr = `{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":false,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":1,"required_review_thread_resolution":true,"required_reviewers":[]}`
	cases := []struct {
		name, rule string
		evaluate   bool
	}{
		{name: "evaluate", evaluate: true},
		{name: "push"},
		{name: "quality", rule: `{"type":"code_quality","parameters":{"severity":"errors"}}`},
		{name: "coverage", rule: `{"type":"code_coverage","parameters":{"minimum_coverage":0,"max_coverage_drop":0}}`},
		{name: "teams", rule: `{"type":"pull_request","parameters":` + strings.Replace(pr, `"required_reviewers":[]`, `"required_reviewers":[{"reviewer":{"type":"Team","id":1},"minimum_approvals":0,"file_patterns":[]}]`, 1) + `}`},
		{name: "file_path_restriction", rule: `{"type":"file_path_restriction","parameters":{}}`},
		{name: "max_file_path_length", rule: `{"type":"max_file_path_length","parameters":{}}`},
		{name: "file_extension_restriction", rule: `{"type":"file_extension_restriction","parameters":{}}`},
		{name: "max_file_size", rule: `{"type":"max_file_size","parameters":{}}`},

		{name: "queue", rule: `{"type":"merge_queue","parameters":{"check_response_timeout_minutes":60,"grouping_strategy":"ALLGREEN","max_entries_to_build":5,"max_entries_to_merge":5,"merge_method":"SQUASH","min_entries_to_merge":1,"min_entries_to_merge_wait_minutes":0}}`},
		{name: "license", rule: `{"type":"license_compliance_scanning"}`},
	}
	for _, kind := range []string{"commit_message_pattern", "commit_author_email_pattern", "committer_email_pattern", "branch_name_pattern", "tag_name_pattern"} {
		cases = append(cases, struct {
			name, rule string
			evaluate   bool
		}{name: kind, rule: fmt.Sprintf(`{"type":%q,"parameters":{"operator":"contains","pattern":"x"}}`, kind)})
	}
	for _, restriction := range []string{`null`, `{"enabled":false,"allowed_actors":[]}`, `{"enabled":true,"allowed_actors":[{"type":"User","id":42}]}`} {
		cases = append(cases, struct {
			name, rule string
			evaluate   bool
		}{name: "dismissal_" + restriction, rule: `{"type":"pull_request","parameters":` + strings.TrimSuffix(pr, "}") + `,"dismissal_restriction":` + restriction + `}}`})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var state map[string]any
			if err := json.Unmarshal([]byte(acceptanceRuleset), &state); err != nil {
				t.Fatal(err)
			}
			if test.name == "push" {
				state["target"] = "push"
				state["conditions"] = nil
			} else if test.evaluate {
				state["enforcement"] = "evaluate"
			} else {
				var rule map[string]any
				if err := json.Unmarshal([]byte(test.rule), &rule); err != nil {
					t.Fatal(err)
				}
				state["rules"] = append(state["rules"].([]any), rule)
			}
			var writes atomic.Int32
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method != "GET" && r.URL.Path != "/graphql" {
					writes.Add(1)
					w.WriteHeader(500)
					return
				}
				if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					fmt.Fprint(w, planResponse)
				case "/repos/owner/current/topics":
					fmt.Fprint(w, `{"names":[]}`)
				case "/repos/owner/current/labels":
					fmt.Fprint(w, `[]`)
				case "/repos/owner/current/rulesets":
					fmt.Fprint(w, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`)
				case "/repos/owner/current/rulesets/7":
					json.NewEncoder(w).Encode(state)
				default:
					t.Error("unexpected request", r.Method, r.URL)
					w.WriteHeader(404)
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
			for _, deleting := range []bool{false, true} {
				selection := "enforcement_status: active\n        target_branches: {include_default_branch: true}\n        rules: {restrict_deletions: true}"
				if deleting {
					selection = "delete: true"
				}
				content := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    rulesets:\n      - id: 7\n        ruleset_name: basic\n        " + selection + "\n"
				if err := writeSettingsFile(fixture.work, content); err != nil {
					t.Fatal(err)
				}
				for _, args := range [][]string{{"export", "--rulesets"}, {"plan"}, {"apply"}} {
					result, err := executeCLI(binary, fixture.work, fixture.root, env, args, false)
					if err != nil || result.runErr == nil {
						t.Fatalf("%v accepted unsupported policy: %v %s", args, err, result.stderr.String())
					}
					saved, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
					if err != nil || string(saved) != content {
						t.Fatal("failed read replaced declaration", err)
					}
				}
			}
			if writes.Load() != 0 {
				t.Fatalf("unsupported protection caused %d writes", writes.Load())
			}
		})
	}
}
