package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsExecutionWarningsRequireVisibleFreshCompleteRadio(t *testing.T) {
	visible := executionVerificationState("FUTURE_POLICY")
	hidden := visible
	hidden.Enabled = false
	fresh := executionVerificationState("OTHER_POLICY")
	invalid := visible
	invalid.AllowedActionsNormalization.RawValue = "all"
	for _, test := range []struct {
		name       string
		state      settings.ActionsPolicyState
		desired    settings.ActionsPolicyDeclaration
		sink, want bool
	}{
		{"visible disable", visible, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(false)}, true, true},
		{"visible all", visible, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true), AllowedActions: actionsUIPtr("all")}, true, true},
		{"visible local", visible, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true), AllowedActions: actionsUIPtr("local_only")}, true, true},
		{"visible selected", visible, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true), AllowedActions: actionsUIPtr("selected")}, true, true},
		{"hidden preflight", hidden, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true), AllowedActions: actionsUIPtr("all")}, true, false},
		{"fresh visible readback", fresh, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true), AllowedActions: actionsUIPtr("all")}, true, true},
		{"SHA only", visible, settings.ActionsPolicyDeclaration{SHAPinningRequired: actionsUIPtr(false)}, true, false},
		{"enabled only", visible, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true)}, true, false},
		{"selected only", visible, settings.ActionsPolicyDeclaration{SelectedActions: settings.SelectedActionsDeclaration{VerifiedAllowed: actionsUIPtr(false)}}, true, false},
		{"canonical", settings.ActionsPolicyState{Enabled: true, AllowedActions: "all"}, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(false)}, true, false},
		{"invalid state", invalid, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(false)}, true, false},
		{"nil sink", visible, settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(false)}, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &ActionsPolicyClient{}
			var got []settings.ObservationNormalization
			if test.sink {
				client.NormalizationWarning = func(n settings.ObservationNormalization) { got = append(got, n) }
			}
			client.warnExecutionNormalization(test.state, test.desired)
			var want []settings.ObservationNormalization
			if test.want {
				want = []settings.ObservationNormalization{test.state.AllowedActionsNormalization}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatal(got, want)
			}
		})
	}
}
