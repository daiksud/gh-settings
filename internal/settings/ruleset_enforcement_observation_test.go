package settings

import (
	"reflect"
	"testing"
)

func normalizedEnforcementState(id int64, name, raw string) RulesetState {
	policy := basicPolicy(name)
	policy.Enforcement = rsText("disabled")
	return RulesetState{ID: id, Policy: policy, EnforcementNormalization: ObservationNormalization{Path: RulesetEnforcementPath, RawValue: raw, Fallback: "disabled"}}
}

func TestRulesetEnforcementObservationOwnsProvenancePerID(t *testing.T) {
	for _, raw := range []string{"evaluate", "FUTURE_MODE", "", " \t"} {
		t.Run(raw, func(t *testing.T) {
			input := []RulesetState{normalizedEnforcementState(7, "first", raw), normalizedEnforcementState(9, "second", "OTHER_MODE")}
			observed, err := NewRulesetObservation(input, 2)
			if err != nil {
				t.Fatal(err)
			}
			*input[0].Policy.Enforcement = "active"
			input[0].EnforcementNormalization.RawValue = "mutated"
			values, err := observed.Values()
			if err != nil || len(values) != 2 || values[0].ID != 7 || values[1].ID != 9 || observed.ExcludedPush() != 2 {
				t.Fatal(values, err)
			}
			got, err := values[0].RawEnforcement()
			if err != nil || got != raw || *values[0].Policy.Enforcement != "disabled" {
				t.Fatal(values, got, err)
			}
			if second, err := values[1].RawEnforcement(); err != nil || second != "OTHER_MODE" {
				t.Fatal("second ID provenance lost", second, err)
			}
			*values[0].Policy.Enforcement = "active"
			values[0].EnforcementNormalization.RawValue = "mutated again"
			fresh, err := observed.Values()
			got, rawErr := fresh[0].RawEnforcement()
			if err != nil || rawErr != nil || got != raw || *fresh[0].Policy.Enforcement != "disabled" {
				t.Fatal(fresh, got, err, rawErr)
			}
			if second, err := fresh[1].RawEnforcement(); err != nil || second != "OTHER_MODE" {
				t.Fatal("second ID provenance changed", second, err)
			}
			exported, err := observed.ExportDeclarations()
			if err != nil || exported.Len() != 2 || *exported.Values()[0].Enforcement != "disabled" || *exported.Values()[0].ID != 7 {
				t.Fatal(exported, err)
			}
		})
	}
	canonical := RulesetState{ID: 7, Policy: basicPolicy("canonical")}
	canonical.Policy.Enforcement = rsText("evaluate")
	if got, err := canonical.RawEnforcement(); err != nil || got != "evaluate" {
		t.Fatal("canonical native contract changed", got, err)
	}
}

func TestRulesetEnforcementObservationRejectsForgedOrIncompleteEvidence(t *testing.T) {
	for _, mutate := range []func(*RulesetState){
		func(s *RulesetState) { s.ID = 0 },
		func(s *RulesetState) { s.Policy.Enforcement = nil },
		func(s *RulesetState) { s.Policy.Enforcement = rsText("active") },
		func(s *RulesetState) { s.Policy.Enforcement = rsText("evaluate") },
		func(s *RulesetState) { s.Policy.ID = rsID(8) },
		func(s *RulesetState) { s.Policy.Rules = nil },
		func(s *RulesetState) { s.EnforcementNormalization.Path = "repository.settings.rulesets.other" },
		func(s *RulesetState) { s.EnforcementNormalization.Fallback = "active" },
		func(s *RulesetState) { s.EnforcementNormalization.RawValue = "disabled" },
		func(s *RulesetState) { s.EnforcementNormalization.RawValue = "active" },
	} {
		state := normalizedEnforcementState(7, "invalid", "evaluate")
		mutate(&state)
		if raw, err := state.RawEnforcement(); err == nil || raw != "" {
			t.Fatal("invalid raw accessor", state, raw, err)
		}
		observed, err := NewRulesetObservation([]RulesetState{{ID: 9, Policy: basicPolicy("valid")}, state}, 0)
		if err == nil || !reflect.DeepEqual(observed, RulesetObservation{}) {
			t.Fatal("partial invalid observation escaped", observed, err)
		}
	}
}
