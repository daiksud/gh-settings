package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"reflect"
	"strings"
	"testing"
)

const classicReviewsPrefix = "version: daiksud/gh-settings@v0\nrepository: {settings: {branch_protection_rules: [{branch_name_pattern: main, require_a_pull_request_before_merging: "
const activeReviewsLiteral = "{enabled: true, require_approvals: true, required_number_of_approvals_before_merging: 2, dismiss_stale_pull_request_approvals_when_new_commits_are_pushed: false, require_review_from_code_owners: false, require_approval_of_the_most_recent_reviewable_push: true}"

func TestClassicPullRequestReviewLiteralCodec(t *testing.T) {
	for _, value := range []string{"{}", "{enabled: false}", activeReviewsLiteral, strings.Replace(activeReviewsLiteral, "true, required_number_of_approvals_before_merging: 2", "false", 1)} {
		ui, err := DecodeConfig([]byte(classicReviewsPrefix + value + "}]}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = github.DeclarationFromUI(ui); err != nil {
			t.Fatal(value, err)
		}
		data, err := EncodeConfig(ui)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || !reflect.DeepEqual(ui, again) {
			t.Fatal(string(data), err)
		}
	}
}
func TestClassicReviewsRejectIncompleteUnion(t *testing.T) {
	invalid := []string{"null", "true", "[]", "{enabled: null}", "{enabled: 'false'}", "{enabled: true}", "{require_approvals: true}", "{enabled: false, require_approvals: false}", "{requiresApprovingReviews: true}"}
	for _, count := range []string{"null", "0", "7", "-1", "1.5", "'2'"} {
		invalid = append(invalid, strings.Replace(activeReviewsLiteral, ": 2,", ": "+count+",", 1))
	}
	for _, field := range []string{"require_approvals: true, ", "required_number_of_approvals_before_merging: 2, ", "dismiss_stale_pull_request_approvals_when_new_commits_are_pushed: false, ", "require_review_from_code_owners: false, ", ", require_approval_of_the_most_recent_reviewable_push: true"} {
		invalid = append(invalid, strings.Replace(activeReviewsLiteral, field, "", 1))
	}
	invalid = append(invalid, strings.Replace(activeReviewsLiteral, "require_approvals: true", "require_approvals: false", 1))
	for _, value := range invalid {
		ui, err := DecodeConfig([]byte(classicReviewsPrefix + value + "}]}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted", value)
		}
	}
}
