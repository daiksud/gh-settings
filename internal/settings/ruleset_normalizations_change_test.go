package settings

import (
	"reflect"
	"testing"
)

func TestRulesetNormalizationAggregateOwnsIndependentAndCombinedDecisions(t *testing.T) {
	for _, evidence := range []string{"enforcement", "quality", "both"} {
		for _, desired := range []string{"errors", "warnings", "notes", "all", "removed"} {
			state := normalizedQualityState(7, "policy", "FUTURE_SEVERITY")
			if evidence == "enforcement" {
				state.QualitySeverityNormalization = ObservationNormalization{}
			}
			if evidence == "quality" {
				state.EnforcementNormalization = ObservationNormalization{}
				state.Policy.Enforcement = rsText("active")
			}
			before := state.Policy
			before.ID = rsID(7)
			after := cloneRuleset(before)
			if desired == "removed" {
				after.Rules = &[]RulesetRule{}
			} else {
				ruleOf(t, after, "code_quality").Severity = rsText(desired)
			}
			records := RulesetNormalizations{Enforcement: state.EnforcementNormalization, QualitySeverity: state.QualitySeverityNormalization}
			change, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
			if err != nil || change.Operation() != "update" || change.ID() != 7 || !reflect.DeepEqual(change.Normalizations(), records) || !reflect.DeepEqual(change.Display().Normalizations(), records) {
				t.Fatal(change, err)
			}
			if desired == "errors" {
				if _, err := NewRulesetChange("update", 7, &before, &after); err == nil {
					t.Fatal("ordinary equality accepted")
				}
			}
			wantBefore, wantAfter := cloneRuleset(before), cloneRuleset(after)
			before.Name = "mutated input"
			*after.Enforcement = "evaluate"
			copy := change.Normalizations()
			copy.Enforcement.RawValue = "mutated record"
			copy.QualitySeverity.RawValue = "mutated record"
			returned := change.After()
			returned.Name = "mutated returned policy"
			if !reflect.DeepEqual(change.Normalizations(), records) || !reflect.DeepEqual(change.Display().Normalizations(), records) || !reflect.DeepEqual(change.Before(), &wantBefore) || !reflect.DeepEqual(change.After(), &wantAfter) || !reflect.DeepEqual(change.Display().After(), &wantAfter) {
				t.Fatal("aggregate/policy alias", change)
			}
			if desired == "removed" {
				value, err := (RulesetState{ID: 7, Policy: *change.After()}).RawQualitySeverity()
				if err != nil || value != nil {
					t.Fatal("rule removal invented after severity", value, err)
				}
			}
		}
	}
}

func TestRulesetNormalizationAggregateRejectsUnboundAndForgedEvidence(t *testing.T) {
	for _, mutate := range []func(*string, *int64, **RulesetDeclaration, **RulesetDeclaration, *RulesetNormalizations){
		func(op *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, _ *RulesetNormalizations) {
			*op = "create"
		},
		func(_ *string, id *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, _ *RulesetNormalizations) {
			*id = 0
		},
		func(_ *string, _ *int64, b **RulesetDeclaration, _ **RulesetDeclaration, _ *RulesetNormalizations) {
			(*b).ID = nil
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, a **RulesetDeclaration, _ *RulesetNormalizations) {
			(*a).ID = rsID(8)
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, a **RulesetDeclaration, _ *RulesetNormalizations) {
			ruleOf(t, **a, "code_quality").Severity = nil
		},
		func(_ *string, _ *int64, b **RulesetDeclaration, _ **RulesetDeclaration, _ *RulesetNormalizations) {
			(*b).Rules = &[]RulesetRule{}
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *RulesetNormalizations) {
			*n = RulesetNormalizations{}
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *RulesetNormalizations) {
			n.QualitySeverity.RawValue = "warnings"
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *RulesetNormalizations) {
			n.QualitySeverity.Path = RulesetEnforcementPath
		},
		func(_ *string, _ *int64, _ **RulesetDeclaration, _ **RulesetDeclaration, n *RulesetNormalizations) {
			n.Enforcement.RawValue = "active"
		},
	} {
		state := normalizedQualityState(7, "policy", "FUTURE_SEVERITY")
		before := state.Policy
		before.ID = rsID(7)
		after := cloneRuleset(before)
		b, a := &before, &after
		op, id := "update", int64(7)
		records := RulesetNormalizations{Enforcement: state.EnforcementNormalization, QualitySeverity: state.QualitySeverityNormalization}
		mutate(&op, &id, &b, &a, &records)
		change, err := NewRulesetChangeWithNormalizations(op, id, b, a, records)
		if err == nil || !reflect.DeepEqual(change, RulesetChange{}) {
			t.Fatal("invalid aggregate accepted", change, err)
		}
		if id != 0 {
			display, err := NewRulesetDisplayChangeWithNormalizations(op, b, a, records)
			if err == nil || !reflect.DeepEqual(display, RulesetDisplayChange{}) {
				t.Fatal("invalid aggregate display accepted", display, err)
			}
		}
	}
}

func TestRulesetNormalizationAggregatePreservesDismissalOnQualityRemoval(t *testing.T) {
	before := reviewPolicy()
	before.ID, before.Enforcement = rsID(7), rsText("disabled")
	after := cloneRuleset(before)
	*before.Rules = append(*before.Rules, RulesetRule{Type: "code_quality", Parameters: &RuleParameters{Severity: rsText("errors")}})
	records := RulesetNormalizations{
		Enforcement:     ObservationNormalization{Path: RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"},
		QualitySeverity: ObservationNormalization{Path: RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"},
	}
	change, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	if !change.Before().HasWriteOnlyDismissal() || !change.After().HasWriteOnlyDismissal() || change.Display().Before().HasWriteOnlyDismissal() || change.Display().After().HasWriteOnlyDismissal() || !reflect.DeepEqual(change.Display().Normalizations(), records) {
		t.Fatal("dismissal or simultaneous evidence lost", change)
	}
}
