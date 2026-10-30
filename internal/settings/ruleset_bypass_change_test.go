package settings

import (
	"reflect"
	"testing"
)

func TestRulesetBypassChangesOwnActorBoundEvidence(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, remove := range []bool{false, true} {
			state := normalizedBypassState("FUTURE_MODE", companions)
			state.Policy.ID = rsID(7)
			before, after := state.Policy, cloneRuleset(state.Policy)
			if remove {
				after.BypassActors = &[]BypassActor{}
			}
			records := RulesetNormalizations{Enforcement: state.EnforcementNormalization, QualitySeverity: state.QualitySeverityNormalization, RequiredApprovals: state.RequiredApprovalsNormalization, BypassModes: state.BypassModeNormalizations}
			independent := normalizedBypassState("FUTURE_MODE", companions)
			want := RulesetNormalizations{Enforcement: independent.EnforcementNormalization, QualitySeverity: independent.QualitySeverityNormalization, RequiredApprovals: independent.RequiredApprovalsNormalization, BypassModes: independent.BypassModeNormalizations}
			change, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
			if err != nil || change.ID() != 7 || change.Operation() != "update" || !reflect.DeepEqual(change.Normalizations(), want) || !reflect.DeepEqual(change.Display().Normalizations(), want) {
				t.Fatal("selected bypass evidence lost", change, err)
			}
			display, err := NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
			if err != nil || !reflect.DeepEqual(display.Normalizations(), want) {
				t.Fatal("display evidence lost", display, err)
			}
			if !remove && !companions {
				if _, err := NewRulesetChange("update", 7, &before, &after); err == nil {
					t.Fatal("ordinary equality became actionable")
				}
			}
			wantBefore, wantAfter := cloneRuleset(before), cloneRuleset(after)
			*records.BypassModes[1].ActorID = 90
			records.BypassModes[0].Normalization.RawValue = "mutated input"
			*(*before.BypassActors)[0].ActorID = 99
			before.Name, after.Name = "mutated before", "mutated after"
			for _, returned := range []RulesetNormalizations{change.Normalizations(), change.Display().Normalizations(), display.Normalizations()} {
				*returned.BypassModes[1].ActorID = 91
				returned.BypassModes[2].Normalization.RawValue = "mutated accessor"
				returned.Enforcement.RawValue = "mutated scalar"
			}
			if !reflect.DeepEqual(change.Normalizations(), want) || !reflect.DeepEqual(change.Display().Normalizations(), want) || !reflect.DeepEqual(display.Normalizations(), want) || !reflect.DeepEqual(change.Before(), &wantBefore) || !reflect.DeepEqual(change.After(), &wantAfter) {
				t.Fatal("bypass evidence or policy alias", change, display)
			}
			if remove && (change.After().BypassActors == nil || len(*change.After().BypassActors) != 0 || len(*display.After().BypassActors) != 0) {
				t.Fatal("actor removal invented a desired bypass mode", change)
			}
			if display.Before().HasWriteOnlyDismissal() || display.After().HasWriteOnlyDismissal() || change.Display().Before().HasWriteOnlyDismissal() {
				t.Fatal("display leaked write-only intent")
			}
		}
	}
}

func TestRulesetBypassChangesRejectUnboundOrEmptyEvidence(t *testing.T) {
	for _, mutate := range []func(*string, *int64, *RulesetDeclaration, *RulesetDeclaration, *RulesetNormalizations){
		func(op *string, _ *int64, _, _ *RulesetDeclaration, _ *RulesetNormalizations) { *op = "create" },
		func(_ *string, id *int64, _, _ *RulesetDeclaration, _ *RulesetNormalizations) { *id = 8 },
		func(_ *string, _ *int64, b, _ *RulesetDeclaration, _ *RulesetNormalizations) { b.ID = nil },
		func(_ *string, _ *int64, _, a *RulesetDeclaration, _ *RulesetNormalizations) { a.ID = rsID(8) },
		func(_ *string, _ *int64, _, a *RulesetDeclaration, _ *RulesetNormalizations) {
			(*a.BypassActors)[0].BypassMode = rsText("FUTURE_MODE")
		},
		func(_ *string, _ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.BypassModes[1].ActorID = rsID(99)
		},
		func(_ *string, _ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.BypassModes[1].ActorType, n.BypassModes[1].ActorID = "Team:9", nil
		},
		func(_ *string, _ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.BypassModes[0].ActorID = rsID(0)
		},
		func(_ *string, _ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.BypassModes = append(n.BypassModes, n.BypassModes[1])
		},
		func(_ *string, _ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.BypassModes = []RulesetBypassModeNormalization{{}}
		},
		func(_ *string, _ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.BypassModes = []RulesetBypassModeNormalization{}
		},
		func(_ *string, _ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) { n.BypassModes = nil },
	} {
		state := normalizedBypassState("FUTURE_MODE", false)
		state.Policy.ID = rsID(7)
		before, after := state.Policy, cloneRuleset(state.Policy)
		records := RulesetNormalizations{BypassModes: state.BypassModeNormalizations}
		op, id := "update", int64(7)
		mutate(&op, &id, &before, &after, &records)
		change, err := NewRulesetChangeWithNormalizations(op, id, &before, &after, records)
		if err == nil || !reflect.DeepEqual(change, RulesetChange{}) {
			t.Fatal("invalid bypass change accepted", change, err)
		}
		// Display identity comes from before.ID; the transport-ID-only mutation
		// applies to the executable change boundary above.
		if id == 7 {
			display, err := NewRulesetDisplayChangeWithNormalizations(op, &before, &after, records)
			if err == nil || !reflect.DeepEqual(display, RulesetDisplayChange{}) {
				t.Fatal("invalid bypass display accepted", display, err)
			}
		}
	}
}
