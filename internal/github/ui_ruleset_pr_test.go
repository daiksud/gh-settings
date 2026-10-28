package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIRulesetPRSettings(t *testing.T) {
	for _, count := range []int{0, 10} {
		ui := uiRulesetBase()
		ui.Rules.RequireAPullRequestBeforeMerging = &uimodel.PullRequestRule{
			RequiredApprovals:   &count,
			AllowedMergeMethods: rulesetPtr([]uimodel.MergeMethod{"merge", "squash", "rebase"}),
			DismissStalePullRequestApprovalsWhenNewCommitsArePushed:       rulesetPtr(false),
			RequireReviewFromCodeOwners:                                   rulesetPtr(true),
			RequireApprovalOfTheMostRecentReviewablePush:                  rulesetPtr(false),
			RequireConversationResolutionBeforeMerging:                    rulesetPtr(true),
			RequireAnAdditionalApprovalForUnattributedCopilotPullRequests: rulesetPtr(true),
		}
		got, err := RulesetDeclaration(ui)
		want := settings.RuleParameters{RequiredApprovingReviewCount: &count, AllowedMergeMethods: rulesetPtr([]string{"merge", "squash", "rebase"}), DismissStaleReviewsOnPush: rulesetPtr(false), RequireCodeOwnerReview: rulesetPtr(true), RequireLastPushApproval: rulesetPtr(false), RequiredReviewThreadResolution: rulesetPtr(true), RequireExtraApprovalForUnattributedChanges: rulesetPtr(true)}
		if err != nil || got.Rules == nil || len(*got.Rules) != 1 || (*got.Rules)[0].Type != "pull_request" || !reflect.DeepEqual((*got.Rules)[0].Parameters, &want) {
			t.Fatalf("rules=%+v err=%v want %+v", got.Rules, err, want)
		}
		want.RequiredReviewers = rulesetPtr([]settings.RequiredReviewer{})
		source := apiRulesetBase()
		source.Rules = rulesetPtr([]settings.RulesetRule{{Type: "pull_request", Parameters: &want}})
		observed, err := RulesetObservation(source)
		if err != nil || !reflect.DeepEqual(observed.Rules.RequireAPullRequestBeforeMerging, ui.Rules.RequireAPullRequestBeforeMerging) {
			t.Fatalf("PR=%+v err=%v", observed.Rules.RequireAPullRequestBeforeMerging, err)
		}
	}
}

func TestUIRulesetPRCountChoices(t *testing.T) {
	for _, count := range []int{-1, 11} {
		ui := uiRulesetBase()
		ui.Rules.RequireAPullRequestBeforeMerging = &uimodel.PullRequestRule{RequiredApprovals: &count}
		if _, err := RulesetDeclaration(ui); err == nil {
			t.Fatalf("out-of-UI required approvals accepted: %d", count)
		}
	}
	ui := uiRulesetBase()
	ui.Rules.RequireAPullRequestBeforeMerging = &uimodel.PullRequestRule{AllowedMergeMethods: rulesetPtr([]uimodel.MergeMethod{"MERGE"})}
	if _, err := RulesetDeclaration(ui); err == nil {
		t.Fatal("invalid merge method accepted")
	}
	source := apiRulesetBase()
	source.Rules = rulesetPtr([]settings.RulesetRule{{Type: "pull_request", Parameters: &settings.RuleParameters{RequiredApprovingReviewCount: rulesetPtr(11), AllowedMergeMethods: rulesetPtr([]string{"merge"}), DismissStaleReviewsOnPush: rulesetPtr(false), RequireCodeOwnerReview: rulesetPtr(false), RequireLastPushApproval: rulesetPtr(false), RequiredReviewThreadResolution: rulesetPtr(false), RequireExtraApprovalForUnattributedChanges: rulesetPtr(false), RequiredReviewers: rulesetPtr([]settings.RequiredReviewer{})}}})
	if _, err := RulesetObservation(source); err == nil {
		t.Fatal("API-valid out-of-UI count accepted")
	}
}

func TestUIRulesetPRSingletonFlags(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, test := range []struct {
			ui  uimodel.PullRequestRule
			api settings.RuleParameters
		}{
			{uimodel.PullRequestRule{DismissStalePullRequestApprovalsWhenNewCommitsArePushed: &value}, settings.RuleParameters{DismissStaleReviewsOnPush: &value}},
			{uimodel.PullRequestRule{RequireReviewFromCodeOwners: &value}, settings.RuleParameters{RequireCodeOwnerReview: &value}},
			{uimodel.PullRequestRule{RequireApprovalOfTheMostRecentReviewablePush: &value}, settings.RuleParameters{RequireLastPushApproval: &value}},
			{uimodel.PullRequestRule{RequireConversationResolutionBeforeMerging: &value}, settings.RuleParameters{RequiredReviewThreadResolution: &value}},
			{uimodel.PullRequestRule{RequireAnAdditionalApprovalForUnattributedCopilotPullRequests: &value}, settings.RuleParameters{RequireExtraApprovalForUnattributedChanges: &value}},
		} {
			ui := uiRulesetBase()
			ui.Rules.RequireAPullRequestBeforeMerging = &test.ui
			got, err := RulesetDeclaration(ui)
			if err != nil || got.Rules == nil || len(*got.Rules) != 1 || !reflect.DeepEqual((*got.Rules)[0].Parameters, &test.api) {
				t.Fatalf("rules=%+v err=%v want %+v", got.Rules, err, test.api)
			}
		}
	}
}
