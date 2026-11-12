package settings

import "testing"

func TestActionsPolicyPrunePresenceIdentityAndUnsupported(t *testing.T) {
	policy := ActionsEventPolicy{ID: 7, Name: "old", Enforcement: "active", Events: []string{"push"}}
	observed, _ := NewActionsPolicyObservation([]ActionsEventPolicyState{{ID: 7, Name: "old", Policy: &policy}})
	empty, _ := NewActionsPolicyDeclarations(nil)
	for _, tc := range []struct {
		d    *ActionsPolicyDeclarations
		p    bool
		want int
	}{{nil, true, 0}, {&empty, false, 0}, {&empty, true, 1}} {
		got, err := ReconcileActionsPoliciesWithPrune(observed, tc.d, tc.p)
		if err != nil || len(got) != tc.want {
			t.Fatal(got, err)
		}
	}
	if _, err := ReconcileActionsPoliciesWithPrune(ActionsPolicyObservation{}, &empty, true); err == nil {
		t.Fatal("missing observation accepted")
	}
	unsupported, _ := NewActionsPolicyObservation([]ActionsEventPolicyState{{ID: 8, Name: "unknown", Unsupported: "future protection"}})
	if _, err := ReconcileActionsPoliciesWithPrune(unsupported, &empty, true); err == nil {
		t.Fatal("unsupported deletion accepted")
	}
	desired, _ := NewActionsPolicyDeclarations([]ActionsEventPolicy{policy, {Name: "new", Enforcement: "disabled", Events: []string{"push"}}})
	got, err := ReconcileActionsPoliciesWithPrune(observed, &desired, true)
	if err != nil || len(got) != 1 || got[0].Operation() != "create" {
		t.Fatal(got, err)
	}
	got, err = ReconcileActionsPoliciesWithPrune(observed, &empty, true)
	if err != nil || got[0].ID() != 7 || got[0].Name() != "old" || got[0].Operation() != "delete" || got[0].Before() == nil {
		t.Fatal(got, err)
	}
	if _, err := ParsePruneScopes("actions-policies"); err != nil {
		t.Fatal(err)
	}
}
