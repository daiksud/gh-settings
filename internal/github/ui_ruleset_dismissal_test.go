package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func uiDismissalDeclaration(t *testing.T, rule uimodel.PullRequestRule) (*settings.ReviewDismissalRestriction, error) {
	t.Helper()
	ui := uiRulesetBase()
	ui.Rules.RequireAPullRequestBeforeMerging = &rule
	got, err := RulesetsDeclaration(rulesetPtr([]uimodel.Ruleset{ui}))
	if err != nil {
		return nil, err
	}
	return (*got.Values()[0].Rules)[0].Parameters.DismissalRestriction, nil
}
func TestUIRulesetReviewDismissalPresenceAndChoices(t *testing.T) {
	rawUI := uiRulesetBase()
	rawUI.Rules.RequireAPullRequestBeforeMerging = &uimodel.PullRequestRule{RestrictWhoCanDismissPullRequestReviews: rulesetPtr(true)}
	raw, err := RulesetDeclaration(rawUI)
	if err != nil || (*raw.Rules)[0].Parameters.DismissalRestriction == nil || (*raw.Rules)[0].Parameters.DismissalRestriction.AllowedActors != nil {
		t.Fatal("pre-default presence changed", raw, err)
	}

	if got, err := uiDismissalDeclaration(t, uimodel.PullRequestRule{}); err != nil || got != nil {
		t.Fatal(got, err)
	}
	for _, enabled := range []bool{false, true} {
		got, err := uiDismissalDeclaration(t, uimodel.PullRequestRule{RestrictWhoCanDismissPullRequestReviews: &enabled})
		want := &settings.ReviewDismissalRestriction{Enabled: &enabled, AllowedActors: rulesetPtr([]settings.ReviewActor{})}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatal(got, want, err)
		}
	}
	for _, test := range []struct {
		ui  uimodel.ReviewDismissalActor
		api settings.ReviewActor
	}{
		{uimodel.ReviewDismissalActor{Role: rulesetPtr(uimodel.RepositoryAdmin)}, settings.ReviewActor{Type: "RepositoryRole", ID: 5}},
		{uimodel.ReviewDismissalActor{Role: rulesetPtr(uimodel.Maintain)}, settings.ReviewActor{Type: "RepositoryRole", ID: 2}},
		{uimodel.ReviewDismissalActor{Role: rulesetPtr(uimodel.Write)}, settings.ReviewActor{Type: "RepositoryRole", ID: 4}},
		{uimodel.ReviewDismissalActor{User: rulesetPtr(int64(11))}, settings.ReviewActor{Type: "User", ID: 11}},
		{uimodel.ReviewDismissalActor{Team: rulesetPtr(int64(12))}, settings.ReviewActor{Type: "Team", ID: 12}},
		{uimodel.ReviewDismissalActor{App: rulesetPtr(int64(13))}, settings.ReviewActor{Type: "IntegrationInstallation", ID: 13}},
	} {
		actors := []uimodel.ReviewDismissalActor{test.ui}
		got, err := uiDismissalDeclaration(t, uimodel.PullRequestRule{RestrictWhoCanDismissPullRequestReviews: rulesetPtr(true), PeopleTeamsOrAppsThatCanDismissReviews: &actors})
		if err != nil || !reflect.DeepEqual(got, &settings.ReviewDismissalRestriction{Enabled: rulesetPtr(true), AllowedActors: rulesetPtr([]settings.ReviewActor{test.api})}) {
			t.Fatal(got, err)
		}
	}
	empty := []uimodel.ReviewDismissalActor{}
	got, err := uiDismissalDeclaration(t, uimodel.PullRequestRule{RestrictWhoCanDismissPullRequestReviews: rulesetPtr(false), PeopleTeamsOrAppsThatCanDismissReviews: &empty})
	if err != nil || got == nil || *got.Enabled || len(*got.AllowedActors) != 0 {
		t.Fatal(got, err)
	}
}
func TestUIRulesetReviewDismissalInvalidAndOwned(t *testing.T) {
	actor := uimodel.ReviewDismissalActor{User: rulesetPtr(int64(11))}
	for _, enabled := range []*bool{nil, rulesetPtr(false)} {
		if _, err := uiDismissalDeclaration(t, uimodel.PullRequestRule{RestrictWhoCanDismissPullRequestReviews: enabled, PeopleTeamsOrAppsThatCanDismissReviews: rulesetPtr([]uimodel.ReviewDismissalActor{actor})}); err == nil {
			t.Fatal("hidden actor intent accepted")
		}
	}
	for _, invalid := range []uimodel.ReviewDismissalActor{{}, {User: rulesetPtr(int64(0))}, {App: rulesetPtr(int64(-1))}, {Team: rulesetPtr(int64(12)), User: rulesetPtr(int64(11))}, {Role: rulesetPtr(uimodel.BypassRole("unknown"))}} {
		if _, err := uiDismissalDeclaration(t, uimodel.PullRequestRule{RestrictWhoCanDismissPullRequestReviews: rulesetPtr(true), PeopleTeamsOrAppsThatCanDismissReviews: rulesetPtr([]uimodel.ReviewDismissalActor{invalid})}); err == nil {
			t.Fatal("invalid actor accepted")
		}
	}
	actors := []uimodel.ReviewDismissalActor{actor}
	got, err := uiDismissalDeclaration(t, uimodel.PullRequestRule{RestrictWhoCanDismissPullRequestReviews: rulesetPtr(true), PeopleTeamsOrAppsThatCanDismissReviews: &actors})
	if err != nil {
		t.Fatal(err)
	}
	*actor.User = 99
	actors[0] = uimodel.ReviewDismissalActor{}
	if (*got.AllowedActors)[0].ID != 11 {
		t.Fatal("desired restriction aliases caller")
	}
}
