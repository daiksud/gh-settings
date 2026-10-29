package settings

import (
	"reflect"
	"testing"
)

func normalizedQualityState(id int64, name, raw string) RulesetState {
	state := normalizedEnforcementState(id, name, "evaluate")
	state.Policy.Rules = &[]RulesetRule{{Type: "code_quality", Parameters: &RuleParameters{Severity: rsText("errors")}}}
	state.QualitySeverityNormalization = ObservationNormalization{Path: RulesetQualitySeverityPath, RawValue: raw, Fallback: "errors"}
	return state
}

func TestRulesetQualityObservationDistinguishesAbsenceAndOwnsBothRecords(t *testing.T) {
	for _, raw := range []string{"FUTURE_SEVERITY", "", " \t", "warnings_and_higher"} {
		state := normalizedQualityState(7, "first", raw)
		other := normalizedQualityState(9, "other", "OTHER_SEVERITY")
		observed, err := NewRulesetObservation([]RulesetState{state, other}, 0)
		if err != nil {
			t.Fatal(err)
		}
		*ruleOf(t, state.Policy, "code_quality").Severity = "all"
		state.QualitySeverityNormalization.RawValue = "mutated caller"
		values, err := observed.Values()
		if err != nil {
			t.Fatal(err)
		}
		for i, want := range []string{raw, "OTHER_SEVERITY"} {
			got, err := values[i].RawQualitySeverity()
			if err != nil || got == nil || *got != want {
				t.Fatal("ID-bound raw quality lost", values, got, err)
			}
			*got = "mutated result"
			again, err := values[i].RawQualitySeverity()
			if err != nil || again == nil || *again != want {
				t.Fatal("raw pointer alias", again, err)
			}
			if enf, err := values[i].RawEnforcement(); err != nil || enf != "evaluate" {
				t.Fatal("simultaneous enforcement lost", enf, err)
			}
		}
		*ruleOf(t, values[0].Policy, "code_quality").Severity = "all"
		values[0].QualitySeverityNormalization.RawValue = "mutated returned record"
		fresh, err := observed.Values()
		got, rawErr := fresh[0].RawQualitySeverity()
		if err != nil || rawErr != nil || got == nil || *got != raw || *ruleOf(t, fresh[0].Policy, "code_quality").Severity != "errors" {
			t.Fatal("observation alias", fresh, got, err, rawErr)
		}
		if second, err := fresh[1].RawQualitySeverity(); err != nil || second == nil || *second != "OTHER_SEVERITY" {
			t.Fatal("second ID provenance changed", second, err)
		}
		exported, err := observed.ExportDeclarations()
		if err != nil || exported.Len() != 2 || *ruleOf(t, exported.Values()[0], "code_quality").Severity != "errors" || *exported.Values()[0].Enforcement != "disabled" {
			t.Fatal("effective export changed", exported, err)
		}
		only := normalizedQualityState(11, "quality only", raw)
		only.Policy.Enforcement = rsText("active")
		only.EnforcementNormalization = ObservationNormalization{}
		independent, err := NewRulesetObservation([]RulesetState{only}, 0)
		if err != nil {
			t.Fatal(err)
		}
		owned, err := independent.Values()
		if err != nil || owned[0].EnforcementNormalization != (ObservationNormalization{}) {
			t.Fatal("enforcement provenance invented", owned, err)
		}
		if quality, err := owned[0].RawQualitySeverity(); err != nil || quality == nil || *quality != raw {
			t.Fatal("quality-only evidence lost", quality, err)
		}
		if enforcement, err := owned[0].RawEnforcement(); err != nil || enforcement != "active" {
			t.Fatal("quality changed enforcement", enforcement, err)
		}
		projected, err := independent.ExportDeclarations()
		if err != nil || *projected.Values()[0].Enforcement != "active" || *ruleOf(t, projected.Values()[0], "code_quality").Severity != "errors" {
			t.Fatal("quality-only export changed", projected, err)
		}
	}
	absent := RulesetState{ID: 7, Policy: basicPolicy("absent")}
	if value, err := absent.RawQualitySeverity(); err != nil || value != nil {
		t.Fatal("absent rule invented severity", value, err)
	}
	for _, native := range []string{"errors", "warnings", "notes", "all"} {
		state := absent
		state.Policy.Rules = &[]RulesetRule{{Type: "code_quality", Parameters: &RuleParameters{Severity: rsText(native)}}}
		if got, err := state.RawQualitySeverity(); err != nil || got == nil || *got != native {
			t.Fatal("canonical policy changed", got, err)
		}
	}
	if value, err := (RulesetState{}).RawQualitySeverity(); err == nil || value != nil {
		t.Fatal("unacquired became absence", value, err)
	}
}

func TestRulesetQualityObservationRejectsIncompleteAndCrossRecordForgery(t *testing.T) {
	for _, mutate := range []func(*RulesetState){
		func(s *RulesetState) { s.ID = 0 },
		func(s *RulesetState) { s.Policy.ID = rsID(8) },
		func(s *RulesetState) { s.Policy.Rules = nil },
		func(s *RulesetState) { s.Policy.Rules = &[]RulesetRule{} },
		func(s *RulesetState) { (*s.Policy.Rules)[0].Parameters = nil },
		func(s *RulesetState) { ruleOf(t, s.Policy, "code_quality").Severity = nil },
		func(s *RulesetState) { ruleOf(t, s.Policy, "code_quality").Severity = rsText("warnings") },
		func(s *RulesetState) { s.QualitySeverityNormalization.Path = "other" },
		func(s *RulesetState) { s.QualitySeverityNormalization.RawValue = "errors" },
		func(s *RulesetState) { s.QualitySeverityNormalization.RawValue = "warnings" },
		func(s *RulesetState) { s.QualitySeverityNormalization.Fallback = "all" },
		func(s *RulesetState) { s.EnforcementNormalization.Path = "other" },
		func(s *RulesetState) { s.EnforcementNormalization.RawValue = "disabled" },
	} {
		state := normalizedQualityState(7, "invalid", "FUTURE_SEVERITY")
		mutate(&state)
		if raw, err := state.RawQualitySeverity(); err == nil || raw != nil {
			t.Fatal("invalid quality evidence returned", raw, err)
		}
		if raw, err := state.RawEnforcement(); err == nil || raw != "" {
			t.Fatal("cross-record forgery returned enforcement", raw, err)
		}
		observed, err := NewRulesetObservation([]RulesetState{{ID: 9, Policy: basicPolicy("valid")}, state}, 0)
		if err == nil || !reflect.DeepEqual(observed, RulesetObservation{}) {
			t.Fatal("invalid partial collection escaped", observed, err)
		}
	}
}
