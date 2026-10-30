package settings

import (
	"reflect"
	"testing"
)

func mergeMethodRecords(state RulesetState) RulesetNormalizations {
	return RulesetNormalizations{Enforcement: state.EnforcementNormalization, QualitySeverity: state.QualitySeverityNormalization, RequiredApprovals: state.RequiredApprovalsNormalization, BypassModes: state.BypassModeNormalizations, MergeMethods: state.MergeMethodsNormalization}
}

func TestRulesetMergeMethodsChangesOwnEvidenceForEqualityChoiceAndRemoval(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, desired := range []string{"equal", "squash", "removed"} {
			state := normalizedMergeMethodsState([]string{"squash", "future"}, companions)
			before, after := state.Policy, cloneRuleset(state.Policy)
			if desired == "squash" {
				ruleOf(t, after, "pull_request").AllowedMergeMethods = rsList("squash")
			} else if desired == "removed" {
				rules := []RulesetRule{}
				for _, rule := range *after.Rules {
					if rule.Type != "pull_request" {
						rules = append(rules, rule)
					}
				}
				after.Rules = &rules
			}
			records := mergeMethodRecords(state)
			want := mergeMethodRecords(normalizedMergeMethodsState([]string{"squash", "future"}, companions))
			change, err := NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
			if err != nil || change.ID() != 7 || !reflect.DeepEqual(change.Normalizations(), want) {
				t.Fatal("selected methods evidence lost", desired, change, err)
			}
			display, err := NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
			if err != nil || !reflect.DeepEqual(display.Normalizations(), want) || !reflect.DeepEqual(change.Display().Normalizations(), want) {
				t.Fatal("display methods evidence lost", display, err)
			}
			if desired == "equal" {
				if _, err := NewRulesetChange("update", 7, &before, &after); err == nil {
					t.Fatal("ordinary canonical equality became actionable")
				}
			}
			records.MergeMethods.RawValue[0] = "mutated input"
			records.MergeMethods.Fallback[0] = "mutated fallback"
			for _, returned := range []RulesetNormalizations{change.Normalizations(), change.Display().Normalizations(), display.Normalizations()} {
				returned.MergeMethods.RawValue[0] = "mutated output"
				returned.MergeMethods.Fallback[0] = "mutated fallback output"
			}
			if !reflect.DeepEqual(change.Normalizations(), want) || !reflect.DeepEqual(change.Display().Normalizations(), want) || !reflect.DeepEqual(display.Normalizations(), want) {
				t.Fatal("record storage escaped", change, display)
			}
			if desired == "removed" && (rulesetMergeMethods(*change.After()) != nil || rulesetMergeMethods(*display.After()) != nil) {
				t.Fatal("removal invented a desired methods list")
			}
		}
	}
}

func TestRulesetMergeMethodsChangesRejectInvalidEvidence(t *testing.T) {
	for name, mutate := range map[string]func(*int64, *RulesetDeclaration, *RulesetDeclaration, *RulesetNormalizations){
		"transport ID":   func(id *int64, _, _ *RulesetDeclaration, _ *RulesetNormalizations) { *id = 8 },
		"missing ID":     func(_ *int64, b, _ *RulesetDeclaration, _ *RulesetNormalizations) { b.ID = nil },
		"other after ID": func(_ *int64, _, a *RulesetDeclaration, _ *RulesetNormalizations) { a.ID = rsID(8) },
		"bad desired": func(_ *int64, _, a *RulesetDeclaration, _ *RulesetNormalizations) {
			ruleOf(t, *a, "pull_request").AllowedMergeMethods = rsList("future")
		},
		"forged raw": func(_ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.MergeMethods.RawValue = []string{"merge"}
		},
		"wrong fallback": func(_ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.MergeMethods.Fallback = []string{"squash"}
		},
		"present empty": func(_ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.MergeMethods = RulesetMergeMethodsNormalization{RawValue: []string{}}
		},
		"no evidence": func(_ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) { *n = RulesetNormalizations{} },
		"bad with others": func(_ *int64, _, _ *RulesetDeclaration, n *RulesetNormalizations) {
			n.MergeMethods = RulesetMergeMethodsNormalization{RawValue: []string{}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			state := normalizedMergeMethodsState([]string{"future"}, name == "bad with others")
			before, after := state.Policy, cloneRuleset(state.Policy)
			records, id := mergeMethodRecords(state), int64(7)
			mutate(&id, &before, &after, &records)
			if c, err := NewRulesetChangeWithNormalizations("update", id, &before, &after, records); err == nil || !reflect.DeepEqual(c, RulesetChange{}) {
				t.Fatal("invalid methods change accepted", c, err)
			}
			if id == 7 {
				if c, err := NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records); err == nil || !reflect.DeepEqual(c, RulesetDisplayChange{}) {
					t.Fatal("invalid methods display accepted", c, err)
				}
			}
		})
	}
	state := normalizedMergeMethodsState([]string{"future"}, false)
	after, records := state.Policy, mergeMethodRecords(state)
	after.ID = nil
	if _, err := NewRulesetChangeWithNormalizations("create", 0, nil, &after, records); err == nil {
		t.Fatal("create accepted observation evidence")
	}
	if _, err := NewRulesetDisplayChangeWithNormalizations("create", nil, &after, records); err == nil {
		t.Fatal("create display accepted observation evidence")
	}
}
