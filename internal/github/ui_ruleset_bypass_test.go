package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func uiRulesetBase() uimodel.Ruleset {
	return uimodel.Ruleset{RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("active")), TargetBranches: &uimodel.BranchTargets{}}
}

func apiRulesetBase() settings.RulesetDeclaration {
	return settings.RulesetDeclaration{Name: "Policy", Target: rulesetPtr("branch"), Enforcement: rulesetPtr("active"), Conditions: &settings.RulesetConditions{RefName: &settings.RefNameCondition{Include: rulesetPtr([]string{}), Exclude: rulesetPtr([]string{})}}, BypassActors: rulesetPtr([]settings.BypassActor{}), Rules: rulesetPtr([]settings.RulesetRule{})}
}

func TestUIRulesetBypassChoices(t *testing.T) {
	for _, mode := range []struct{ ui, api string }{{"always", "always"}, {"exempt", "exempt"}, {"for_pull_requests_only", "pull_request"}} {
		for _, actor := range []struct {
			ui   uimodel.BypassEntry
			kind string
			id   *int64
		}{
			{uimodel.BypassEntry{Role: rulesetPtr(uimodel.BypassRole("repository_admin"))}, "RepositoryRole", rulesetPtr(int64(5))},
			{uimodel.BypassEntry{Role: rulesetPtr(uimodel.BypassRole("maintain"))}, "RepositoryRole", rulesetPtr(int64(2))},
			{uimodel.BypassEntry{Role: rulesetPtr(uimodel.BypassRole("write"))}, "RepositoryRole", rulesetPtr(int64(4))},
			{uimodel.BypassEntry{Team: rulesetPtr(int64(12))}, "Team", rulesetPtr(int64(12))},
			{uimodel.BypassEntry{App: rulesetPtr(int64(13))}, "Integration", rulesetPtr(int64(13))},
			{uimodel.BypassEntry{User: rulesetPtr(int64(14))}, "User", rulesetPtr(int64(14))},
			{uimodel.BypassEntry{DeployKeys: rulesetPtr(true)}, "DeployKey", nil},
		} {
			if actor.kind == "DeployKey" && mode.api == "pull_request" {
				continue
			}
			actor.ui.BypassMode = rulesetPtr(uimodel.BypassMode(mode.ui))
			ui := uiRulesetBase()
			ui.BypassList = rulesetPtr([]uimodel.BypassEntry{actor.ui})
			got, err := RulesetDeclaration(ui)
			want := []settings.BypassActor{{ActorType: actor.kind, ActorID: actor.id, BypassMode: &mode.api}}
			if err != nil || got.BypassActors == nil || !reflect.DeepEqual(*got.BypassActors, want) {
				t.Fatalf("actors=%+v err=%v want %+v", got.BypassActors, err, want)
			}
			source := apiRulesetBase()
			source.BypassActors = &want
			observed, err := RulesetObservation(source)
			if err != nil || !reflect.DeepEqual(observed.BypassList, ui.BypassList) {
				t.Fatalf("bypass=%+v err=%v want %+v", observed.BypassList, err, ui.BypassList)
			}
		}
	}
}

func TestUIRulesetBypassFailures(t *testing.T) {
	for _, entry := range []uimodel.BypassEntry{
		{},
		{Role: rulesetPtr(uimodel.BypassRole("admin")), BypassMode: rulesetPtr(uimodel.BypassMode("always"))},
		{Role: rulesetPtr(uimodel.BypassRole("write"))},
		{Team: rulesetPtr(int64(0)), BypassMode: rulesetPtr(uimodel.BypassMode("always"))},
		{App: rulesetPtr(int64(13)), User: rulesetPtr(int64(14)), BypassMode: rulesetPtr(uimodel.BypassMode("always"))},
		{DeployKeys: rulesetPtr(false), BypassMode: rulesetPtr(uimodel.BypassMode("always"))},
		{DeployKeys: rulesetPtr(true), BypassMode: rulesetPtr(uimodel.BypassMode("for_pull_requests_only"))},
		{User: rulesetPtr(int64(14)), BypassMode: rulesetPtr(uimodel.BypassMode("pull_request"))},
	} {
		ui := uiRulesetBase()
		ui.BypassList = rulesetPtr([]uimodel.BypassEntry{entry})
		if _, err := RulesetDeclaration(ui); err == nil {
			t.Fatal("invalid bypass intent accepted", entry)
		}
	}
	for _, actor := range []settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: rulesetPtr("always")}, {ActorType: "RepositoryRole", ActorID: rulesetPtr(int64(99)), BypassMode: rulesetPtr("always")}} {
		source := apiRulesetBase()
		source.BypassActors = rulesetPtr([]settings.BypassActor{actor})
		if _, err := RulesetObservation(source); err == nil {
			t.Fatal("API-only actor accepted")
		}
	}
}
