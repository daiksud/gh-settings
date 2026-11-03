package settings

import (
	"reflect"
	"testing"
)

func TestRulesetEnforcementChangesOwnActionableWholeObjects(t *testing.T) {
	for _, raw := range []string{"FUTURE_ENFORCEMENT", "FUTURE_MODE", "", " \t"} {
		for _, desired := range []string{"active", "disabled"} {
			state := normalizedEnforcementState(7, "policy", raw)
			before := state.Policy
			before.ID = rsID(7)
			after := cloneRuleset(before)
			after.Enforcement = rsText(desired)
			change, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, RulesetNormalizations{Enforcement: state.EnforcementNormalization})
			if err != nil || change.Operation() != "update" || change.ID() != 7 || change.Normalizations().Enforcement != state.EnforcementNormalization {
				t.Fatal(change, err)
			}
			display := change.Display()
			if display.Operation() != "update" || display.Normalizations().Enforcement != state.EnforcementNormalization || *display.Before().Enforcement != "disabled" || *display.After().Enforcement != desired {
				t.Fatal(display)
			}
			before.Name = "mutated caller"
			*after.Enforcement = "FUTURE_ENFORCEMENT"
			returned := change.Before()
			returned.Name = "mutated result"
			*returned.Enforcement = "active"
			displayed := display.After()
			*displayed.Enforcement = "FUTURE_ENFORCEMENT"
			if change.Name() != "policy" || *change.Before().Enforcement != "disabled" || *change.After().Enforcement != desired || *display.After().Enforcement != desired {
				t.Fatal("policy ownership lost", change, display)
			}
			if desired == "disabled" {
				if _, err := NewRulesetChange("update", 7, change.Before(), change.After()); err == nil {
					t.Fatal("ordinary equality became actionable")
				}
				if _, err := NewRulesetDisplayChange("update", display.Before(), display.After()); err == nil {
					t.Fatal("ordinary display equality became actionable")
				}
			}
		}
	}
}

func TestRulesetEnforcementChangeRejectsForgedOrUnboundPolicies(t *testing.T) {
	for _, mutate := range []func(*string, *int64, **RulesetDeclaration, **RulesetDeclaration, *ObservationNormalization){
		func(op *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, _ *ObservationNormalization) {
			*op = "create"
		},
		func(_ *string, id *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, _ *ObservationNormalization) {
			*id = 0
		},
		func(_ *string, _ *int64, before **RulesetDeclaration, _ **RulesetDeclaration, _ *ObservationNormalization) {
			*before = nil
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, after **RulesetDeclaration, _ *ObservationNormalization) {
			*after = nil
		},
		func(_ *string, _ *int64, before **RulesetDeclaration, _ **RulesetDeclaration, _ *ObservationNormalization) {
			(*before).ID = rsID(8)
		},
		func(_ *string, _ *int64, before **RulesetDeclaration, _ **RulesetDeclaration, _ *ObservationNormalization) {
			(*before).ID = nil
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, after **RulesetDeclaration, _ *ObservationNormalization) {
			(*after).ID = rsID(8)
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, after **RulesetDeclaration, _ *ObservationNormalization) {
			(*after).ID = nil
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, after **RulesetDeclaration, _ *ObservationNormalization) {
			(*after).Enforcement = rsText("FUTURE_ENFORCEMENT")
		},
		func(_ *string, _ *int64, before **RulesetDeclaration, _ **RulesetDeclaration, _ *ObservationNormalization) {
			(*before).Enforcement = rsText("active")
		},
		func(_ *string, _ *int64, before **RulesetDeclaration, _ **RulesetDeclaration, _ *ObservationNormalization) {
			(*before).Rules = nil
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *ObservationNormalization) {
			*n = ObservationNormalization{}
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *ObservationNormalization) {
			n.Path = "other"
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *ObservationNormalization) {
			n.RawValue = "disabled"
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *ObservationNormalization) {
			n.RawValue = "active"
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *ObservationNormalization) {
			n.Fallback = "active"
		},
	} {
		state := normalizedEnforcementState(7, "policy", "FUTURE_ENFORCEMENT")
		before := state.Policy
		before.ID = rsID(7)
		after := cloneRuleset(before)
		b, a := &before, &after
		op, id := "update", int64(7)
		record := state.EnforcementNormalization
		mutate(&op, &id, &b, &a, &record)
		change, err := NewRulesetChangeWithNormalizations(op, id, b, a, RulesetNormalizations{Enforcement: record})
		if err == nil || !reflect.DeepEqual(change, RulesetChange{}) {
			t.Fatal("invalid normalized operation accepted", change, err)
		}
		// Display has no separate ID argument; all other malformed evidence is identical.
		if id != 0 {
			display, err := NewRulesetDisplayChangeWithNormalizations(op, b, a, RulesetNormalizations{Enforcement: record})
			if err == nil || !reflect.DeepEqual(display, RulesetDisplayChange{}) {
				t.Fatal("invalid normalized display accepted", display, err)
			}
		}
	}
}
