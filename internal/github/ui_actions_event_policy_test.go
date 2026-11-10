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
		{"missing restriction", func(p *uimodel.ActionsEventPolicy) { p.RestrictEvents = nil }},
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
	if err != nil || !reflect.DeepEqual(again.ActionsPolicies.Values(), policies.Values()) || again.Variables.Len() != 1 || *again.Actions.RetentionDays != days {
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
