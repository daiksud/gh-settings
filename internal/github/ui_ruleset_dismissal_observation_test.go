package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func TestUIRulesetDismissalAcquiredProvenance(t *testing.T) {
	source := apiRulesetBase()
	p := completeUIAPIPR()
	source.Rules = rulesetPtr([]settings.RulesetRule{{Type: "pull_request", Parameters: p}})
	view, err := RulesetObservation(source)
	if err != nil || view.UnverifiedReviewDismissalRestriction != nil || view.Rules.RequireAPullRequestBeforeMerging.RestrictWhoCanDismissPullRequestReviews != nil || view.Rules.RequireAPullRequestBeforeMerging.PeopleTeamsOrAppsThatCanDismissReviews != nil {
		t.Fatal("absent restriction became false", view, err)
	}
	for _, test := range []struct {
		api settings.ReviewActor
		ui  uimodel.ReviewDismissalActor
	}{
		{settings.ReviewActor{Type: "RepositoryRole", ID: 5}, uimodel.ReviewDismissalActor{Role: rulesetPtr(uimodel.RepositoryAdmin)}},
		{settings.ReviewActor{Type: "RepositoryRole", ID: 2}, uimodel.ReviewDismissalActor{Role: rulesetPtr(uimodel.Maintain)}},
		{settings.ReviewActor{Type: "RepositoryRole", ID: 4}, uimodel.ReviewDismissalActor{Role: rulesetPtr(uimodel.Write)}},
		{settings.ReviewActor{Type: "User", ID: 11}, uimodel.ReviewDismissalActor{User: rulesetPtr(int64(11))}},
		{settings.ReviewActor{Type: "Team", ID: 12}, uimodel.ReviewDismissalActor{Team: rulesetPtr(int64(12))}},
		{settings.ReviewActor{Type: "IntegrationInstallation", ID: 13}, uimodel.ReviewDismissalActor{App: rulesetPtr(int64(13))}},
	} {
		for _, enabled := range []bool{false, true} {
			p.DismissalRestriction = &settings.ReviewDismissalRestriction{Enabled: &enabled, AllowedActors: rulesetPtr([]settings.ReviewActor{test.api})}
			view, err := RulesetObservation(source)
			if err != nil || !reflect.DeepEqual(view.UnverifiedReviewDismissalRestriction, p.DismissalRestriction) || *view.Rules.RequireAPullRequestBeforeMerging.RestrictWhoCanDismissPullRequestReviews != enabled {
				t.Fatal(view, err)
			}
			visible := view.Rules.RequireAPullRequestBeforeMerging.PeopleTeamsOrAppsThatCanDismissReviews
			if enabled {
				if !reflect.DeepEqual(visible, rulesetPtr([]uimodel.ReviewDismissalActor{test.ui})) {
					t.Fatal(visible)
				}
			} else if visible != nil {
				t.Fatal("latent actors exposed as UI intent")
			}
			(*view.UnverifiedReviewDismissalRestriction.AllowedActors)[0].ID = 99
			if (*p.DismissalRestriction.AllowedActors)[0].ID != test.api.ID {
				t.Fatal("raw metadata aliases source")
			}
		}
	}
	p.DismissalRestriction = &settings.ReviewDismissalRestriction{Enabled: rulesetPtr(false), AllowedActors: rulesetPtr([]settings.ReviewActor{})}
	acquired, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: source}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	collected, err := RulesetsObservation(acquired)
	if err != nil || len(collected.Rulesets) != 1 || collected.Rulesets[0].UnverifiedReviewDismissalRestriction == nil || *collected.Rulesets[0].ID != 7 {
		t.Fatal("collection lost write-only provenance", collected, err)
	}
}
