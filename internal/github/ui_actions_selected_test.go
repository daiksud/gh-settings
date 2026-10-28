package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIActionsSelectedSingletons(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, test := range []struct {
			ui   uimodel.ActionsGeneral
			want settings.SelectedActionsDeclaration
		}{
			{uimodel.ActionsGeneral{AllowActionsCreatedByGitHub: &value}, settings.SelectedActionsDeclaration{GitHubOwnedAllowed: &value}},
			{uimodel.ActionsGeneral{AllowActionsByMarketplaceVerifiedCreators: &value}, settings.SelectedActionsDeclaration{VerifiedAllowed: &value}},
			{uimodel.ActionsGeneral{AllowOrBlockSpecifiedActionsAndReusableWorkflows: actionsUIPtr([]string{})}, settings.SelectedActionsDeclaration{PatternsAllowed: actionsUIPtr([]string{})}},
		} {
			got, err := ActionsDeclaration(test.ui)
			if err != nil || !reflect.DeepEqual(got, settings.ActionsPolicyDeclaration{SelectedActions: test.want}) {
				t.Fatal(got, err, test.want)
			}
		}
	}
	patterns := []string{"example/a@v1", "!example/a@v2", "example/a@v1"}
	got, err := ActionsDeclaration(uimodel.ActionsGeneral{AllowOrBlockSpecifiedActionsAndReusableWorkflows: &patterns})
	if err != nil {
		t.Fatal(err)
	}
	patterns[0] = "changed"
	if !reflect.DeepEqual(*got.SelectedActions.PatternsAllowed, []string{"example/a@v1", "!example/a@v2", "example/a@v1"}) {
		t.Fatal("pattern conversion aliases or reorders values")
	}
}

func TestUIActionsSelectedVisibilityAndDependencies(t *testing.T) {
	for _, choice := range []uimodel.ActionsPermissionChoice{"disable_actions", "allow_all_actions_and_reusable_workflows", "allow_owner_actions_and_reusable_workflows"} {
		if _, err := ActionsDeclaration(uimodel.ActionsGeneral{ActionsPermissions: &choice, AllowActionsCreatedByGitHub: actionsUIPtr(false)}); err == nil {
			t.Fatal("incompatible radio/companion accepted")
		}
	}
	ui := uimodel.ActionsGeneral{ActionsPermissions: actionsUIPtr(uimodel.ActionsPermissionChoice("allow_owner_and_select_non_owner_actions_and_reusable_workflows")), AllowActionsCreatedByGitHub: actionsUIPtr(false)}
	partial, err := ActionsDeclaration(ui)
	if err != nil {
		t.Fatal(err)
	}
	current := settings.ActionsPolicyState{Enabled: true, AllowedActions: "all"}
	if err := settings.PrepareActions(current, partial); err == nil {
		t.Fatal("incomplete selected transition accepted")
	}
	ui.AllowActionsByMarketplaceVerifiedCreators = actionsUIPtr(false)
	ui.AllowOrBlockSpecifiedActionsAndReusableWorkflows = actionsUIPtr([]string{})
	complete, err := ActionsDeclaration(ui)
	if err != nil || settings.PrepareActions(current, complete) != nil {
		t.Fatal(complete, err)
	}
	state := settings.ActionsPolicyState{Enabled: true, AllowedActions: "selected", Selected: &settings.SelectedActionsState{GitHubOwnedAllowed: false, VerifiedAllowed: true, PatternsAllowed: []string{}}}
	view, err := ActionsExecutionObservation(state)
	if err != nil || view.General.AllowActionsCreatedByGitHub == nil || *view.General.AllowActionsCreatedByGitHub || !*view.General.AllowActionsByMarketplaceVerifiedCreators || view.General.AllowOrBlockSpecifiedActionsAndReusableWorkflows == nil || len(*view.General.AllowOrBlockSpecifiedActionsAndReusableWorkflows) != 0 {
		t.Fatal(view, err)
	}
	state.Enabled = false
	hidden, err := ActionsExecutionObservation(state)
	if err != nil || hidden.General.AllowActionsCreatedByGitHub != nil || hidden.General.AllowOrBlockSpecifiedActionsAndReusableWorkflows != nil || hidden.Raw.Selected == nil {
		t.Fatal(hidden, err)
	}
}
