package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func actionsPolicyUIPtr[T any](v T) *T { return &v }
func validActionsPolicyUI() uimodel.ActionsEventPolicy {
	return uimodel.ActionsEventPolicy{PolicyName: actionsPolicyUIPtr("Restrict events"), EnforcementStatus: actionsPolicyUIPtr("active"), RestrictEvents: &uimodel.ActionsRestrictEvents{AllowedEvents: actionsPolicyUIPtr([]string{"push", "pull_request"})}}
}

func TestActionsPoliciesUIConversion(t *testing.T) {
	empty, err := ActionsPoliciesDeclaration(nil)
	if err != nil || empty != nil {
		t.Fatalf("omission changed: %#v, %v", empty, err)
	}
	values := []uimodel.ActionsEventPolicy{}
	declared, err := ActionsPoliciesDeclaration(&values)
	if err != nil || declared == nil || declared.Len() != 0 {
		t.Fatalf("empty collection changed: %#v, %v", declared, err)
	}
	values = []uimodel.ActionsEventPolicy{validActionsPolicyUI()}
	values[0].ID = actionsPolicyUIPtr(int64(17))
	declared, err = ActionsPoliciesDeclaration(&values)
	if err != nil {
		t.Fatal(err)
	}
	expected := settings.ActionsEventPolicy{ID: 17, Name: "Restrict events", Enforcement: "active", Events: []string{"pull_request", "push"}}
	if !reflect.DeepEqual(declared.Values(), []settings.ActionsEventPolicy{expected}) {
		t.Fatalf("policy conversion: %#v", declared.Values())
	}
	(*values[0].RestrictEvents.AllowedEvents)[0] = "issues"
	if !reflect.DeepEqual(declared.Values()[0], expected) {
		t.Fatal("conversion leaked input event slice")
	}
	ui, err := ActionsEventPolicyUI(expected)
	if err != nil {
		t.Fatal(err)
	}
	(*ui.RestrictEvents.AllowedEvents)[0] = "issues"
	if expected.Events[0] != "pull_request" {
		t.Fatal("UI conversion leaked native event slice")
	}
	repository := uimodel.Repository{Settings: &uimodel.RepositorySettings{Actions: &uimodel.ActionsSettings{Policies: &values}}}
	result, err := DeclarationFromUI(repository)
	if err != nil || result.ActionsPolicies == nil || result.ActionsPolicies.Len() != 1 {
		t.Fatalf("declaration omitted policies: %#v, %v", result.ActionsPolicies, err)
	}
}

func TestActionsPoliciesUIRejectsIncompleteOrInvalidDeclarations(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*uimodel.ActionsEventPolicy)
	}{
		{"missing name", func(p *uimodel.ActionsEventPolicy) { p.PolicyName = nil }},
		{"empty name", func(p *uimodel.ActionsEventPolicy) { p.PolicyName = actionsPolicyUIPtr("") }},
		{"missing enforcement", func(p *uimodel.ActionsEventPolicy) { p.EnforcementStatus = nil }},
		{"unsupported enforcement", func(p *uimodel.ActionsEventPolicy) { p.EnforcementStatus = actionsPolicyUIPtr("evaluate") }},
		{"missing both restrictions", func(p *uimodel.ActionsEventPolicy) { p.RestrictEvents = nil }},
		{"missing events", func(p *uimodel.ActionsEventPolicy) { p.RestrictEvents.AllowedEvents = nil }},
		{"empty events", func(p *uimodel.ActionsEventPolicy) { p.RestrictEvents.AllowedEvents = actionsPolicyUIPtr([]string{}) }},
		{"unknown event", func(p *uimodel.ActionsEventPolicy) {
			p.RestrictEvents.AllowedEvents = actionsPolicyUIPtr([]string{"invented"})
		}},
		{"duplicate event", func(p *uimodel.ActionsEventPolicy) {
			p.RestrictEvents.AllowedEvents = actionsPolicyUIPtr([]string{"push", "push"})
		}},
		{"zero id", func(p *uimodel.ActionsEventPolicy) { p.ID = actionsPolicyUIPtr(int64(0)) }},
		{"negative id", func(p *uimodel.ActionsEventPolicy) { p.ID = actionsPolicyUIPtr(int64(-1)) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			policy := validActionsPolicyUI()
			test.change(&policy)
			values := []uimodel.ActionsEventPolicy{policy}
			if _, err := ActionsPoliciesDeclaration(&values); err == nil {
				t.Fatal("accepted invalid policy")
			}
		})
	}
	duplicate := []uimodel.ActionsEventPolicy{validActionsPolicyUI(), validActionsPolicyUI()}
	if _, err := ActionsPoliciesDeclaration(&duplicate); err == nil {
		t.Fatal("accepted duplicate policies")
	}
}

func TestActionsPoliciesExportRetainsOtherActionsSettings(t *testing.T) {
	empty := ""
	topics := []string{}
	days := 7
	policies, err := settings.NewActionsPolicyDeclarations([]settings.ActionsEventPolicy{{ID: 17, Name: "Restrict events", Enforcement: "disabled", Events: []string{"push"}}})
	if err != nil {
		t.Fatal(err)
	}
	variables, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "EXAMPLE", Value: "retained"}})
	if err != nil {
		t.Fatal(err)
	}
	source := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &empty}, Topics: settings.TopicsDeclaration{Names: &topics}, Actions: settings.ActionsPolicyDeclaration{RetentionDays: &days}, ActionsPolicies: &policies, Variables: &variables}
	ui, err := UIFromExportDeclaration(source)
	if err != nil {
		t.Fatal(err)
	}
	if ui.Settings.Actions.General == nil || ui.Settings.Actions.Variables == nil || ui.Settings.Actions.Policies == nil {
		t.Fatal("policies replaced other Actions settings")
	}
	again, err := DeclarationFromUI(ui)
	expectedPolicies := policies.Values()
	expectedPolicies[0].DisableActors = true
	if err != nil || !reflect.DeepEqual(again.ActionsPolicies.Values(), expectedPolicies) || again.Variables.Len() != 1 || *again.Actions.RetentionDays != days {
		t.Fatalf("export roundtrip lost values: %#v, %v", again, err)
	}
	// Explicit empty and omitted acquisition remain distinct in exported YAML intent.
	zero, err := settings.NewActionsPolicyDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	source.ActionsPolicies = &zero
	ui, err = UIFromExportDeclaration(source)
	if err != nil || ui.Settings.Actions.Policies == nil || len(*ui.Settings.Actions.Policies) != 0 {
		t.Fatalf("empty policy export lost presence: %#v, %v", ui.Settings.Actions, err)
	}
	source.ActionsPolicies = nil
	ui, err = UIFromExportDeclaration(source)
	if err != nil || ui.Settings.Actions.Policies != nil {
		t.Fatalf("unacquired policy export populated: %#v, %v", ui.Settings.Actions, err)
	}
}

func TestActionsPoliciesWorkflowTargetProjection(t *testing.T) {
	for _, target := range []*uimodel.ActionsWorkflowTarget{
		{}, {Include: actionsPolicyUIPtr([]string{})}, {Exclude: actionsPolicyUIPtr([]string{})},
	} {
		policy := validActionsPolicyUI()
		policy.TargetWorkflows = target
		policies := []uimodel.ActionsEventPolicy{policy}
		if _, err := ActionsPoliciesDeclaration(&policies); err == nil {
			t.Fatal("accepted incomplete targeting", target)
		}
	}
	policy := validActionsPolicyUI()
	policy.TargetWorkflows = &uimodel.ActionsWorkflowTarget{Include: actionsPolicyUIPtr([]string{"z", "a"}), Exclude: actionsPolicyUIPtr([]string{"skip"})}
	policies := []uimodel.ActionsEventPolicy{policy}
	declared, err := ActionsPoliciesDeclaration(&policies)
	if err != nil {
		t.Fatal(err)
	}
	(*policy.TargetWorkflows.Include)[0] = "mutated"
	value := declared.Values()[0]
	if !reflect.DeepEqual(value.TargetWorkflows.Include, []string{"a", "z"}) {
		t.Fatal(value)
	}
	ui, err := ActionsEventPolicyUI(value)
	if err != nil {
		t.Fatal(err)
	}
	(*ui.TargetWorkflows.Exclude)[0] = "mutated"
	if value.TargetWorkflows.Exclude[0] != "skip" {
		t.Fatal("projection alias")
	}
}

func TestActionsPoliciesActorMappingAndOwnedValues(t *testing.T) {
	actors := []uimodel.ActionsPolicyActor{
		{User: actionsPolicyUIPtr(int64(123))},
		{Role: actionsPolicyUIPtr("write")},
		{Role: actionsPolicyUIPtr("maintain")},
		{Role: actionsPolicyUIPtr("repository_admin")},
	}
	p := validActionsPolicyUI()
	p.RestrictActors = &uimodel.ActionsRestrictActors{AllowedActors: &actors}
	policies := []uimodel.ActionsEventPolicy{p}
	declared, err := ActionsPoliciesDeclaration(&policies)
	if err != nil {
		t.Fatal(err)
	}
	want := []settings.ActionsPolicyActor{{Type: "RepositoryRole", ID: 2}, {Type: "RepositoryRole", ID: 4}, {Type: "RepositoryRole", ID: 5}, {Type: "User", ID: 123}}
	native := declared.Values()[0]
	if native.Actors == nil || !reflect.DeepEqual(*native.Actors, want) {
		t.Fatalf("actor mapping: %#v", native.Actors)
	}
	*actors[0].User = 999
	if (*declared.Values()[0].Actors)[3].ID != 123 {
		t.Fatal("actor input alias")
	}
	ui, err := ActionsEventPolicyUI(native)
	if err != nil {
		t.Fatal(err)
	}
	if ui.RestrictActors.Enabled != nil || ui.RestrictEvents.Enabled != nil {
		t.Fatal("export needlessly enabled restrictions")
	}
	got := *ui.RestrictActors.AllowedActors
	if *got[0].Role != "maintain" || *got[1].Role != "write" || *got[2].Role != "repository_admin" || *got[3].User != 123 {
		t.Fatalf("reverse mapping: %#v", got)
	}
	*got[3].User = 456
	if (*native.Actors)[3].ID != 123 {
		t.Fatal("export alias")
	}
}

func TestActionsPoliciesRestrictionsPresence(t *testing.T) {
	for _, test := range []struct {
		name                           string
		events                         *uimodel.ActionsRestrictEvents
		actors                         *uimodel.ActionsRestrictActors
		disabledEvents, disabledActors bool
		hasActors                      bool
	}{
		{name: "omitted"},
		{name: "events off", events: &uimodel.ActionsRestrictEvents{Enabled: actionsPolicyUIPtr(false)}, disabledEvents: true},
		{name: "actors off", actors: &uimodel.ActionsRestrictActors{Enabled: actionsPolicyUIPtr(false)}, disabledActors: true},
		{name: "selected zero", actors: &uimodel.ActionsRestrictActors{AllowedActors: actionsPolicyUIPtr([]uimodel.ActionsPolicyActor{})}, hasActors: true},
		{name: "enabled selected zero", actors: &uimodel.ActionsRestrictActors{Enabled: actionsPolicyUIPtr(true), AllowedActors: actionsPolicyUIPtr([]uimodel.ActionsPolicyActor{})}, hasActors: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := validActionsPolicyUI()
			p.ID = actionsPolicyUIPtr(int64(17))
			p.RestrictEvents = test.events
			p.RestrictActors = test.actors
			policies := []uimodel.ActionsEventPolicy{p}
			result, err := ActionsPoliciesDeclaration(&policies)
			if err != nil {
				t.Fatal(err)
			}
			got := result.Values()[0]
			if got.Events != nil || got.DisableEvents != test.disabledEvents || got.DisableActors != test.disabledActors || (got.Actors != nil) != test.hasActors {
				t.Fatalf("lost presence: %#v", got)
			}
		})
	}
}

func TestActionsPoliciesRejectInvalidActorDeclarationsAtomically(t *testing.T) {
	for _, actor := range []uimodel.ActionsPolicyActor{
		{}, {User: actionsPolicyUIPtr(int64(1)), Role: actionsPolicyUIPtr("write")},
		{User: actionsPolicyUIPtr(int64(0))}, {User: actionsPolicyUIPtr(int64(-1))},
		{Role: actionsPolicyUIPtr("read")}, {Role: actionsPolicyUIPtr("admin")}, {Role: actionsPolicyUIPtr("")},
	} {
		bad := validActionsPolicyUI()
		bad.PolicyName = actionsPolicyUIPtr("bad")
		bad.RestrictActors = &uimodel.ActionsRestrictActors{AllowedActors: actionsPolicyUIPtr([]uimodel.ActionsPolicyActor{actor})}
		policies := []uimodel.ActionsEventPolicy{validActionsPolicyUI(), bad}
		result, err := ActionsPoliciesDeclaration(&policies)
		if err == nil || result != nil {
			t.Fatalf("partial/invalid conversion: %#v, %v", result, err)
		}
	}
	for _, restriction := range []*uimodel.ActionsRestrictActors{
		{}, {Enabled: actionsPolicyUIPtr(true)},
		{Enabled: actionsPolicyUIPtr(false), AllowedActors: actionsPolicyUIPtr([]uimodel.ActionsPolicyActor{})},
		{AllowedActors: actionsPolicyUIPtr([]uimodel.ActionsPolicyActor{{User: actionsPolicyUIPtr(int64(1))}, {User: actionsPolicyUIPtr(int64(1))}})},
		{AllowedActors: actionsPolicyUIPtr([]uimodel.ActionsPolicyActor{{Role: actionsPolicyUIPtr("write")}, {Role: actionsPolicyUIPtr("write")}})},
	} {
		p := validActionsPolicyUI()
		p.RestrictActors = restriction
		policies := []uimodel.ActionsEventPolicy{p}
		if result, err := ActionsPoliciesDeclaration(&policies); err == nil || result != nil {
			t.Fatal("accepted invalid restriction", restriction)
		}
	}
	p := validActionsPolicyUI()
	p.RestrictEvents.Enabled = actionsPolicyUIPtr(false)
	policies := []uimodel.ActionsEventPolicy{p}
	if _, err := ActionsPoliciesDeclaration(&policies); err == nil {
		t.Fatal("accepted events with disabled restriction")
	}
}

func TestActionsPolicyExportAbsentRestrictionsRoundTrips(t *testing.T) {
	for _, actors := range []*[]settings.ActionsPolicyActor{nil, actionsPolicyUIPtr([]settings.ActionsPolicyActor{}), actionsPolicyUIPtr([]settings.ActionsPolicyActor{{Type: "User", ID: 9}})} {
		for _, events := range [][]string{nil, {"push"}} {
			if actors == nil && events == nil {
				continue
			}
			original := settings.ActionsEventPolicy{ID: 17, Name: "export", Enforcement: "disabled", Actors: actors, Events: events}
			ui, err := ActionsEventPolicyUI(original)
			if err != nil {
				t.Fatal(err)
			}
			if (events == nil) != (ui.RestrictEvents.Enabled != nil && !*ui.RestrictEvents.Enabled) {
				t.Fatal("lost event off state")
			}
			if (actors == nil) != (ui.RestrictActors.Enabled != nil && !*ui.RestrictActors.Enabled) {
				t.Fatal("lost actor off state")
			}
			values := []uimodel.ActionsEventPolicy{ui}
			result, err := ActionsPoliciesDeclaration(&values)
			if err != nil {
				t.Fatal(err)
			}
			got := result.Values()[0]
			if !reflect.DeepEqual(got.Events, original.Events) || !reflect.DeepEqual(got.Actors, original.Actors) || got.DisableEvents != (events == nil) || got.DisableActors != (actors == nil) {
				t.Fatalf("lost exported restrictions: %#v", got)
			}
		}
	}
	none, err := ActionsEventPolicyUI(settings.ActionsEventPolicy{ID: 17, Name: "none", Enforcement: "active"})
	if err == nil || !reflect.DeepEqual(none, uimodel.ActionsEventPolicy{}) {
		t.Fatal("export accepted unrestricted state")
	}
	for _, actor := range []settings.ActionsPolicyActor{{Type: "Team", ID: 1}, {Type: "RepositoryRole", ID: 99}, {Type: "User", ID: 0}} {
		value := settings.ActionsEventPolicy{Name: "bad", Enforcement: "active", Actors: actionsPolicyUIPtr([]settings.ActionsPolicyActor{actor})}
		got, err := ActionsEventPolicyUI(value)
		if err == nil || !reflect.DeepEqual(got, uimodel.ActionsEventPolicy{}) {
			t.Fatal("partial/invalid export", got, err)
		}
	}
}
