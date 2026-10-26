package settings

import (
	"strings"
	"testing"
)

func TestRulesetDeclarationIDsArePositiveUniqueAndOwned(t *testing.T) {
	for _, id := range []int64{0, -1} {
		policy := basicPolicy("existing")
		policy.ID = &id
		if _, err := NewRulesetDeclarations([]RulesetDeclaration{policy}); err == nil {
			t.Fatalf("accepted id %d", id)
		}
	}
	first, second := basicPolicy("first"), basicPolicy("second")
	first.ID, second.ID = rsID(7), rsID(7)
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{first, second}); err == nil {
		t.Fatal("accepted duplicate ID under different names")
	}
	declarations, err := NewRulesetDeclarations([]RulesetDeclaration{first})
	if err != nil {
		t.Fatal(err)
	}
	*first.ID = 8
	values := declarations.Values()
	if *values[0].ID != 7 {
		t.Fatal("declaration retained external ID pointer")
	}
	*values[0].ID = 9
	if *declarations.Values()[0].ID != 7 {
		t.Fatal("returned ID pointer changed declaration")
	}
}

func TestRulesetIDSelectionAndWholeObjectDefaults(t *testing.T) {
	current := rsObserved(t, RulesetState{ID: 7, Policy: basicPolicy("old")}, RulesetState{ID: 8, Policy: basicPolicy("unmanaged")})
	for _, empty := range []*RulesetDeclarations{nil, rsDesired(t)} {
		changes, err := ReconcileRulesets(RulesetObservation{}, empty)
		if err != nil || len(changes) != 0 {
			t.Fatal("empty selection acquired state or planned deletion", changes, err)
		}
	}
	minimal := RulesetDeclaration{ID: rsID(7), Name: "renamed", Enforcement: rsText("active"), Conditions: &RulesetConditions{RefName: &RefNameCondition{Include: rsList("~DEFAULT_BRANCH")}}}
	desired := rsDesired(t, minimal)
	changes, err := ReconcileRulesets(current, desired)
	if err != nil || len(changes) != 1 || changes[0].ID() != 7 || changes[0].Operation() != "update" {
		t.Fatal(changes, err)
	}
	after := changes[0].After()
	if after.Name != "renamed" || after.ID == nil || *after.ID != 7 || *after.Target != "branch" || len(*after.BypassActors) != 0 || len(*after.Rules) != 0 || len(*after.Conditions.RefName.Exclude) != 0 {
		t.Fatal("omission preserved remote values instead of fixed defaults", after)
	}
	if len(*changes[0].Display().Before().Rules) != 2 || changes[0].Display().After().ID == nil {
		t.Fatal("plan hid selected supported settings/default-derived changes")
	}
	repeated, err := ReconcileRulesets(rsObserved(t, RulesetState{ID: 7, Policy: *after}, RulesetState{ID: 8, Policy: basicPolicy("unmanaged")}), desired)
	if err != nil || len(repeated) != 0 {
		t.Fatal("repeat ID-selected comparison drifted", repeated, err)
	}
	created := minimal
	created.ID, created.Name = nil, "new"
	changes, err = ReconcileRulesets(current, rsDesired(t, created))
	if err != nil || len(changes) != 1 || changes[0].Operation() != "create" || changes[0].After().ID != nil || len(*changes[0].After().Rules) != 0 {
		t.Fatal("create defaults differ or unlisted objects deleted", changes, err)
	}
}

func TestRulesetIDSelectionNeverAdoptsOrFallsBack(t *testing.T) {
	current := rsObserved(t, RulesetState{ID: 7, Policy: basicPolicy("existing")}, RulesetState{ID: 8, Policy: basicPolicy("other")})
	for _, test := range []struct {
		name string
		id   *int64
		want string
	}{
		{"existing", nil, "specify id: 7"},
		{"new", rsID(99), "refusing create fallback"},
		{"other", rsID(7), "refusing rename collision"},
	} {
		policy := basicPolicy(test.name)
		policy.ID = test.id
		changes, err := ReconcileRulesets(current, rsDesired(t, policy))
		if err == nil || !strings.Contains(err.Error(), test.want) || len(changes) != 0 {
			t.Fatalf("test=%+v changes=%v error=%v", test, changes, err)
		}
	}
}
