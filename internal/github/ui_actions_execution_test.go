package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func actionsUIPtr[T any](value T) *T { return &value }

func TestUIActionsPermissionChoices(t *testing.T) {
	for _, test := range []struct {
		choice  string
		enabled bool
		allowed *string
	}{
		{"allow_all_actions_and_reusable_workflows", true, actionsUIPtr("all")},
		{"disable_actions", false, nil},
		{"allow_owner_actions_and_reusable_workflows", true, actionsUIPtr("local_only")},
		{"allow_owner_and_select_non_owner_actions_and_reusable_workflows", true, actionsUIPtr("selected")},
	} {
		ui := uimodel.ActionsGeneral{ActionsPermissions: actionsUIPtr(uimodel.ActionsPermissionChoice(test.choice))}
		got, err := ActionsDeclaration(ui)
		want := settings.ActionsPolicyDeclaration{Enabled: &test.enabled, AllowedActions: test.allowed}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("declaration=%+v err=%v want %+v", got, err, want)
		}
		allowed := "local_only"
		if test.allowed != nil {
			allowed = *test.allowed
		}
		view, err := ActionsExecutionObservation(settings.ActionsPolicyState{Enabled: test.enabled, AllowedActions: allowed, SHAPinningRequired: true})
		if err != nil || view.General.ActionsPermissions == nil || *view.General.ActionsPermissions != uimodel.ActionsPermissionChoice(test.choice) || !*view.General.RequireActionsToBePinnedToAFullLengthCommitSHA {
			t.Fatalf("view=%+v err=%v", view, err)
		}
	}
}

func TestUIActionsOmissionSHAAndDisabledProvenance(t *testing.T) {
	if got, err := ActionsDeclaration(uimodel.ActionsGeneral{}); err != nil || !reflect.DeepEqual(got, settings.ActionsPolicyDeclaration{}) {
		t.Fatal(got, err)
	}
	for _, value := range []bool{false, true} {
		got, err := ActionsDeclaration(uimodel.ActionsGeneral{RequireActionsToBePinnedToAFullLengthCommitSHA: &value})
		want := settings.ActionsPolicyDeclaration{SHAPinningRequired: &value}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatal(got, err, want)
		}
	}
	for _, latent := range []string{"all", "local_only", "selected"} {
		view, err := ActionsExecutionObservation(settings.ActionsPolicyState{Enabled: false, AllowedActions: latent})
		if err != nil || *view.General.ActionsPermissions != "disable_actions" || view.Raw.AllowedActions != latent || view.Raw.Selected != nil {
			t.Fatal(view, err)
		}
	}
	state := settings.ActionsPolicyState{Enabled: false, AllowedActions: "selected", Selected: &settings.SelectedActionsState{PatternsAllowed: []string{"example/action@v1"}}}
	view, err := ActionsExecutionObservation(state)
	if err != nil {
		t.Fatal(err)
	}
	state.Selected.PatternsAllowed[0] = "changed"
	if view.Raw.Selected.PatternsAllowed[0] != "example/action@v1" {
		t.Fatal("raw provenance aliases caller")
	}
}

func TestUIActionsPermissionFailures(t *testing.T) {
	if _, err := ActionsDeclaration(uimodel.ActionsGeneral{ActionsPermissions: actionsUIPtr(uimodel.ActionsPermissionChoice("local_only"))}); err == nil {
		t.Fatal("API token accepted as UI choice")
	}
	if _, err := ActionsExecutionObservation(settings.ActionsPolicyState{Enabled: true, AllowedActions: "UNKNOWN"}); err == nil {
		t.Fatal("out-of-UI API token accepted")
	}
}
