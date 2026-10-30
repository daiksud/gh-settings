package settings

import (
	"reflect"
	"slices"
	"testing"
)

func normalizedMergeMethodsState(raw []string, companions bool) RulesetState {
	state := RulesetState{ID: 7, Policy: reviewPolicy()}
	if companions {
		state = normalizedBypassState("BYPASS_FUTURE", true)
	}
	state.Policy.ID, state.Policy.Name = rsID(7), "methods"
	pr := ruleOfForApprovals(state.Policy)
	pr.DismissalRestriction = nil
	pr.AllowedMergeMethods = rsList("merge", "rebase", "squash")
	state.MergeMethodsNormalization = RulesetMergeMethodsNormalization{
		Path: RulesetMergeMethodsPath, RawValue: slices.Clone(raw), Fallback: []string{"merge", "squash", "rebase"},
	}
	return state
}

func TestRulesetMergeMethodsObservationOwnsRawOrderAndCompanions(t *testing.T) {
	for _, raw := range [][]string{{"squash", "FUTURE_METHOD", "merge"}, {""}, {" \t"}} {
		for _, companions := range []bool{false, true} {
			state := normalizedMergeMethodsState(raw, companions)
			observed, err := NewRulesetObservation([]RulesetState{state}, 0)
			if err != nil {
				t.Fatal(err)
			}
			state.MergeMethodsNormalization.RawValue[0] = "mutated input"
			state.MergeMethodsNormalization.Fallback[0] = "mutated fallback"
			(*ruleOf(t, state.Policy, "pull_request").AllowedMergeMethods)[0] = "mutated policy"
			values, err := observed.Values()
			if err != nil {
				t.Fatal(err)
			}
			got, err := values[0].RawMergeMethods()
			if err != nil || !slices.Equal(got, raw) {
				t.Fatal("raw order or values lost", got, raw, err)
			}
			got[0] = "mutated returned raw"
			if again, err := values[0].RawMergeMethods(); err != nil || !slices.Equal(again, raw) {
				t.Fatal("raw list alias", again, err)
			}
			if companions {
				if v, err := values[0].RawEnforcement(); err != nil || v != "evaluate" {
					t.Fatal("enforcement companion lost", v, err)
				}
				if v, err := values[0].RawRequiredApprovals(); err != nil || v == nil || *v != 11 {
					t.Fatal("approvals companion lost", v, err)
				}
				if v, err := values[0].RawQualitySeverity(); err != nil || v == nil || *v != "FUTURE_SEVERITY" {
					t.Fatal("quality companion lost", v, err)
				}
				actors, err := values[0].RawBypassActors()
				if err != nil || len(actors) != 4 || *actors[2].BypassMode != "BYPASS_FUTURE" {
					t.Fatal("bypass companion lost", actors, err)
				}
			}
			values[0].MergeMethodsNormalization.RawValue[0] = "mutated output"
			values[0].MergeMethodsNormalization.Fallback[0] = "mutated fallback output"
			(*ruleOf(t, values[0].Policy, "pull_request").AllowedMergeMethods)[0] = "mutated policy output"
			fresh, err := observed.Values()
			if err != nil {
				t.Fatal(err)
			}
			if got, err := fresh[0].RawMergeMethods(); err != nil || !slices.Equal(got, raw) {
				t.Fatal("observation list alias", got, err)
			}
			exported, err := observed.ExportDeclarations()
			if err != nil || exported.Len() != 1 || !slices.Equal(*ruleOf(t, exported.Values()[0], "pull_request").AllowedMergeMethods, []string{"merge", "rebase", "squash"}) {
				t.Fatal("effective export changed", exported, err)
			}
		}
	}
}

func TestRulesetMergeMethodsObservationKeepsAbsenceAndCanonicalLists(t *testing.T) {
	normalized := normalizedMergeMethodsState([]string{"future", "squash"}, false)
	ruleOf(t, normalized.Policy, "pull_request").AllowedMergeMethods = rsList("rebase", "squash", "merge")
	if got, err := normalized.RawMergeMethods(); err != nil || !slices.Equal(got, []string{"future", "squash"}) {
		t.Fatal("effective fallback order changed evidence validity", got, err)
	}
	state := RulesetState{ID: 7, Policy: reviewPolicy()}
	ruleOf(t, state.Policy, "pull_request").AllowedMergeMethods = rsList("squash", "merge")
	got, err := state.RawMergeMethods()
	if err != nil || !slices.Equal(got, []string{"squash", "merge"}) {
		t.Fatal("canonical values changed", got, err)
	}
	got[0] = "mutated"
	if !slices.Equal(*ruleOf(t, state.Policy, "pull_request").AllowedMergeMethods, []string{"squash", "merge"}) {
		t.Fatal("canonical raw aliases policy")
	}
	if got, err := (RulesetState{ID: 7, Policy: basicPolicy("absent")}).RawMergeMethods(); err != nil || got != nil {
		t.Fatal("absent rule invented methods", got, err)
	}
	if got, err := (RulesetState{}).RawMergeMethods(); err == nil || got != nil {
		t.Fatal("unacquired became absence", got, err)
	}
}

func TestRulesetMergeMethodsObservationRejectsForgedOrIncompleteEvidence(t *testing.T) {
	isolated := normalizedMergeMethodsState([]string{"future"}, false)
	isolated.Policy.ID = rsID(8)
	if got, err := isolated.RawMergeMethods(); err == nil || got != nil {
		t.Fatal("merge-only evidence ignored policy ID", got, err)
	}
	for name, mutate := range map[string]func(*RulesetState){
		"missing ID":          func(s *RulesetState) { s.ID = 0 },
		"missing rules":       func(s *RulesetState) { s.Policy.Rules = nil },
		"absent rule":         func(s *RulesetState) { s.Policy.Rules = &[]RulesetRule{} },
		"missing methods":     func(s *RulesetState) { ruleOf(t, s.Policy, "pull_request").AllowedMergeMethods = nil },
		"not default":         func(s *RulesetState) { ruleOf(t, s.Policy, "pull_request").AllowedMergeMethods = rsList("merge") },
		"wrong path":          func(s *RulesetState) { s.MergeMethodsNormalization.Path = "other" },
		"missing raw":         func(s *RulesetState) { s.MergeMethodsNormalization.RawValue = nil },
		"empty raw":           func(s *RulesetState) { s.MergeMethodsNormalization.RawValue = []string{} },
		"duplicate raw":       func(s *RulesetState) { s.MergeMethodsNormalization.RawValue = []string{"future", "future"} },
		"duplicate canonical": func(s *RulesetState) { s.MergeMethodsNormalization.RawValue = []string{"merge", "merge", "future"} },
		"canonical raw":       func(s *RulesetState) { s.MergeMethodsNormalization.RawValue = []string{"merge", "squash"} },
		"missing fallback":    func(s *RulesetState) { s.MergeMethodsNormalization.Fallback = nil },
		"wrong fallback":      func(s *RulesetState) { s.MergeMethodsNormalization.Fallback = []string{"merge"} },
		"present empty": func(s *RulesetState) {
			s.MergeMethodsNormalization = RulesetMergeMethodsNormalization{RawValue: []string{}}
		},
		"invalid companion": func(s *RulesetState) { s.RequiredApprovalsNormalization.RawValue = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			state := normalizedMergeMethodsState([]string{"future"}, name == "invalid companion")
			mutate(&state)
			if got, err := state.RawMergeMethods(); err == nil || got != nil {
				t.Fatal("invalid evidence escaped", got, err)
			}
			if got, err := state.RawBypassActors(); err == nil || got != nil {
				t.Fatal("cross-record evidence escaped", got, err)
			}
			observed, err := NewRulesetObservation([]RulesetState{{ID: 9, Policy: basicPolicy("valid")}, state}, 0)
			if err == nil || !reflect.DeepEqual(observed, RulesetObservation{}) {
				t.Fatal("partial collection escaped", observed, err)
			}
		})
	}
}
