package settings

import (
	"reflect"
	"testing"
)

func actionsTestPointer[T any](v T) *T { return &v }

func TestActionsPolicyObservationAndTransition(t *testing.T) {
	current := ActionsPolicyState{Enabled: false, AllowedActions: "all", SHAPinningRequired: true}
	full := ActionsPolicyDeclaration{AllowedActions: actionsTestPointer("selected"), SelectedActions: SelectedActionsDeclaration{GitHubOwnedAllowed: actionsTestPointer(false), VerifiedAllowed: actionsTestPointer(false), PatternsAllowed: actionsTestPointer([]string{})}}
	managed, changes, err := CompareActionsPolicy(current, full)
	if err != nil || managed != 4 || len(changes) != 4 || ValidateActionsPolicyChanges(changes) != nil {
		t.Fatalf("managed=%d changes=%v error=%v", managed, changes, err)
	}
	for _, c := range changes[1:] {
		before, _ := c.Values()
		if before != nil {
			t.Fatal("inactive values invented", c)
		}
	}
	for _, d := range []ActionsPolicyDeclaration{
		{AllowedActions: actionsTestPointer("selected")},
		{AllowedActions: actionsTestPointer("selected"), SelectedActions: SelectedActionsDeclaration{VerifiedAllowed: actionsTestPointer(false)}},
		{SelectedActions: full.SelectedActions},
		{AllowedActions: actionsTestPointer("all"), SelectedActions: full.SelectedActions},
	} {
		if _, _, err := CompareActionsPolicy(current, d); err == nil {
			t.Fatal("incomplete/inconsistent transition accepted", d)
		}
	}
	current.AllowedActions = "selected"
	if _, _, err := CompareActionsPolicy(current, full); err == nil {
		t.Fatal("missing acquired selected state accepted")
	}
	current.Selected = &SelectedActionsState{PatternsAllowed: []string{}}
	managed, changes, err = CompareActionsPolicy(current, full)
	if err != nil || managed != 4 || len(changes) != 0 {
		t.Fatal(managed, changes, err)
	}
	declaration, err := current.Declaration()
	if err != nil {
		t.Fatal(err)
	}
	current.Selected.GitHubOwnedAllowed = true
	if *declaration.SelectedActions.GitHubOwnedAllowed {
		t.Fatal("export aliases observed state")
	}
	if _, err := (ActionsPolicyState{AllowedActions: "all"}).Declaration(); err != nil {
		t.Fatal(err)
	}
}

func TestActionsPatternsAreExactSequences(t *testing.T) {
	patterns := []string{"org/*", "!org/bad@*", "org/*"}
	current := ActionsPolicyState{AllowedActions: "selected", Selected: &SelectedActionsState{PatternsAllowed: append([]string{}, patterns...)}}
	d := ActionsPolicyDeclaration{SelectedActions: SelectedActionsDeclaration{PatternsAllowed: &patterns}}
	managed, changes, err := CompareActionsPolicy(current, d)
	if err != nil || managed != 1 || len(changes) != 0 {
		t.Fatal(managed, changes, err)
	}
	patterns[0], patterns[1] = patterns[1], patterns[0]
	_, changes, err = CompareActionsPolicy(current, d)
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	before, after := changes[0].Values()
	after.([]string)[0] = "changed"
	_, again := changes[0].Values()
	if reflect.DeepEqual(after, again) || !reflect.DeepEqual(before, current.Selected.PatternsAllowed) {
		t.Fatal("comparison aliases mutable values")
	}
	for _, v := range [][]string{nil, {""}, make([]string, 1001)} {
		if ValidateActionPatterns(v) == nil {
			t.Fatal("invalid patterns accepted")
		}
	}
}

func TestActionsReportRejectsInvalidAndForgedUnknownValues(t *testing.T) {
	for _, test := range []struct {
		path          string
		before, after any
	}{
		{"actions.enabled", nil, false}, {"actions.allowed_actions", "all", "unknown"}, {"actions.enabled", false, "false"},
		{"actions.selected_actions.patterns_allowed", []any{}, []any{false}}, {"actions.other", false, true},
		{"actions.enabled", false, false},
	} {
		if _, err := NewActionsPolicyChange(test.path, test.before, test.after); err == nil {
			t.Fatal(test)
		}
	}
	c, err := NewActionsPolicyChange("actions.selected_actions.verified_allowed", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if ValidateActionsPolicyChanges([]ActionsPolicyChange{c}) == nil {
		t.Fatal("forged unknown without policy transition accepted")
	}
	c, err = NewActionsPolicyChange("actions.enabled", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if ValidateActionsPolicyChanges([]ActionsPolicyChange{c, c}) == nil {
		t.Fatal("duplicate changes accepted")
	}
}
