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

func actorList(actors ...ActionsPolicyActor) *[]ActionsPolicyActor { return &actors }

func TestActionsActorPolicyValidation(t *testing.T) {
	valid := ActionsEventPolicy{Name: "Actors", Enforcement: "active", Actors: actorList(
		ActionsPolicyActor{Type: "User", ID: 2},
		ActionsPolicyActor{Type: "RepositoryRole", ID: 2},
		ActionsPolicyActor{Type: "RepositoryRole", ID: 4},
		ActionsPolicyActor{Type: "RepositoryRole", ID: 5},
	)}
	for _, actors := range []*[]ActionsPolicyActor{valid.Actors, actorList()} {
		p := valid
		p.Actors = actors
		if err := ValidateActionsEventPolicy(p); err != nil {
			t.Fatal(err)
		}
	}
	for name, actors := range map[string]*[]ActionsPolicyActor{
		"zero user":     actorList(ActionsPolicyActor{Type: "User"}),
		"negative user": actorList(ActionsPolicyActor{Type: "User", ID: -1}),
		"role zero":     actorList(ActionsPolicyActor{Type: "RepositoryRole"}),
		"role one":      actorList(ActionsPolicyActor{Type: "RepositoryRole", ID: 1}),
		"role three":    actorList(ActionsPolicyActor{Type: "RepositoryRole", ID: 3}),
		"role six":      actorList(ActionsPolicyActor{Type: "RepositoryRole", ID: 6}),
		"lowercase":     actorList(ActionsPolicyActor{Type: "user", ID: 2}),
		"team":          actorList(ActionsPolicyActor{Type: "Team", ID: 2}),
		"duplicate":     actorList(ActionsPolicyActor{Type: "User", ID: 2}, ActionsPolicyActor{Type: "User", ID: 2}),
	} {
		t.Run(name, func(t *testing.T) {
			p := valid
			p.Actors = actors
			if err := ValidateActionsEventPolicy(p); err == nil {
				t.Fatal("accepted invalid actor")
			}
		})
	}
	for _, mutate := range []func(*ActionsEventPolicy){
		func(p *ActionsEventPolicy) { p.DisableActors = true },
		func(p *ActionsEventPolicy) { p.Events = []string{"push"}; p.DisableEvents = true },
		func(p *ActionsEventPolicy) { p.Events = []string{} },
		func(p *ActionsEventPolicy) { p.Actors = nil },
	} {
		p := valid
		mutate(&p)
		if err := ValidateActionsEventPolicy(p); err == nil {
			t.Fatalf("accepted invalid declaration %#v", p)
		}
	}
}

func TestActionsActorPolicyOwnershipAndCanonicalIdentity(t *testing.T) {
	p := eventPolicy(1, "Combined")
	p.Actors = actorList(ActionsPolicyActor{Type: "User", ID: 8}, ActionsPolicyActor{Type: "RepositoryRole", ID: 4}, ActionsPolicyActor{Type: "User", ID: 3})
	original := cloneActionsEventPolicy(p)
	d := eventDeclarations(t, p)
	o := eventObservation(t, p)
	(*p.Actors)[0].ID = 99
	*d.Values()[0].Actors = nil
	observed, _ := o.Values()
	(*observed[0].Policy.Actors)[0].ID = 99
	again, _ := o.Values()
	want := []ActionsPolicyActor{{Type: "RepositoryRole", ID: 4}, {Type: "User", ID: 3}, {Type: "User", ID: 8}}
	if !reflect.DeepEqual(*d.Values()[0].Actors, want) || !reflect.DeepEqual(*again[0].Policy.Actors, want) {
		t.Fatal("actor ownership or canonical order lost")
	}
	changes, err := ReconcileActionsPolicies(o, eventDeclarations(t, original))
	if err != nil || len(changes) != 0 {
		t.Fatalf("order produced drift: %v %v", changes, err)
	}
	original.Name = "Renamed"
	change, err := NewActionsEventPolicyChange(again[0].Policy, original)
	if err != nil {
		t.Fatal(err)
	}
	(*original.Actors)[0].ID = 99
	(*change.Before().Actors)[0].ID = 99
	(*change.After().Actors)[0].ID = 99
	if !reflect.DeepEqual(*change.Before().Actors, want) || !reflect.DeepEqual(*change.After().Actors, want) {
		t.Fatal("change actor alias")
	}
	empty := eventPolicy(1, "Combined")
	empty.Actors = actorList()
	if equalActionsEventPolicies(eventPolicy(1, "Combined"), empty) {
		t.Fatal("absent and zero-allowlist actor rules compared equal")
	}
}

func TestActionsPolicyRuleResolution(t *testing.T) {
	combined := eventPolicy(1, "Combined")
	combined.Actors = actorList(ActionsPolicyActor{Type: "User", ID: 7})
	cases := []struct {
		name        string
		before      ActionsEventPolicy
		after       ActionsEventPolicy
		wantActors  bool
		wantEvents  bool
		wantChanges int
		wantErr     bool
	}{
		{name: "omitted actors preserved", before: combined, after: eventPolicy(1, "Renamed"), wantActors: true, wantEvents: true, wantChanges: 1},
		{name: "omitted events preserved", before: combined, after: ActionsEventPolicy{ID: 1, Name: "Renamed", Enforcement: "active", Actors: actorList()}, wantActors: true, wantEvents: true, wantChanges: 1},
		{name: "omitted actors no drift", before: combined, after: eventPolicy(1, "Combined"), wantActors: true, wantEvents: true},
		{name: "omitted events no drift", before: combined, after: ActionsEventPolicy{ID: 1, Name: "Combined", Enforcement: "active", Actors: combined.Actors}, wantActors: true, wantEvents: true},
		{name: "disable actors", before: combined, after: ActionsEventPolicy{ID: 1, Name: "Combined", Enforcement: "active", DisableActors: true}, wantEvents: true, wantChanges: 1},
		{name: "disable events", before: combined, after: ActionsEventPolicy{ID: 1, Name: "Combined", Enforcement: "active", DisableEvents: true}, wantActors: true, wantChanges: 1},
		{name: "last event rule", before: eventPolicy(1, "Combined"), after: ActionsEventPolicy{ID: 1, Name: "Combined", Enforcement: "active", DisableEvents: true}, wantErr: true},
		{name: "last actor rule", before: ActionsEventPolicy{ID: 1, Name: "Combined", Enforcement: "active", Actors: actorList()}, after: ActionsEventPolicy{ID: 1, Name: "Combined", Enforcement: "active", DisableActors: true}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changes, err := ReconcileActionsPolicies(eventObservation(t, tc.before), eventDeclarations(t, tc.after))
			if tc.wantErr {
				if err == nil {
					t.Fatal("accepted ruleless effective policy")
				}
				if _, err := NewActionsEventPolicyChange(&tc.before, tc.after); err == nil {
					t.Fatal("direct constructor accepted ruleless state")
				}
				return
			}
			if err != nil || len(changes) != tc.wantChanges {
				t.Fatalf("changes %v error %v", changes, err)
			}
			if len(changes) > 0 {
				after := changes[0].After()
				if (after.Actors != nil) != tc.wantActors || (after.Events != nil) != tc.wantEvents || after.DisableActors || after.DisableEvents {
					t.Fatalf("wrong resolved state %#v", after)
				}
			}
		})
	}
}

func TestActionsActorPolicyCreateObservationAndExport(t *testing.T) {
	for _, p := range []ActionsEventPolicy{
		{Name: "Actors", Enforcement: "active", Actors: actorList()},
		{Name: "Combined", Enforcement: "disabled", Actors: actorList(ActionsPolicyActor{Type: "User", ID: 1}), Events: []string{"push"}},
		eventPolicy(0, "Events"),
	} {
		change, err := NewActionsEventPolicyChange(nil, p)
		if err != nil {
			t.Fatal(err)
		}
		if change.Operation() != "create" || (change.After().Actors == nil) != (p.Actors == nil) || (change.After().Events == nil) != (p.Events == nil) {
			t.Fatal("wrong creation")
		}
		p.ID = 4
		exported, err := ExportActionsPolicies(eventObservation(t, p))
		if err != nil {
			t.Fatal(err)
		}
		actual := exported.Values()[0]
		if actual.ID != p.ID || (actual.Actors == nil) != (p.Actors == nil) || (actual.Events == nil) != (p.Events == nil) || actual.DisableActors || actual.DisableEvents {
			t.Fatalf("lossy export %#v", actual)
		}
	}
	for _, p := range []ActionsEventPolicy{
		{Name: "Remove", Enforcement: "active", DisableActors: true},
		{Name: "Remove", Enforcement: "active", DisableEvents: true},
	} {
		if _, err := NewActionsEventPolicyChange(nil, p); err == nil {
			t.Fatal("created ruleless policy")
		}
		p.ID = 1
		if _, err := NewActionsPolicyObservation([]ActionsEventPolicyState{{ID: 1, Name: p.Name, Policy: &p}}); err == nil {
			t.Fatal("observation accepted declaration intent")
		}
	}
}

func TestActionsPolicyMetadataOnlyUpdatePreservesRules(t *testing.T) {
	before := eventPolicy(1, "Before")
	before.Actors = actorList()
	after := ActionsEventPolicy{ID: 1, Name: "After", Enforcement: "disabled"}
	if err := ValidateActionsEventPolicy(after); err != nil {
		t.Fatal(err)
	}
	change, err := NewActionsEventPolicyChange(&before, after)
	if err != nil {
		t.Fatal(err)
	}
	if change.After().Actors == nil || change.After().Events == nil {
		t.Fatal("metadata update cleared rules")
	}
	if err := ValidateActionsEventPolicyState(after); err == nil {
		t.Fatal("ruleless observed state accepted")
	}
	after.Name, after.Enforcement = before.Name, before.Enforcement
	changes, err := ReconcileActionsPolicies(eventObservation(t, before), eventDeclarations(t, after))
	if err != nil || len(changes) != 0 {
		t.Fatalf("metadata only no-op: %v %v", changes, err)
	}
}

func TestActionsActorRuleDriftUsesCompleteIdentityAndPresence(t *testing.T) {
	before := eventPolicy(1, "Build")
	for _, actors := range []*[]ActionsPolicyActor{
		actorList(),
		actorList(ActionsPolicyActor{Type: "User", ID: 2}),
		actorList(ActionsPolicyActor{Type: "RepositoryRole", ID: 2}),
	} {
		after := eventPolicy(1, "Build")
		after.Actors = actors
		changes, err := ReconcileActionsPolicies(eventObservation(t, before), eventDeclarations(t, after))
		if err != nil || len(changes) != 1 {
			t.Fatalf("actor drift missing: %v %v", changes, err)
		}
		before = after
	}
	after := eventPolicy(1, "Build")
	after.DisableActors = true
	changes, err := ReconcileActionsPolicies(eventObservation(t, before), eventDeclarations(t, after))
	if err != nil || len(changes) != 1 || changes[0].After().Actors != nil {
		t.Fatalf("actor removal missing: %v %v", changes, err)
	}
}

func TestActionsPolicyChangeRejectsInvalidBaseline(t *testing.T) {
	before := ActionsEventPolicy{ID: 1, Name: "Build", Enforcement: "active"}
	if _, err := NewActionsEventPolicyChange(&before, eventPolicy(1, "Build")); err == nil {
		t.Fatal("accepted ruleless baseline")
	}
	before = eventPolicy(1, "Build")
	before.DisableActors = true
	if _, err := NewActionsEventPolicyChange(&before, eventPolicy(1, "Build")); err == nil {
		t.Fatal("accepted baseline with declaration intent")
	}
}

func TestActionsPolicyDeclarationCannotDisableEveryRule(t *testing.T) {
	for _, id := range []int64{0, 1} {
		p := ActionsEventPolicy{ID: id, Name: "Build", Enforcement: "active", DisableActors: true, DisableEvents: true}
		if err := ValidateActionsEventPolicy(p); err == nil {
			t.Fatal("accepted removal of every rule")
		}
		if _, err := NewActionsPolicyDeclarations([]ActionsEventPolicy{p}); err == nil {
			t.Fatal("accepted removal of every rule in declarations")
		}
	}
}

func TestActionsPolicyStateChangeReconstructsRemoval(t *testing.T) {
	before := eventPolicy(1, "Build")
	before.Actors = actorList(ActionsPolicyActor{Type: "User", ID: 9})
	for name, after := range map[string]ActionsEventPolicy{
		"remove actors": eventPolicy(1, "Build"),
		"remove events": {ID: 1, Name: "Build", Enforcement: "active", Actors: actorList()},
	} {
		t.Run(name, func(t *testing.T) {
			change, err := NewActionsEventPolicyStateChange(&before, after)
			if err != nil {
				t.Fatal(err)
			}
			actual := change.After()
			if (actual.Actors == nil) != (after.Actors == nil) || (actual.Events == nil) != (after.Events == nil) {
				t.Fatalf("reconstruction restored removed rule: %#v", actual)
			}
			reconstructed, err := NewActionsEventPolicyStateChange(change.Before(), actual)
			if err != nil || !equalActionsEventPolicies(reconstructed.After(), actual) {
				t.Fatalf("state reconstruction not stable: %#v %v", reconstructed, err)
			}
		})
	}
	for _, after := range []ActionsEventPolicy{
		{ID: 1, Name: "Build", Enforcement: "active"},
		{ID: 1, Name: "Build", Enforcement: "active", DisableActors: true, Events: []string{"push"}},
	} {
		if _, err := NewActionsEventPolicyStateChange(&before, after); err == nil {
			t.Fatal("state constructor accepted missing restrictions or disable intent")
		}
	}
}

func TestActionsPolicyStateChangeUsesCompleteTargeting(t *testing.T) {
	before := eventPolicy(1, "Build")
	before.TargetWorkflows = &ActionsWorkflowTarget{Include: []string{".github/workflows/build.yml"}}
	after := eventPolicy(1, "Build")
	change, err := NewActionsEventPolicyStateChange(&before, after)
	if err != nil {
		t.Fatal(err)
	}
	if !EqualActionsWorkflowTargets(change.After().TargetWorkflows, nil) {
		t.Fatal("state reconstruction preserved omitted targeting")
	}
	created, err := NewActionsEventPolicyStateChange(nil, eventPolicy(0, "New"))
	if err != nil || created.Operation() != "create" {
		t.Fatalf("state creation: %#v %v", created, err)
	}
}
