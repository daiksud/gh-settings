package settings

import (
	"reflect"
	"testing"
)

func normalizedRequiredApprovalsState(id int64, name string, raw int, both bool) RulesetState {
	state := RulesetState{ID: id, Policy: reviewPolicy(), RequiredApprovalsNormalization: RulesetRequiredApprovalsNormalization{Path: RulesetRequiredApprovalsPath, RawValue: raw, Fallback: 0}}
	state.Policy.ID = rsID(id)
	state.Policy.Name = name
	ruleOfForApprovals(state.Policy).RequiredApprovingReviewCount = paramInt(0)
	if both {
		state.Policy.Enforcement = rsText("disabled")
		state.EnforcementNormalization = ObservationNormalization{Path: RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
		*state.Policy.Rules = append(*state.Policy.Rules, RulesetRule{Type: "code_quality", Parameters: &RuleParameters{Severity: rsText("errors")}})
		state.QualitySeverityNormalization = ObservationNormalization{Path: RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"}
	}
	return state
}

func ruleOfForApprovals(policy RulesetDeclaration) *RuleParameters {
	for _, rule := range *policy.Rules {
		if rule.Type == "pull_request" {
			return rule.Parameters
		}
	}
	return nil
}

func TestRulesetRequiredApprovalsObservationOwnsTypedRawAndCompanions(t *testing.T) {
	for _, raw := range []int{-1, 11, 100} {
		for _, both := range []bool{false, true} {
			state := normalizedRequiredApprovalsState(7, "first", raw, both)
			other := normalizedRequiredApprovalsState(9, "other", 12, both)
			observed, err := NewRulesetObservation([]RulesetState{state, other}, 0)
			if err != nil {
				t.Fatal(err)
			}
			state.RequiredApprovalsNormalization.RawValue = 13
			*ruleOf(t, state.Policy, "pull_request").RequiredApprovingReviewCount = 9
			values, err := observed.Values()
			if err != nil {
				t.Fatal(err)
			}
			for i, want := range []int{raw, 12} {
				got, err := values[i].RawRequiredApprovals()
				if err != nil || got == nil || *got != want {
					t.Fatal("raw integer/ID lost", got, err)
				}
				*got = 14
				fresh, err := values[i].RawRequiredApprovals()
				if err != nil || fresh == nil || *fresh != want {
					t.Fatal("raw pointer alias", fresh, err)
				}
				if both {
					if enf, err := values[i].RawEnforcement(); err != nil || enf != "evaluate" {
						t.Fatal(enf, err)
					}
					if q, err := values[i].RawQualitySeverity(); err != nil || q == nil || *q != "FUTURE_SEVERITY" {
						t.Fatal(q, err)
					}
				} else if values[i].EnforcementNormalization != (ObservationNormalization{}) || values[i].QualitySeverityNormalization != (ObservationNormalization{}) {
					t.Fatal("invented companion")
				}
			}
			*ruleOf(t, values[0].Policy, "pull_request").RequiredApprovingReviewCount = 7
			values[0].RequiredApprovalsNormalization.RawValue = 99
			fresh, err := observed.Values()
			if err != nil {
				t.Fatal(err)
			}
			got, err := fresh[0].RawRequiredApprovals()
			if err != nil || got == nil || *got != raw || *ruleOf(t, fresh[0].Policy, "pull_request").RequiredApprovingReviewCount != 0 {
				t.Fatal("returned clone alias", fresh, got, err)
			}
			second, err := fresh[1].RawRequiredApprovals()
			if err != nil || second == nil || *second != 12 {
				t.Fatal("cross-ID corruption", second, err)
			}
			exported, err := observed.ExportDeclarations()
			if err != nil || exported.Len() != 2 || *ruleOf(t, exported.Values()[0], "pull_request").RequiredApprovingReviewCount != 0 {
				t.Fatal("effective export changed", exported, err)
			}
		}
	}
	for _, count := range []int{0, 10, 11} { // Ordinary native observations retain their native contract until acquisition projection.
		state := RulesetState{ID: 7, Policy: reviewPolicy()}
		ruleOf(t, state.Policy, "pull_request").RequiredApprovingReviewCount = paramInt(count)
		got, err := state.RawRequiredApprovals()
		if err != nil || got == nil || *got != count {
			t.Fatal("native raw changed", got, err)
		}
	}
	if got, err := (RulesetState{ID: 7, Policy: basicPolicy("absent")}).RawRequiredApprovals(); err != nil || got != nil {
		t.Fatal("absence invented count", got, err)
	}
	if got, err := (RulesetState{}).RawRequiredApprovals(); err == nil || got != nil {
		t.Fatal("unacquired became absence", got, err)
	}
}

func TestRulesetRequiredApprovalsObservationRejectsIncompleteAndForgedRecords(t *testing.T) {
	for _, mutate := range []func(*RulesetState){
		func(s *RulesetState) { s.ID = 0 }, func(s *RulesetState) { s.Policy.ID = rsID(9) }, func(s *RulesetState) { s.Policy.Rules = nil }, func(s *RulesetState) { s.Policy.Rules = &[]RulesetRule{} },
		func(s *RulesetState) { ruleOf(t, s.Policy, "pull_request").RequiredApprovingReviewCount = nil }, func(s *RulesetState) { ruleOf(t, s.Policy, "pull_request").RequiredApprovingReviewCount = paramInt(1) },
		func(s *RulesetState) { s.RequiredApprovalsNormalization.Path = "other" }, func(s *RulesetState) { s.RequiredApprovalsNormalization.RawValue = 0 }, func(s *RulesetState) { s.RequiredApprovalsNormalization.RawValue = 10 }, func(s *RulesetState) { s.RequiredApprovalsNormalization.Fallback = 1 },
		func(s *RulesetState) { s.EnforcementNormalization.Path = "other" }, func(s *RulesetState) { s.QualitySeverityNormalization.RawValue = "errors" },
	} {
		state := normalizedRequiredApprovalsState(7, "invalid", 11, true)
		mutate(&state)
		if got, err := state.RawRequiredApprovals(); err == nil || got != nil {
			t.Fatal("invalid raw integer returned", got, err)
		}
		if got, err := state.RawEnforcement(); err == nil || got != "" {
			t.Fatal("cross-record enforcement escaped", got, err)
		}
		if got, err := state.RawQualitySeverity(); err == nil || got != nil {
			t.Fatal("cross-record Quality escaped", got, err)
		}
		observed, err := NewRulesetObservation([]RulesetState{{ID: 9, Policy: basicPolicy("valid")}, state}, 0)
		if err == nil || !reflect.DeepEqual(observed, RulesetObservation{}) {
			t.Fatal("partial collection escaped", observed, err)
		}
	}
}
