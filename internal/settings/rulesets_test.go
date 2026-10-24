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
func TestRulesetsMembershipAndPresence(t *testing.T) {
	before := basicPolicy("default")
	observed := rsObserved(t, RulesetState{7, before})
	for _, tt := range []struct {
		name string
		d    *RulesetDeclarations
		op   string
	}{{"omitted", nil, ""}, {"empty", rsDesired(t), "delete"}, {"name only", rsDesired(t, RulesetDeclaration{Name: "default"}), ""}, {"empty mapping", rsDesired(t, RulesetDeclaration{Name: "default", Conditions: &RulesetConditions{RefName: &RefNameCondition{}}}), ""}, {"enforcement", rsDesired(t, RulesetDeclaration{Name: "default", Enforcement: rsText("disabled")}), "update"}} {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := ReconcileRulesets(observed, tt.d)
			if err != nil {
				t.Fatal(err)
			}
			if tt.op == "" {
				if len(changes) != 0 {
					t.Fatal(changes)
				}
			} else if len(changes) != 1 || changes[0].Operation() != tt.op || changes[0].ID() != 7 {
				t.Fatal(changes)
			}
		})
	}
	empty := []RulesetRule{}
	actors := []BypassActor{{ActorType: "Team", ActorID: rsID(9)}}
	d := RulesetDeclaration{Name: "default", Rules: &empty, BypassActors: &actors, Conditions: &RulesetConditions{RefName: &RefNameCondition{Exclude: rsList("refs/heads/dev*")}}}
	changes, err := ReconcileRulesets(observed, rsDesired(t, d))
	if err != nil || len(changes) != 1 {
		t.Fatalf("%v %v", changes, err)
	}
	after := changes[0].After()
	if len(*after.Rules) != 0 || *(*after.BypassActors)[0].BypassMode != "always" || !reflect.DeepEqual(*after.Conditions.RefName.Include, []string{"~DEFAULT_BRANCH"}) || *after.Enforcement != "active" {
		t.Fatal(after)
	}
	display := changes[0].Display()
	if display.Before().Rules == nil || display.Before().Conditions == nil || display.Before().Enforcement != nil || display.After().Rules == nil || display.After().Conditions == nil || display.After().Enforcement != nil {
		t.Fatal("display did not follow declaration presence", display.Before(), display.After())
	}
	repeated, err := ReconcileRulesets(rsObserved(t, RulesetState{7, *after}), rsDesired(t, d))
	if err != nil || len(repeated) != 0 {
		t.Fatalf("not idempotent: %v %v", repeated, err)
	}
	*after.Enforcement = "disabled"
	if *changes[0].After().Enforcement != "active" {
		t.Fatal("operation exposed pointer")
	}
}

func TestRulesetDisplayProjectionSeparatesMutationPreservation(t *testing.T) {
	before := reviewPolicy()
	disabled := "disabled"
	changes, err := ReconcileRulesets(rsObserved(t, RulesetState{7, before}), rsDesired(t, RulesetDeclaration{Name: "default", Enforcement: &disabled}))
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	mutation := changes[0].After()
	if mutation.Conditions == nil || mutation.BypassActors == nil || mutation.Rules == nil || len(*mutation.Rules) == 0 {
		t.Fatal("mutation lost unmanaged policy", mutation)
	}
	display := changes[0].Display()
	wantBefore := RulesetDeclaration{Name: "default", Enforcement: rsText("active")}
	wantAfter := RulesetDeclaration{Name: "default", Enforcement: rsText("disabled")}
	if !reflect.DeepEqual(display.Before(), &wantBefore) || !reflect.DeepEqual(display.After(), &wantAfter) {
		t.Fatalf("display leaked unmanaged policy: before=%+v after=%+v", display.Before(), display.After())
	}

	desired := RulesetDeclaration{Name: "default", Rules: &[]RulesetRule{{Type: "pull_request", Parameters: &RuleParameters{RequireCodeOwnerReview: paramBool(false), RequiredReviewers: &[]RequiredReviewer{}}}}}
	changes, err = ReconcileRulesets(rsObserved(t, RulesetState{7, before}), rsDesired(t, desired))
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	display = changes[0].Display()
	if display.Before().Target != nil || display.Before().Conditions != nil || display.Before().BypassActors != nil {
		t.Fatal("top-level omission leaked", display.Before())
	}
	beforePR := ruleOf(t, *display.Before(), "pull_request")
	afterPR := ruleOf(t, *display.After(), "pull_request")
	if beforePR.RequireCodeOwnerReview == nil || !*beforePR.RequireCodeOwnerReview || afterPR.RequireCodeOwnerReview == nil || *afterPR.RequireCodeOwnerReview || len(*afterPR.RequiredReviewers) != 0 {
		t.Fatal("false or empty declaration lost", display.Before(), display.After())
	}
	if beforePR.RequiredApprovingReviewCount != nil || beforePR.AllowedMergeMethods != nil {
		t.Fatal("omitted nested values leaked", beforePR)
	}
}
func TestRulesetsCreationValidationAndOrdering(t *testing.T) {
	current := rsObserved(t, RulesetState{4, basicPolicy("old")}, RulesetState{8, basicPolicy("z")})
	changes, err := ReconcileRulesets(current, rsDesired(t, basicPolicy("new"), basicPolicy("A")))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, c := range changes {
		names = append(names, c.Operation()+":"+c.Name())
	}
	if !reflect.DeepEqual(names, []string{"create:A", "create:new", "delete:old", "delete:z"}) {
		t.Fatal(names)
	}
	if _, err := ReconcileRulesets(current, rsDesired(t, RulesetDeclaration{Name: "missing"})); err == nil {
		t.Fatal("incomplete create accepted")
	}
	actors := []BypassActor{{ActorType: "Team", ActorID: rsID(10)}}
	if _, err := ReconcileRulesets(rsObserved(t, RulesetState{4, basicPolicy("old")}), rsDesired(t, RulesetDeclaration{Name: "old", BypassActors: &actors})); err == nil {
		t.Fatal("new bypass defaulted")
	}
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{{Name: "a"}, {Name: "A"}}); err != nil {
		t.Fatal("case-distinct names conflated")
	}
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{{Name: "a"}, {Name: "a"}}); err == nil {
		t.Fatal("duplicate accepted")
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
	if _, err := ReconcileRulesets(RulesetObservation{}, rsDesired(t)); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatal(err)
	}
	if _, err := NewRulesetObservation([]RulesetState{{1, basicPolicy("a")}, {2, basicPolicy("a")}}, 0); err == nil {
		t.Fatal("ambiguous names accepted")
	}
	for _, mutate := range []func(*RulesetDeclaration){func(d *RulesetDeclaration) { d.Rules = nil }, func(d *RulesetDeclaration) { d.BypassActors = nil }, func(d *RulesetDeclaration) { d.Conditions.RefName.Exclude = nil }} {
		d := basicPolicy("a")
		mutate(&d)
		if _, err := NewRulesetObservation([]RulesetState{{1, d}}, 0); err == nil {
			t.Fatal("incomplete observation accepted")
		}
	}
}
func TestRulesetsCanonicalCopiesAndNoop(t *testing.T) {
	d := basicPolicy("a")
	a := rsObserved(t, RulesetState{1, d})
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
