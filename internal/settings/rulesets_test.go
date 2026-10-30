package settings

import (
	"reflect"
	"strings"
	"testing"
)

func rsText(v string) *string      { return &v }
func rsID(v int64) *int64          { return &v }
func rsList(v ...string) *[]string { out := append([]string{}, v...); return &out }
func basicPolicy(name string) RulesetDeclaration {
	actors := []BypassActor{{ActorType: "Team", ActorID: rsID(9), BypassMode: rsText("always")}}
	rules := []RulesetRule{{Type: "deletion"}, {Type: "non_fast_forward"}}
	return RulesetDeclaration{Name: name, Target: rsText("branch"), Enforcement: rsText("active"), Conditions: &RulesetConditions{RefName: &RefNameCondition{Include: rsList("~DEFAULT_BRANCH"), Exclude: rsList()}}, BypassActors: &actors, Rules: &rules}
}
func rsDesired(t *testing.T, values ...RulesetDeclaration) *RulesetDeclarations {
	t.Helper()
	d, e := NewRulesetDeclarations(values)
	if e != nil {
		t.Fatal(e)
	}
	return &d
}
func rsObserved(t *testing.T, values ...RulesetState) RulesetObservation {
	t.Helper()
	o, e := NewRulesetObservation(values, 0)
	if e != nil {
		t.Fatal(e)
	}
	return o
}
func TestRulesetsRequireProtectionIntent(t *testing.T) {
	for _, change := range []func(*RulesetDeclaration){
		func(d *RulesetDeclaration) { d.Enforcement = nil },
		func(d *RulesetDeclaration) { d.Conditions = nil },
		func(d *RulesetDeclaration) { d.Conditions.RefName.Include = nil },
		func(d *RulesetDeclaration) { (*d.BypassActors)[0].BypassMode = nil },
	} {
		for _, id := range []*int64{nil, rsID(7)} {
			d := basicPolicy("selected")
			d.ID = id
			change(&d)
			if _, err := NewRulesetDeclarations([]RulesetDeclaration{d}); err == nil {
				t.Fatal("guessed protection intent", d)
			}
		}
	}
}
func TestRulesetsCreationValidationAndOrdering(t *testing.T) {
	current := rsObserved(t, RulesetState{ID: 4, Policy: basicPolicy("old")}, RulesetState{ID: 8, Policy: basicPolicy("z")})
	changes, err := ReconcileRulesets(current, rsDesired(t, basicPolicy("new"), basicPolicy("A")))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, change := range changes {
		names = append(names, change.Operation()+":"+change.Name())
	}
	if !reflect.DeepEqual(names, []string{"create:A", "create:new"}) {
		t.Fatal(names)
	}
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{{Name: "missing"}}); err == nil {
		t.Fatal("incomplete declaration accepted")
	}
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{basicPolicy("a"), basicPolicy("A")}); err != nil {
		t.Fatal("case-distinct names conflated")
	}
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{basicPolicy("a"), basicPolicy("a")}); err == nil {
		t.Fatal("duplicate names accepted")
	}
}
func TestRulesetsRejectInvalidAndIncompleteState(t *testing.T) {
	for _, mutate := range []func(*RulesetDeclaration){func(d *RulesetDeclaration) { d.Name = " " }, func(d *RulesetDeclaration) { d.Name = "x\n" }, func(d *RulesetDeclaration) { d.Target = rsText("push") }, func(d *RulesetDeclaration) { d.Enforcement = rsText("enabled") }, func(d *RulesetDeclaration) { d.Rules = &[]RulesetRule{{Type: "merge_queue"}} }, func(d *RulesetDeclaration) { d.Rules = &[]RulesetRule{{Type: "deletion"}, {Type: "deletion"}} }, func(d *RulesetDeclaration) { d.BypassActors = &[]BypassActor{{ActorType: "Team", ActorID: rsID(0)}} }, func(d *RulesetDeclaration) {
		d.BypassActors = &[]BypassActor{{ActorType: "DeployKey", ActorID: rsID(4)}}
	}, func(d *RulesetDeclaration) {
		d.Target = rsText("tag")
		d.BypassActors = &[]BypassActor{{ActorType: "Team", ActorID: rsID(1), BypassMode: rsText("pull_request")}}
	}} {
		d := basicPolicy("a")
		mutate(&d)
		if _, err := NewRulesetDeclarations([]RulesetDeclaration{d}); err == nil {
			t.Fatalf("accepted %+v", d)
		}
	}
	if _, err := ReconcileRulesets(RulesetObservation{}, rsDesired(t, basicPolicy("selected"))); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatal(err)
	}
	if _, err := NewRulesetObservation([]RulesetState{{ID: 1, Policy: basicPolicy("a")}, {ID: 2, Policy: basicPolicy("a")}}, 0); err == nil {
		t.Fatal("ambiguous names accepted")
	}
	for _, mutate := range []func(*RulesetDeclaration){func(d *RulesetDeclaration) { d.Rules = nil }, func(d *RulesetDeclaration) { d.BypassActors = nil }, func(d *RulesetDeclaration) { d.Conditions.RefName.Exclude = nil }} {
		d := basicPolicy("a")
		mutate(&d)
		if _, err := NewRulesetObservation([]RulesetState{{ID: 1, Policy: d}}, 0); err == nil {
			t.Fatal("incomplete observation accepted")
		}
	}
}
func TestRulesetsCanonicalCopiesAndNoop(t *testing.T) {
	d := basicPolicy("a")
	a := rsObserved(t, RulesetState{ID: 1, Policy: d})
	d.ID = rsID(1)
	desired := rsDesired(t, d)
	(*d.Rules)[0].Type = "required_signatures"
	changes, err := ReconcileRulesets(a, desired)
	if err != nil || len(changes) != 0 {
		t.Fatalf("copy changed: %v %v", changes, err)
	}
	values := desired.Values()
	*values[0].Target = "tag"
	obs, _ := a.Values()
	obs[0].Policy.Conditions.RefName.Include = rsList("~ALL")
	changes, err = ReconcileRulesets(a, desired)
	if err != nil || len(changes) != 0 {
		t.Fatal("exposed nested values")
	}
}
