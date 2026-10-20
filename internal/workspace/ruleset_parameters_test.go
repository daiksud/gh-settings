package workspace

import (
	"reflect"
	"strings"
	"testing"
)

const reviewConfig = `schemaVersion: 1
settings:
  rulesets:
    - name: default
      target: branch
      enforcement: active
      conditions: {ref_name: {include: ['~DEFAULT_BRANCH'], exclude: []}}
      bypass_actors: []
      rules:
        - type: pull_request
          parameters:
            allowed_merge_methods: [squash, merge]
            dismiss_stale_reviews_on_push: true
            require_code_owner_review: true
            require_extra_approval_for_unattributed_changes: false
            require_last_push_approval: false
            required_approving_review_count: 0
            required_review_thread_resolution: true
            dismissal_restriction: {enabled: false, allowed_actors: [{type: Team, id: 21}]}
            required_reviewers:
              - reviewer: {type: Team, id: 22}
                minimum_approvals: 0
                file_patterns: ['**', '!docs/**', 'docs/README.md', '**']
        - type: required_status_checks
          parameters:
            required_status_checks: [{context: build, integration_id: 8}, {context: build}]
            strict_required_status_checks_policy: false
            do_not_enforce_on_create: false
`

func TestRulesetReviewConfigRoundtripPreservesOrderAndPresence(t *testing.T) {
	d, e := DecodeConfig([]byte(reviewConfig))
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := EncodeConfig(d)
	if e != nil {
		t.Fatal(e)
	}
	round, e := DecodeConfig(encoded)
	if e != nil || !reflect.DeepEqual(round, d) {
		t.Fatalf("%s %v", encoded, e)
	}
	text := string(encoded)
	if strings.Index(text, "'**'") > strings.Index(text, "'!docs/**'") && strings.Contains(text, "'**'") {
		t.Fatal("pattern order changed", text)
	}
	policies := round.Rulesets.Values()
	for _, r := range *policies[0].Rules {
		if r.Type == "pull_request" {
			want := []string{"**", "!docs/**", "docs/README.md", "**"}
			if !reflect.DeepEqual(*(*r.Parameters.RequiredReviewers)[0].FilePatterns, want) || *r.Parameters.RequiredApprovingReviewCount != 0 || *r.Parameters.RequireExtraApprovalForUnattributedChanges {
				t.Fatal("lost policy", r)
			}
		}
	}
	// Native YAML bool case variants retain the same meaning as repository bools.
	if _, e := DecodeConfig([]byte(strings.Replace(reviewConfig, "require_code_owner_review: true", "require_code_owner_review: TRUE", 1))); e != nil {
		t.Fatal(e)
	}
}
func TestRulesetReviewConfigStrictNestedTypesAndNull(t *testing.T) {
	for _, pair := range [][2]string{{"required_approving_review_count: 0", "required_approving_review_count: '0'"}, {"require_last_push_approval: false", "require_last_push_approval: 'false'"}, {"required_reviewers:", "Required_reviewers:"}, {"minimum_approvals: 0", "minimum_approvals: null"}, {"file_patterns: ['**', '!docs/**', 'docs/README.md', '**']", "file_patterns: null"}, {"reviewer: {type: Team, id: 22}", "reviewer: {type: Team, id: 22, unknown: false}"}, {"enabled: false, allowed_actors:", "enabled: false, unknown: true, allowed_actors:"}, {"integration_id: 8", "integration_id: null"}, {"strict_required_status_checks_policy: false", "strict_required_status_checks_policy: null"}, {"do_not_enforce_on_create: false", "do_not_enforce_on_create: false\n            unknown_policy: false"}} {
		input := strings.Replace(reviewConfig, pair[0], pair[1], 1)
		if input == reviewConfig {
			t.Fatal("bad test anchor", pair)
		}
		if _, e := DecodeConfig([]byte(input)); e == nil {
			t.Fatal("accepted invalid review config", pair)
		}
	}
}
