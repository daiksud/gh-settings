package settings

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func TestRulesetMergeMethodsReconciliationRetainsSelectedEvidence(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, desired := range []string{"equal", "subset", "methods omitted", "rule removed", "rules omitted"} {
			t.Run(fmt.Sprintf("companions=%t/%s", companions, desired), func(t *testing.T) {
				state := normalizedMergeMethodsState([]string{"future", "squash"}, companions)
				unlisted := normalizedMergeMethodsState([]string{"PRIVATE_METHOD"}, false)
				unlisted.ID, unlisted.Policy.ID, unlisted.Policy.Name = 9, rsID(9), "unlisted"
				observed := rsObserved(t, unlisted, state)
				after := cloneRuleset(state.Policy)
				switch desired {
				case "subset":
					ruleOf(t, after, "pull_request").AllowedMergeMethods = rsList("squash")
				case "methods omitted":
					ruleOf(t, after, "pull_request").AllowedMergeMethods = nil
				case "rule removed":
					rules := []RulesetRule{}
					for _, rule := range *after.Rules {
						if rule.Type != "pull_request" {
							rules = append(rules, rule)
						}
					}
					after.Rules = &rules
				case "rules omitted":
					after.Rules = nil
				}
				changes, err := ReconcileRulesets(observed, rsDesired(t, after))
				if err != nil || len(changes) != 1 || changes[0].ID() != 7 || changes[0].Operation() != "update" {
					t.Fatal("selected normalized update lost", changes, err)
				}
				want := mergeMethodRecords(normalizedMergeMethodsState([]string{"future", "squash"}, companions))
				if !reflect.DeepEqual(changes[0].Normalizations(), want) || !reflect.DeepEqual(changes[0].Display().Normalizations(), want) {
					t.Fatal("selected record or companions dropped", changes[0].Normalizations())
				}
				methods := rulesetMergeMethods(*changes[0].After())
				if desired == "rule removed" || desired == "rules omitted" {
					if methods != nil {
						t.Fatal("removal invented after methods", *methods)
					}
				} else {
					wantMethods := []string{"merge", "rebase", "squash"}
					if desired == "subset" {
						wantMethods = []string{"squash"}
					}
					if methods == nil || !slices.Equal(*methods, wantMethods) {
						t.Fatal("desired/defaulted methods changed", methods)
					}
				}
				for _, declarations := range []*RulesetDeclarations{nil, rsDesired(t)} {
					if got, err := ReconcileRulesets(observed, declarations); err != nil || len(got) != 0 {
						t.Fatal("unmanaged scope became actionable", got, err)
					}
				}
			})
		}
	}
}

func TestRulesetMergeMethodsReconciliationKeepsCanonicalNoopAndNameOrder(t *testing.T) {
	z := normalizedMergeMethodsState([]string{"Z_METHOD"}, false)
	z.Policy.Name = "z"
	a := normalizedMergeMethodsState([]string{"A_METHOD"}, false)
	a.ID, a.Policy.ID, a.Policy.Name = 5, rsID(5), "a"
	changes, err := ReconcileRulesets(rsObserved(t, z, a), rsDesired(t, z.Policy, a.Policy))
	if err != nil || len(changes) != 2 || changes[0].ID() != 5 || changes[1].ID() != 7 {
		t.Fatal("selected updates lost stable ID/name order", changes, err)
	}
	if !slices.Equal(changes[0].Normalizations().MergeMethods.RawValue, []string{"A_METHOD"}) || !slices.Equal(changes[1].Normalizations().MergeMethods.RawValue, []string{"Z_METHOD"}) {
		t.Fatal("raw methods crossed IDs")
	}
	canonical := normalizedMergeMethodsState([]string{"unused"}, false)
	canonical.ID, canonical.Policy.ID, canonical.Policy.Name = 8, rsID(8), "canonical"
	canonical.MergeMethodsNormalization = RulesetMergeMethodsNormalization{}
	if got, err := ReconcileRulesets(rsObserved(t, canonical, z), rsDesired(t, canonical.Policy)); err != nil || len(got) != 0 {
		t.Fatal("canonical or unlisted object became actionable", got, err)
	}
}
