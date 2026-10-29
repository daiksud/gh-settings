package settings

import (
	"reflect"
	"testing"
)

func TestNormalizedRulesetRequiredApprovalsChangesOwnTypedDecisionAndRemoval(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, count := range []int{0, 1, 10, -1} {
			state := normalizedRequiredApprovalsState(7, "policy", 11, both)
			before, after := state.Policy, cloneRuleset(state.Policy)
			if count == -1 {
				rules := []RulesetRule{}
				for _, r := range *after.Rules {
					if r.Type != "pull_request" {
						rules = append(rules, r)
					}
				}
				after.Rules = &rules
			} else {
				ruleOf(t, after, "pull_request").RequiredApprovingReviewCount = paramInt(count)
			}
			records := RulesetNormalizations{Enforcement: state.EnforcementNormalization, QualitySeverity: state.QualitySeverityNormalization, RequiredApprovals: state.RequiredApprovalsNormalization}
			change, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
			if err != nil || change.Normalizations() != records || change.Display().Normalizations() != records {
				t.Fatal(change, err)
			}
			if count == 0 {
				ordinaryBefore, ordinaryAfter := cloneRuleset(before), cloneRuleset(after)
				omitDismissalRestriction(&ordinaryBefore)
				omitDismissalRestriction(&ordinaryAfter)
				if _, err := NewRulesetChange("update", 7, &ordinaryBefore, &ordinaryAfter); err == nil {
					t.Fatal("ordinary equality accepted")
				}
			}
			wantBefore, wantAfter := cloneRuleset(before), cloneRuleset(after)
			before.Name = "mutated input"
			after.Name = "mutated after"
			records.RequiredApprovals.RawValue = 99
			returned := change.Normalizations()
			returned.RequiredApprovals.RawValue = 99
			if change.Normalizations().RequiredApprovals.RawValue != 11 || !reflect.DeepEqual(change.Before(), &wantBefore) || !reflect.DeepEqual(change.After(), &wantAfter) {
				t.Fatal("typed decision alias", change)
			}
			if !change.After().HasWriteOnlyDismissal() && count != -1 {
				t.Fatal("write-only intent dropped")
			}
			if change.Display().Before().HasWriteOnlyDismissal() || change.Display().After().HasWriteOnlyDismissal() {
				t.Fatal("display leaked write-only values")
			}
			if count == -1 {
				got, err := (RulesetState{ID: 7, Policy: *change.After()}).RawRequiredApprovals()
				if err != nil || got != nil {
					t.Fatal("removal invented after count", got, err)
				}
			}
		}
	}
}

func TestNormalizedRulesetRequiredApprovalsChangesRejectForgedAndUnsupportedDesired(t *testing.T) {
	for _, mutate := range []func(*RulesetDeclaration, *RulesetDeclaration, *RulesetNormalizations){
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) {
			ruleOf(t, *a, "pull_request").RequiredApprovingReviewCount = paramInt(11)
		},
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) {
			ruleOf(t, *a, "pull_request").RequiredApprovingReviewCount = nil
		},
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) {
			ruleOf(t, *b, "pull_request").RequiredApprovingReviewCount = paramInt(1)
		},
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) { n.RequiredApprovals.RawValue = 0 },
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) { n.RequiredApprovals.Path = "other" },
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) { n.RequiredApprovals.Fallback = 1 },
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) { b.ID = rsID(9) },
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) { a.ID = rsID(9) },
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) { n.Enforcement.RawValue = "active" },
		func(b, a *RulesetDeclaration, n *RulesetNormalizations) { n.QualitySeverity.Path = "other" },
	} {
		state := normalizedRequiredApprovalsState(7, "policy", 11, true)
		before, after := state.Policy, cloneRuleset(state.Policy)
		records := RulesetNormalizations{Enforcement: state.EnforcementNormalization, QualitySeverity: state.QualitySeverityNormalization, RequiredApprovals: state.RequiredApprovalsNormalization}
		mutate(&before, &after, &records)
		change, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
		if err == nil || !reflect.DeepEqual(change, RulesetChange{}) {
			t.Fatal("invalid approval decision accepted", change, err)
		}
		display, err := NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
		if err == nil || !reflect.DeepEqual(display, RulesetDisplayChange{}) {
			t.Fatal("invalid approval display accepted", display, err)
		}
	}
	state := normalizedRequiredApprovalsState(7, "policy", 11, false)
	after := cloneRuleset(state.Policy)
	records := RulesetNormalizations{RequiredApprovals: state.RequiredApprovalsNormalization}
	if change, err := NewRulesetChangeWithNormalizations("create", 7, &state.Policy, &after, records); err == nil || !reflect.DeepEqual(change, RulesetChange{}) {
		t.Fatal("create evidence accepted", change, err)
	}
}
