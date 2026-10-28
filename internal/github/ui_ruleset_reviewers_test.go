package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func completeUIAPIPR() *settings.RuleParameters {
	return &settings.RuleParameters{RequiredApprovingReviewCount: rulesetPtr(0), AllowedMergeMethods: rulesetPtr([]string{"merge", "squash", "rebase"}), DismissStaleReviewsOnPush: rulesetPtr(false), RequireCodeOwnerReview: rulesetPtr(false), RequireLastPushApproval: rulesetPtr(false), RequiredReviewThreadResolution: rulesetPtr(false), RequireExtraApprovalForUnattributedChanges: rulesetPtr(false), RequiredReviewers: rulesetPtr([]settings.RequiredReviewer{})}
}

func TestUIRulesetTeamReviewers(t *testing.T) {
	for _, count := range []int{0, 10} {
		patterns := []string{"*.go", "!generated/*", "*.go"}
		teams := []uimodel.TeamReview{{Reviewer: 12, Approvals: &count, FilePatterns: &patterns}}
		ui := uiRulesetBase()
		ui.Rules.RequireAPullRequestBeforeMerging = &uimodel.PullRequestRule{RequireReviewFromSpecificTeams: &teams}
		got, err := RulesetDeclaration(ui)
		want := []settings.RequiredReviewer{{Reviewer: settings.ReviewActor{Type: "Team", ID: 12}, MinimumApprovals: &count, FilePatterns: rulesetPtr([]string{"*.go", "!generated/*", "*.go"})}}
		if err != nil || got.Rules == nil || !reflect.DeepEqual((*got.Rules)[0].Parameters.RequiredReviewers, &want) {
			t.Fatalf("rules=%+v err=%v want %+v", got.Rules, err, want)
		}
		patterns[0] = "changed"
		if (*(*got.Rules)[0].Parameters.RequiredReviewers)[0].Reviewer.ID != 12 || (*(*(*got.Rules)[0].Parameters.RequiredReviewers)[0].FilePatterns)[0] != "*.go" {
			t.Fatal("reviewer conversion aliases caller")
		}
		source := apiRulesetBase()
		p := completeUIAPIPR()
		p.RequiredReviewers = &want
		source.Rules = rulesetPtr([]settings.RulesetRule{{Type: "pull_request", Parameters: p}})
		observed, err := RulesetObservation(source)
		wantUI := []uimodel.TeamReview{{Reviewer: 12, Approvals: &count, FilePatterns: rulesetPtr([]string{"*.go", "!generated/*", "*.go"})}}
		if err != nil || !reflect.DeepEqual(observed.Rules.RequireAPullRequestBeforeMerging.RequireReviewFromSpecificTeams, &wantUI) {
			t.Fatalf("observation=%+v err=%v", observed.Rules, err)
		}
	}
}

func TestUIRulesetTeamReviewerFailures(t *testing.T) {
	for _, team := range []uimodel.TeamReview{
		{Reviewer: 0, Approvals: rulesetPtr(0), FilePatterns: rulesetPtr([]string{})},
		{Reviewer: 12, FilePatterns: rulesetPtr([]string{})},
		{Reviewer: 12, Approvals: rulesetPtr(0)},
		{Reviewer: 12, Approvals: rulesetPtr(11), FilePatterns: rulesetPtr([]string{})},
	} {
		ui := uiRulesetBase()
		ui.Rules.RequireAPullRequestBeforeMerging = &uimodel.PullRequestRule{RequireReviewFromSpecificTeams: rulesetPtr([]uimodel.TeamReview{team})}
		if _, err := RulesetDeclaration(ui); err == nil {
			t.Fatal("invalid team reviewer accepted")
		}
	}
	source := apiRulesetBase()
	p := completeUIAPIPR()
	p.DismissalRestriction = &settings.ReviewDismissalRestriction{Enabled: rulesetPtr(true), AllowedActors: rulesetPtr([]settings.ReviewActor{{Type: "RepositoryRole", ID: 99}})}
	source.Rules = rulesetPtr([]settings.RulesetRule{{Type: "pull_request", Parameters: p}})
	if _, err := RulesetObservation(source); err == nil {
		t.Fatal("out-of-UI dismissal role silently erased")
	}
}
