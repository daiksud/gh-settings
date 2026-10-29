package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func uiConfigPtr[T any](value T) *T { return &value }

func TestUIConfigRulesetRootsAndSimpleControls(t *testing.T) {
	want := []uimodel.Ruleset{
		{ID: uiConfigPtr(int64(7)), RulesetName: "main", EnforcementStatus: uiConfigPtr(uimodel.Active), TargetBranches: &uimodel.BranchTargets{IncludeDefaultBranch: uiConfigPtr(true), IncludeAllBranches: uiConfigPtr(false), IncludeByPattern: uiConfigPtr([]string{}), ExcludeByPattern: uiConfigPtr([]string{"wip/*"})}, Rules: uimodel.RulesetRules{RestrictDeletions: uiConfigPtr(false), BlockForcePushes: uiConfigPtr(true), RequireLinearHistory: uiConfigPtr(true), RequireSignedCommits: uiConfigPtr(true)}},
		{RulesetName: "tags", TargetTags: &uimodel.TagTargets{IncludeAllTags: uiConfigPtr(true), IncludeByPattern: uiConfigPtr([]string{"release/*"})}, Rules: uimodel.RulesetRules{RestrictDeletions: uiConfigPtr(true)}},
	}
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - id: 7
        ruleset_name: main
        enforcement_status: active
        target_branches:
          include_default_branch: true
          include_all_branches: false
          include_by_pattern: []
          exclude_by_pattern: ['wip/*']
        rules:
          restrict_deletions: false
          block_force_pushes: true
          require_linear_history: true
          require_signed_commits: true
      - ruleset_name: tags
        target_tags:
          include_all_tags: true
          include_by_pattern: ['release/*']
        rules:
          restrict_deletions: true
`
	got, err := DecodeConfig([]byte(input))
	if err != nil || got.Settings == nil || got.Settings.Rulesets == nil || !reflect.DeepEqual(*got.Settings.Rulesets, want) {
		t.Fatalf("Ruleset control/identity changed: %#v, %v", got.Settings, err)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("Ruleset round trip changed intent: %s, %v", encoded, err)
	}
}

func TestUIConfigRulesetCollectionPresence(t *testing.T) {
	for _, input := range []string{"{}", "{rulesets: []}"} {
		got, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: " + input + "}\n"))
		if err != nil || got.Settings == nil || (got.Settings.Rulesets != nil) != (input != "{}") {
			t.Fatalf("Ruleset collection presence changed: %#v, %v", got.Settings, err)
		}
	}
}

func TestUIConfigRulesetsRejectsObsoleteAndMistypedControls(t *testing.T) {
	for _, entry := range []string{
		"name: old", "target: branch", "enforcement: active", "conditions: {}",
		"ruleset_name: null", "id: '7'", "target_branches: null",
		"target_branches: {include_all_branches: 'false'}",
		"target_branches: {include_by_pattern: [7]}",
		"rules: []", "rules: {deletion: true}", "rules: {block_force_pushes: null}",
	} {
		t.Run(entry, func(t *testing.T) {
			input := "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: [{" + entry + "}]}}\n"
			if _, err := DecodeConfig([]byte(input)); err == nil {
				t.Fatal("accepted obsolete or mistyped Ruleset control")
			}
		})
	}
}

func TestUIConfigRulesetPullRequestControls(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - ruleset_name: main
        rules:
          require_a_pull_request_before_merging:
            required_approvals: 0
            allowed_merge_methods: [squash, rebase]
            dismiss_stale_pull_request_approvals_when_new_commits_are_pushed: false
            require_review_from_code_owners: true
            require_approval_of_the_most_recent_reviewable_push: false
            require_conversation_resolution_before_merging: true
            require_an_additional_approval_for_unattributed_copilot_pull_requests: false
`
	want := &uimodel.PullRequestRule{
		RequiredApprovals: uiConfigPtr(0), AllowedMergeMethods: uiConfigPtr([]uimodel.MergeMethod{uimodel.Squash, uimodel.Rebase}),
		DismissStalePullRequestApprovalsWhenNewCommitsArePushed: uiConfigPtr(false), RequireReviewFromCodeOwners: uiConfigPtr(true),
		RequireApprovalOfTheMostRecentReviewablePush: uiConfigPtr(false), RequireConversationResolutionBeforeMerging: uiConfigPtr(true),
		RequireAnAdditionalApprovalForUnattributedCopilotPullRequests: uiConfigPtr(false),
	}
	got, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual((*got.Settings.Rulesets)[0].Rules.RequireAPullRequestBeforeMerging, want) {
		t.Fatalf("PR controls changed: %#v", (*got.Settings.Rulesets)[0].Rules)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("PR round trip changed intent: %s, %v", encoded, err)
	}
}

func TestUIConfigRulesetPullRequestRejectsInvalidTypesAndAPIKeys(t *testing.T) {
	for _, controls := range []string{
		"required_approvals: '0'", "required_approvals: 18446744073709551616", "required_approvals: null",
		"required_approving_review_count: 1", "allowed_merge_methods: squash", "allowed_merge_methods: [true]",
		"require_review_from_code_owners: 'true'", "require_conversation_resolution_before_merging: null",
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: [{rules: {require_a_pull_request_before_merging: {" + controls + "}}}]}}\n"
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatalf("accepted invalid PR controls: %s", controls)
		}
	}
}

func TestUIConfigRulesetIdentityCollections(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - ruleset_name: main
        bypass_list:
          - {role: maintain, bypass_mode: always}
          - {team: 12, bypass_mode: for_pull_requests_only}
          - {app: 17, bypass_mode: exempt}
          - {user: 21, bypass_mode: always}
          - {deploy_keys: true, bypass_mode: always}
        rules:
          require_a_pull_request_before_merging:
            restrict_who_can_dismiss_pull_request_reviews: true
            people_teams_or_apps_that_can_dismiss_reviews:
              - {role: repository_admin}
              - {team: 12}
              - {app: 83}
              - {user: 21}
            require_review_from_specific_teams:
              - reviewer: 12
                approvals: 0
                file_patterns: ['*.go', '!generated/*', '*.go']
      - ruleset_name: empty
        bypass_list: []
        rules:
          require_a_pull_request_before_merging:
            people_teams_or_apps_that_can_dismiss_reviews: []
            require_review_from_specific_teams: []
`
	want := []uimodel.Ruleset{
		{RulesetName: "main", BypassList: uiConfigPtr([]uimodel.BypassEntry{
			{Role: uiConfigPtr(uimodel.Maintain), BypassMode: uiConfigPtr(uimodel.Always)},
			{Team: uiConfigPtr(int64(12)), BypassMode: uiConfigPtr(uimodel.ForPullRequestsOnly)},
			{App: uiConfigPtr(int64(17)), BypassMode: uiConfigPtr(uimodel.Exempt)},
			{User: uiConfigPtr(int64(21)), BypassMode: uiConfigPtr(uimodel.Always)},
			{DeployKeys: uiConfigPtr(true), BypassMode: uiConfigPtr(uimodel.Always)},
		}), Rules: uimodel.RulesetRules{RequireAPullRequestBeforeMerging: &uimodel.PullRequestRule{
			RestrictWhoCanDismissPullRequestReviews: uiConfigPtr(true),
			PeopleTeamsOrAppsThatCanDismissReviews:  uiConfigPtr([]uimodel.ReviewDismissalActor{{Role: uiConfigPtr(uimodel.RepositoryAdmin)}, {Team: uiConfigPtr(int64(12))}, {App: uiConfigPtr(int64(83))}, {User: uiConfigPtr(int64(21))}}),
			RequireReviewFromSpecificTeams:          uiConfigPtr([]uimodel.TeamReview{{Reviewer: 12, Approvals: uiConfigPtr(0), FilePatterns: uiConfigPtr([]string{"*.go", "!generated/*", "*.go"})}}),
		}}},
		{RulesetName: "empty", BypassList: uiConfigPtr([]uimodel.BypassEntry{}), Rules: uimodel.RulesetRules{RequireAPullRequestBeforeMerging: &uimodel.PullRequestRule{PeopleTeamsOrAppsThatCanDismissReviews: uiConfigPtr([]uimodel.ReviewDismissalActor{}), RequireReviewFromSpecificTeams: uiConfigPtr([]uimodel.TeamReview{})}}},
	}
	got, err := DecodeConfig([]byte(input))
	if err != nil || !reflect.DeepEqual(*got.Settings.Rulesets, want) {
		t.Fatalf("identity collections changed: %#v, %v", got.Settings, err)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("identity collection round trip: %s, %v", encoded, err)
	}
}

func TestUIConfigRulesetIdentityCollectionsRejectInvalidSyntax(t *testing.T) {
	for _, entry := range []string{
		"bypass_list: null", "bypass_list: [{actor_type: Team}]", "bypass_list: [{team: '12'}]", "bypass_list: [{deploy_keys: null}]",
		"rules: {require_a_pull_request_before_merging: {people_teams_or_apps_that_can_dismiss_reviews: [{app: null}]}}",
		"rules: {require_a_pull_request_before_merging: {require_review_from_specific_teams: [{reviewer: '12'}]}}",
		"rules: {require_a_pull_request_before_merging: {require_review_from_specific_teams: [{approvals: '0'}]}}",
		"rules: {require_a_pull_request_before_merging: {require_review_from_specific_teams: [{file_patterns: [true]}]}}",
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: [{" + entry + "}]}}\n"
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatalf("accepted invalid identity syntax: %s", entry)
		}
	}
}

func TestUIConfigRulesetStatusAndAnalysisControls(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets:
      - ruleset_name: main
        rules:
          require_status_checks_to_pass:
            required_status_checks:
              - {check: ci, source: {any_source: true}}
              - {check: build, source: {app: 17}}
            require_branches_to_be_up_to_date_before_merging: true
            do_not_require_status_checks_on_creation: false
          require_code_scanning_results:
            code_scanning_tools:
              - {tool_name: CodeQL, security_alerts: high_or_higher, alerts: errors}
          require_code_quality_results:
            severity: warnings_and_higher
      - ruleset_name: empty
        rules:
          require_status_checks_to_pass: {required_status_checks: []}
          require_code_scanning_results: {code_scanning_tools: []}
`
	want := []uimodel.Ruleset{
		{RulesetName: "main", Rules: uimodel.RulesetRules{
			RequireStatusChecksToPass:  &uimodel.StatusChecksRule{RequiredStatusChecks: uiConfigPtr([]uimodel.StatusCheck{{Check: "ci", Source: uimodel.CheckSource{AnySource: uiConfigPtr(true)}}, {Check: "build", Source: uimodel.CheckSource{App: uiConfigPtr(int64(17))}}}), RequireBranchesToBeUpToDateBeforeMerging: uiConfigPtr(true), DoNotRequireStatusChecksOnCreation: uiConfigPtr(false)},
			RequireCodeScanningResults: &uimodel.CodeScanningRule{CodeScanningTools: uiConfigPtr([]uimodel.CodeScanningTool{{ToolName: "CodeQL", SecurityAlerts: "high_or_higher", Alerts: "errors"}})},
			RequireCodeQualityResults:  &uimodel.CodeQualityRule{Severity: uiConfigPtr(uimodel.CodeQualitySeverity("warnings_and_higher"))},
		}},
		{RulesetName: "empty", Rules: uimodel.RulesetRules{RequireStatusChecksToPass: &uimodel.StatusChecksRule{RequiredStatusChecks: uiConfigPtr([]uimodel.StatusCheck{})}, RequireCodeScanningResults: &uimodel.CodeScanningRule{CodeScanningTools: uiConfigPtr([]uimodel.CodeScanningTool{})}}},
	}
	got, err := DecodeConfig([]byte(input))
	if err != nil || !reflect.DeepEqual(*got.Settings.Rulesets, want) {
		t.Fatalf("status/analysis controls changed: %#v, %v", got.Settings, err)
	}
	encoded, err := EncodeConfig(got)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("status/analysis round trip: %s, %v", encoded, err)
	}
}

func TestUIConfigRulesetStatusAndAnalysisRejectInvalidSyntax(t *testing.T) {
	for _, rule := range []string{
		"require_status_checks_to_pass: {required_status_checks: [{context: ci}]}",
		"require_status_checks_to_pass: {required_status_checks: [{check: ci, source: {integration_id: 17}}]}",
		"require_status_checks_to_pass: {required_status_checks: [{check: ci, source: {app: '17'}}]}",
		"require_status_checks_to_pass: {do_not_require_status_checks_on_creation: null}",
		"require_code_scanning_results: {code_scanning_tools: [{tool: CodeQL}]}",
		"require_code_scanning_results: {code_scanning_tools: [{security_alerts: false}]}",
		"require_code_scanning_results: {code_scanning_tools: null}",
		"require_code_quality_results: {severity: null}",
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {rulesets: [{rules: {" + rule + "}}]}}\n"
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatalf("accepted invalid analysis syntax: %s", rule)
		}
	}
}
