package settings

import (
	"reflect"
	"strings"
	"testing"
)

func eventPolicy(id int64, name string) ActionsEventPolicy {
	return ActionsEventPolicy{ID: id, Name: name, Enforcement: "active", Events: []string{"push", "pull_request"}}
}
func eventObservation(t *testing.T, policies ...ActionsEventPolicy) ActionsPolicyObservation {
	t.Helper()
	states := make([]ActionsEventPolicyState, len(policies))
	for i, p := range policies {
		p := p
		states[i] = ActionsEventPolicyState{ID: p.ID, Name: p.Name, Policy: &p}
	}
	o, err := NewActionsPolicyObservation(states)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func eventDeclarations(t *testing.T, policies ...ActionsEventPolicy) *ActionsPolicyDeclarations {
	t.Helper()
	d, err := NewActionsPolicyDeclarations(policies)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

func TestActionsPolicyDeclarationValidation(t *testing.T) {
	good := eventPolicy(0, "Build")
	cases := map[string][]ActionsEventPolicy{"negative id": {{ID: -1, Name: "Build", Enforcement: "active", Events: []string{"push"}}}, "blank name": {eventPolicy(0, " ")}, "control name": {eventPolicy(0, "x\ny")}, "duplicate name": {good, good}, "duplicate id": {eventPolicy(2, "one"), eventPolicy(2, "two")}}
	invalid := []ActionsEventPolicy{good, good, good, good, good}
	invalid[0].Enforcement = "evaluate"
	invalid[1].Events = nil
	invalid[2].Events = []string{}
	invalid[3].Events = []string{"unknown"}
	invalid[4].Events = []string{"push", "push"}
	for i, p := range invalid {
		cases[string(rune('a'+i))] = []ActionsEventPolicy{p}
	}
	for name, values := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewActionsPolicyDeclarations(values); err == nil {
				t.Fatal("accepted invalid declarations")
			}
		})
	}
	all := good
	all.Events = SupportedActionsPolicyEvents()
	if len(all.Events) != 36 {
		t.Fatalf("event vocabulary count %d", len(all.Events))
	}
	if err := ValidateActionsEventPolicy(all); err != nil {
		t.Fatal(err)
	}
	all.Events[0] = "corrupted"
	if SupportedActionsPolicyEvents()[0] == "corrupted" {
		t.Fatal("vocabulary aliased")
	}
}

func TestActionsPolicyOwnedValues(t *testing.T) {
	input := eventPolicy(4, "Build")
	d := eventDeclarations(t, input)
	input.Events[0] = "release"
	if d.Values()[0].Events[1] != "push" {
		t.Fatal("constructor aliased declaration events")
	}
	value := d.Values()
	value[0].Events[0] = "release"
	if d.Values()[0].Events[0] != "pull_request" {
		t.Fatal("declaration accessor alias")
	}
	p := eventPolicy(4, "Build")
	o := eventObservation(t, p)
	p.Events[0] = "release"
	states, _ := o.Values()
	states[0].Policy.Events[0] = "release"
	again, _ := o.Values()
	if again[0].Policy.Events[0] != "pull_request" {
		t.Fatal("observation alias")
	}
	after := eventPolicy(4, "Build new")
	c, err := NewActionsEventPolicyChange(again[0].Policy, after)
	if err != nil {
		t.Fatal(err)
	}
	after.Events[0] = "release"
	c.Before().Events[0] = "release"
	c.After().Events[0] = "release"
	if c.Before().Events[0] != "pull_request" || c.After().Events[0] != "pull_request" {
		t.Fatal("change alias")
	}
	if c.Operation() != "update" || c.ID() != 4 || c.Name() != "Build new" {
		t.Fatal("wrong change metadata")
	}
}

func TestActionsPolicyReconciliation(t *testing.T) {
	observed := eventObservation(t, eventPolicy(1, "Build"), eventPolicy(2, "Release"))
	cases := []struct {
		name       string
		desired    []ActionsEventPolicy
		wantErr    string
		operations []string
	}{
		{name: "omitted"}, {name: "empty", desired: []ActionsEventPolicy{}},
		{name: "reordered events unchanged", desired: []ActionsEventPolicy{eventPolicy(1, "Build")}},
		{name: "create", desired: []ActionsEventPolicy{eventPolicy(0, "New")}, operations: []string{"create"}},
		{name: "rename", desired: []ActionsEventPolicy{eventPolicy(1, "New")}, operations: []string{"update"}},
		{name: "no name adoption", desired: []ActionsEventPolicy{eventPolicy(0, "Build")}, wantErr: "already belongs"},
		{name: "missing id", desired: []ActionsEventPolicy{eventPolicy(3, "New")}, wantErr: "does not exist"},
		{name: "rename collision", desired: []ActionsEventPolicy{eventPolicy(1, "Release")}, wantErr: "already belongs"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var desired *ActionsPolicyDeclarations
			if tc.desired != nil {
				desired = eventDeclarations(t, tc.desired...)
			}
			changes, err := ReconcileActionsPolicies(observed, desired)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var operations []string
			for _, c := range changes {
				operations = append(operations, c.Operation())
				if c.Operation() == "create" && (c.Before() != nil || c.ID() != 0) {
					t.Fatal("invalid creation")
				}
			}
			if !reflect.DeepEqual(operations, tc.operations) {
				t.Fatalf("operations %v want %v", operations, tc.operations)
			}
		})
	}
	p := eventPolicy(1, "Build")
	p.Enforcement = "disabled"
	p.Events = []string{"release"}
	comparison, err := CompareActionsPolicies(observed, eventDeclarations(t, p))
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Managed != 1 || !comparison.HasChanges() || len(comparison.ActionsPolicies) != 1 {
		t.Fatalf("comparison %#v", comparison)
	}
	if !reflect.DeepEqual(comparison.ActionsPolicies[0].After(), p) {
		t.Fatal("not whole-object update")
	}
	unchanged, err := CompareActionsPolicies(observed, eventDeclarations(t, eventPolicy(1, "Build")))
	if err != nil || unchanged.Managed != 1 || unchanged.HasChanges() {
		t.Fatalf("unchanged %#v %v", unchanged, err)
	}
}

func TestActionsPolicyObservationAndUnsupportedIsolation(t *testing.T) {
	if _, err := (ActionsPolicyObservation{}).Values(); err == nil {
		t.Fatal("missing observation accepted")
	}
	if _, err := CompareActionsPolicies(ActionsPolicyObservation{}, eventDeclarations(t, eventPolicy(0, "New"))); err == nil {
		t.Fatal("missing observation compared")
	}
	empty, err := NewActionsPolicyObservation(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := empty.Values(); err != nil {
		t.Fatal(err)
	}
	p := eventPolicy(1, "Build")
	states := []ActionsEventPolicyState{{ID: 1, Name: p.Name, Policy: &p}, {ID: 2, Name: "Unsupported", Unsupported: "workflow targeting"}, {ID: 3, Name: "Unsupported", Unsupported: "inherited policy"}}
	o, err := NewActionsPolicyObservation(states)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReconcileActionsPolicies(o, eventDeclarations(t, eventPolicy(1, "Build new"))); err != nil {
		t.Fatalf("unrelated unsupported prevented management: %v", err)
	}
	if _, err := ReconcileActionsPolicies(o, eventDeclarations(t, eventPolicy(2, "New"))); err == nil || !strings.Contains(err.Error(), "workflow targeting") {
		t.Fatalf("selected unsupported error %v", err)
	}
	if _, err := ReconcileActionsPolicies(o, eventDeclarations(t, eventPolicy(0, "Unsupported"))); err == nil {
		t.Fatal("unsupported name adopted")
	}
	if _, err := ExportActionsPolicies(o); err == nil {
		t.Fatal("lossy export accepted")
	}
	for name, values := range map[string][]ActionsEventPolicyState{
		"bad id": {{Name: "bad", Unsupported: "reason"}}, "duplicate id": {states[0], states[0]}, "missing shape": {{ID: 2, Name: "bad"}}, "two shapes": {{ID: 1, Name: p.Name, Policy: &p, Unsupported: "reason"}}, "mismatch": {{ID: 2, Name: p.Name, Policy: &p}}, "bad name": {{ID: 2, Name: "", Unsupported: "reason"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewActionsPolicyObservation(values); err == nil {
				t.Fatal("invalid observation accepted")
			}
		})
	}
}

func TestActionsPolicyExportAndChangeValidation(t *testing.T) {
	exported, err := ExportActionsPolicies(eventObservation(t, eventPolicy(2, "Z"), eventPolicy(1, "A")))
	if err != nil {
		t.Fatal(err)
	}
	if exported.Values()[0].Name != "A" || exported.Values()[0].ID != 1 {
		t.Fatal("export identity/order")
	}
	duplicate := eventPolicy(2, "A")
	if _, err := ExportActionsPolicies(eventObservation(t, eventPolicy(1, "A"), duplicate)); err == nil {
		t.Fatal("ambiguous export accepted")
	}
	before := eventPolicy(1, "A")
	for _, tc := range []struct {
		before *ActionsEventPolicy
		after  ActionsEventPolicy
	}{{nil, before}, {&before, before}, {&before, eventPolicy(2, "B")}} {
		if _, err := NewActionsEventPolicyChange(tc.before, tc.after); err == nil {
			t.Fatal("invalid change accepted")
		}
	}
	if !(Declaration{}).Empty() {
		t.Fatal("zero declaration nonempty")
	}
	if !(Declaration{ActionsPolicies: eventDeclarations(t)}).Empty() {
		t.Fatal("empty selection nonempty")
	}
	if (Declaration{ActionsPolicies: eventDeclarations(t, eventPolicy(0, "A"))}).Empty() {
		t.Fatal("policy declaration empty")
	}
}
