package settings

import (
	"reflect"
	"testing"
)

func normalizedBypassState(raw string, companions bool) RulesetState {
	state := RulesetState{ID: 7, Policy: basicPolicy("bypass")}
	if companions {
		state = normalizedRequiredApprovalsState(7, "bypass", 11, true)
	}
	state.Policy.BypassActors = &[]BypassActor{
		{ActorType: "User", ActorID: rsID(9), BypassMode: rsText("always")},
		{ActorType: "Team", ActorID: rsID(9), BypassMode: rsText("always")},
		{ActorType: "Integration", ActorID: rsID(17), BypassMode: rsText("pull_request")},
		{ActorType: "DeployKey", BypassMode: rsText("always")},
	}
	state.BypassModeNormalizations = []RulesetBypassModeNormalization{
		{ActorType: "DeployKey", Normalization: ObservationNormalization{Path: RulesetBypassModePath, RawValue: "NIL_ID_MODE", Fallback: "always"}},
		{ActorType: "Team", ActorID: rsID(9), Normalization: ObservationNormalization{Path: RulesetBypassModePath, RawValue: raw, Fallback: "always"}},
		{ActorType: "User", ActorID: rsID(9), Normalization: ObservationNormalization{Path: RulesetBypassModePath, RawValue: "USER_MODE", Fallback: "always"}},
	}
	return state
}

func TestRulesetBypassObservationOwnsIdentityBoundRawModes(t *testing.T) {
	for _, raw := range []string{"FUTURE_MODE", "", " \t", "for_pull_requests_only"} {
		for _, companions := range []bool{false, true} {
			want := []BypassActor{
				{ActorType: "DeployKey", BypassMode: rsText("NIL_ID_MODE")},
				{ActorType: "Integration", ActorID: rsID(17), BypassMode: rsText("pull_request")},
				{ActorType: "Team", ActorID: rsID(9), BypassMode: rsText(raw)},
				{ActorType: "User", ActorID: rsID(9), BypassMode: rsText("USER_MODE")},
			}
			state := normalizedBypassState(raw, companions)
			observed, err := NewRulesetObservation([]RulesetState{state}, 0)
			if err != nil {
				t.Fatal(err)
			}
			*state.BypassModeNormalizations[1].ActorID = 90
			state.BypassModeNormalizations[1].Normalization.RawValue = "mutated input"
			*(*state.Policy.BypassActors)[0].ActorID = 99
			*(*state.Policy.BypassActors)[1].BypassMode = "exempt"
			values, err := observed.Values()
			if err != nil {
				t.Fatal(err)
			}
			got, err := values[0].RawBypassActors()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatal("raw actor identities/modes lost", got, err)
			}
			*got[0].BypassMode = "mutated raw"
			*got[2].ActorID = 91
			again, err := values[0].RawBypassActors()
			if err != nil || !reflect.DeepEqual(again, want) {
				t.Fatal("raw actor pointer alias", again, err)
			}
			if companions {
				if enf, err := values[0].RawEnforcement(); err != nil || enf != "evaluate" {
					t.Fatal("enforcement companion lost", enf, err)
				}
				if quality, err := values[0].RawQualitySeverity(); err != nil || quality == nil || *quality != "FUTURE_SEVERITY" {
					t.Fatal("quality companion lost", quality, err)
				}
				if count, err := values[0].RawRequiredApprovals(); err != nil || count == nil || *count != 11 {
					t.Fatal("approvals companion lost", count, err)
				}
			} else if values[0].EnforcementNormalization != (ObservationNormalization{}) || values[0].QualitySeverityNormalization != (ObservationNormalization{}) || values[0].RequiredApprovalsNormalization != (RulesetRequiredApprovalsNormalization{}) {
				t.Fatal("scalar provenance invented")
			}
			*values[0].BypassModeNormalizations[1].ActorID = 92
			values[0].BypassModeNormalizations[2].Normalization.RawValue = "mutated output"
			*(*values[0].Policy.BypassActors)[0].BypassMode = "exempt"
			fresh, err := observed.Values()
			if err != nil {
				t.Fatal(err)
			}
			got, err = fresh[0].RawBypassActors()
			if err != nil || !reflect.DeepEqual(got, want) || *(*fresh[0].Policy.BypassActors)[0].BypassMode != "always" {
				t.Fatal("observation alias", got, err)
			}
		}
	}
}

func TestRulesetBypassObservationKeepsCanonicalAndKnownEmpty(t *testing.T) {
	for _, native := range []string{"always", "exempt", "pull_request"} {
		state := RulesetState{ID: 7, Policy: basicPolicy("canonical")}
		(*state.Policy.BypassActors)[0].BypassMode = rsText(native)
		got, err := state.RawBypassActors()
		if err != nil || !reflect.DeepEqual(got, *state.Policy.BypassActors) {
			t.Fatal("canonical mode changed", got, err)
		}
	}
	state := RulesetState{ID: 7, Policy: basicPolicy("empty")}
	state.Policy.BypassActors = &[]BypassActor{}
	if got, err := state.RawBypassActors(); err != nil || got == nil || len(got) != 0 {
		t.Fatal("known empty became unacquired", got, err)
	}
	state.Policy.BypassActors = nil
	if got, err := state.RawBypassActors(); err == nil || got != nil {
		t.Fatal("unacquired became empty", got, err)
	}
}

func TestRulesetBypassObservationRejectsIncompleteOrForgedEvidence(t *testing.T) {
	bypassOnly := normalizedBypassState("FUTURE_MODE", false)
	bypassOnly.Policy.ID = rsID(8)
	if got, err := bypassOnly.RawBypassActors(); err == nil || got != nil {
		t.Fatal("bypass-only evidence ignored policy ID", got, err)
	}
	if observed, err := NewRulesetObservation([]RulesetState{bypassOnly}, 0); err == nil || !reflect.DeepEqual(observed, RulesetObservation{}) {
		t.Fatal("bypass-only mismatched ID accepted", observed, err)
	}
	for _, mutate := range []func(*RulesetState){
		func(s *RulesetState) { s.ID = 0 },
		func(s *RulesetState) { s.Policy.ID = rsID(8) },
		func(s *RulesetState) { s.Policy.BypassActors = nil },
		func(s *RulesetState) { s.Policy.BypassActors = &[]BypassActor{} },
		func(s *RulesetState) { (*s.Policy.BypassActors)[0].BypassMode = nil },
		func(s *RulesetState) { (*s.Policy.BypassActors)[0].BypassMode = rsText("exempt") },
		func(s *RulesetState) { (*s.Policy.BypassActors)[0].ActorType = "UnknownActor" },
		func(s *RulesetState) { s.BypassModeNormalizations[2].ActorID = rsID(99) },
		func(s *RulesetState) { s.BypassModeNormalizations[2].ActorID = nil },
		func(s *RulesetState) { s.BypassModeNormalizations[0].ActorID = rsID(0) },
		func(s *RulesetState) { s.BypassModeNormalizations[2].ActorType = "Integration" },
		func(s *RulesetState) {
			s.BypassModeNormalizations[1].ActorType = "Team:9"
			s.BypassModeNormalizations[1].ActorID = nil
		},
		func(s *RulesetState) { s.BypassModeNormalizations[2].Normalization.Path = "other" },
		func(s *RulesetState) { s.BypassModeNormalizations[2].Normalization.Fallback = "exempt" },
		func(s *RulesetState) { s.BypassModeNormalizations[2].Normalization.RawValue = "always" },
		func(s *RulesetState) { s.BypassModeNormalizations[2].Normalization.RawValue = "exempt" },
		func(s *RulesetState) { s.BypassModeNormalizations[0].Normalization.RawValue = "pull_request" },
		func(s *RulesetState) {
			s.BypassModeNormalizations = append(s.BypassModeNormalizations, s.BypassModeNormalizations[1])
		},
		func(s *RulesetState) { s.EnforcementNormalization.RawValue = "active" },
		func(s *RulesetState) { s.QualitySeverityNormalization.Path = "other" },
		func(s *RulesetState) { s.RequiredApprovalsNormalization.RawValue = 0 },
	} {
		state := normalizedBypassState("FUTURE_MODE", true)
		mutate(&state)
		if got, err := state.RawBypassActors(); err == nil || got != nil {
			t.Fatal("invalid bypass evidence escaped", got, err)
		}
		if got, err := state.RawEnforcement(); err == nil || got != "" {
			t.Fatal("cross-record enforcement escaped", got, err)
		}
		if got, err := state.RawQualitySeverity(); err == nil || got != nil {
			t.Fatal("cross-record quality escaped", got, err)
		}
		if got, err := state.RawRequiredApprovals(); err == nil || got != nil {
			t.Fatal("cross-record approvals escaped", got, err)
		}
		observed, err := NewRulesetObservation([]RulesetState{{ID: 8, Policy: basicPolicy("valid")}, state}, 0)
		if err == nil || !reflect.DeepEqual(observed, RulesetObservation{}) {
			t.Fatal("partial collection escaped", observed, err)
		}
	}
}
