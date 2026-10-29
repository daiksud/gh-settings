package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func executionVerificationState(raw string) settings.ActionsPolicyState {
	return settings.ActionsPolicyState{Enabled: true, AllowedActions: "all", SHAPinningRequired: true, AllowedActionsNormalization: settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: raw, Fallback: "allow_all_actions_and_reusable_workflows"}}
}

func TestActionsExecutionVerificationPreparesDeclaredAndRawPreservedValues(t *testing.T) {
	current := executionVerificationState("FUTURE_POLICY")
	for _, test := range []struct {
		name    string
		desired settings.ActionsPolicyDeclaration
		enabled bool
		policy  string
		sha     bool
	}{
		{"SHA only", settings.ActionsPolicyDeclaration{SHAPinningRequired: actionsUIPtr(false)}, true, "FUTURE_POLICY", false},
		{"disable only", settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(false)}, false, "FUTURE_POLICY", true},
		{"explicit all", settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true), AllowedActions: actionsUIPtr("all")}, true, "all", true},
		{"explicit local", settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true), AllowedActions: actionsUIPtr("local_only")}, true, "local_only", true},
	} {
		got, err := prepareActionsExecutionVerification(current, test.desired)
		want := actionsExecutionVerification{enabled: test.enabled, allowed: test.policy, sha: test.sha}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatal(test.name, got, want, err)
		}
	}
	if got, err := prepareActionsExecutionVerification(current, settings.ActionsPolicyDeclaration{AllowedActions: actionsUIPtr("all")}); err == nil || !reflect.DeepEqual(got, actionsExecutionVerification{}) {
		t.Fatal("partial radio inferred ownership", got, err)
	}
	if got, err := prepareActionsExecutionVerification(settings.ActionsPolicyState{}, settings.ActionsPolicyDeclaration{}); err == nil || !reflect.DeepEqual(got, actionsExecutionVerification{}) {
		t.Fatal("unacquired state defaulted", got, err)
	}
}

func TestActionsExecutionVerificationRequiresExactRawPolicyAndScalarCompanions(t *testing.T) {
	for _, test := range []struct {
		name, expected, actual string
		want                   bool
	}{
		{"unchanged unknown", "FUTURE_POLICY", "FUTURE_POLICY", true},
		{"changed unknown", "FUTURE_POLICY", "OTHER_POLICY", false},
		{"canonicalized omitted", "FUTURE_POLICY", "all", false},
		{"declared all retained unknown", "all", "FUTURE_POLICY", false},
		{"declared all confirmed", "all", "all", true},
		{"empty raw preserved", "", "", true},
	} {
		expected := actionsExecutionVerification{enabled: true, allowed: test.expected, sha: true}
		actual := executionVerificationState(test.actual)
		if test.actual == "all" {
			actual = settings.ActionsPolicyState{Enabled: true, AllowedActions: "all", SHAPinningRequired: true}
		}
		got, err := sameActionsExecutionVerification(expected, actual)
		if err != nil || got != test.want {
			t.Fatal(test.name, got, test.want, err)
		}
	}
	expected := actionsExecutionVerification{enabled: true, allowed: "FUTURE_POLICY", sha: true}
	for _, field := range []string{"enabled", "SHA"} {
		actual := executionVerificationState("FUTURE_POLICY")
		if field == "enabled" {
			actual.Enabled = false
		} else {
			actual.SHAPinningRequired = false
		}
		if got, err := sameActionsExecutionVerification(expected, actual); err != nil || got {
			t.Fatal(field, got, err)
		}
	}
	if got, err := sameActionsExecutionVerification(expected, settings.ActionsPolicyState{}); err == nil || got {
		t.Fatal("invalid state matched", got, err)
	}
}

func TestActionsExecutionVerificationOwnsCompleteSelectedCompanionIntent(t *testing.T) {
	current := settings.ActionsPolicyState{Enabled: true, AllowedActions: "selected", Selected: &settings.SelectedActionsState{GitHubOwnedAllowed: true, VerifiedAllowed: true, PatternsAllowed: []string{"owner/a@v1", "!owner/b@*", "owner/a@v1"}}}
	d := settings.ActionsPolicyDeclaration{SelectedActions: settings.SelectedActionsDeclaration{VerifiedAllowed: actionsUIPtr(false)}}
	expected, err := prepareActionsExecutionVerification(current, d)
	if err != nil {
		t.Fatal(err)
	}
	want := actionsExecutionVerification{enabled: true, allowed: "selected", selected: &settings.SelectedActionsState{GitHubOwnedAllowed: true, PatternsAllowed: []string{"owner/a@v1", "!owner/b@*", "owner/a@v1"}}}
	if !reflect.DeepEqual(expected, want) {
		t.Fatal(expected, want)
	}
	current.Selected.PatternsAllowed[0] = "mutated caller"
	if expected.selected.PatternsAllowed[0] != "owner/a@v1" {
		t.Fatal("verification aliases caller")
	}
	actual := settings.ActionsPolicyState{Enabled: true, AllowedActions: "selected", Selected: &settings.SelectedActionsState{GitHubOwnedAllowed: true, PatternsAllowed: []string{"owner/a@v1", "!owner/b@*", "owner/a@v1"}}}
	if got, err := sameActionsExecutionVerification(expected, actual); err != nil || !got {
		t.Fatal(got, err)
	}
	for _, field := range []string{"missing", "github owned", "verified", "patterns"} {
		changed := actual
		if field == "missing" {
			changed.Selected = nil
		} else {
			copy := *actual.Selected
			copy.PatternsAllowed = append([]string{}, actual.Selected.PatternsAllowed...)
			changed.Selected = &copy
			switch field {
			case "github owned":
				changed.Selected.GitHubOwnedAllowed = false
			case "verified":
				changed.Selected.VerifiedAllowed = true
			case "patterns":
				changed.Selected.PatternsAllowed[0], changed.Selected.PatternsAllowed[1] = changed.Selected.PatternsAllowed[1], changed.Selected.PatternsAllowed[0]
			}
		}
		if got, _ := sameActionsExecutionVerification(expected, changed); got {
			t.Fatal("selected evidence not retained", field)
		}
	}
	transition := settings.ActionsPolicyDeclaration{Enabled: actionsUIPtr(true), AllowedActions: actionsUIPtr("selected"), SelectedActions: settings.SelectedActionsDeclaration{GitHubOwnedAllowed: actionsUIPtr(false), VerifiedAllowed: actionsUIPtr(false), PatternsAllowed: actionsUIPtr([]string{})}}
	expected, err = prepareActionsExecutionVerification(executionVerificationState("FUTURE_POLICY"), transition)
	if err != nil || expected.selected == nil || expected.selected.PatternsAllowed == nil || len(expected.selected.PatternsAllowed) != 0 {
		t.Fatal("complete transition tuple lost", expected, err)
	}
}

func TestActionsExecutionVerificationDoesNotAcquireOmittedSelectedScope(t *testing.T) {
	current := settings.ActionsPolicyState{Enabled: true, AllowedActions: "selected", SHAPinningRequired: true, Selected: &settings.SelectedActionsState{GitHubOwnedAllowed: true, VerifiedAllowed: true, PatternsAllowed: []string{"owner/a@v1"}}}
	expected, err := prepareActionsExecutionVerification(current, settings.ActionsPolicyDeclaration{SHAPinningRequired: actionsUIPtr(false)})
	if err != nil || expected.selected != nil || expected.allowed != "selected" || !expected.enabled || expected.sha {
		t.Fatal(expected, err)
	}
	for _, selected := range []*settings.SelectedActionsState{nil, {PatternsAllowed: []string{"other/action@v2"}}} {
		actual := settings.ActionsPolicyState{Enabled: true, AllowedActions: "selected", Selected: selected}
		if got, err := sameActionsExecutionVerification(expected, actual); err != nil || !got {
			t.Fatal("unmanaged selected scope compared", got, err)
		}
	}
}
