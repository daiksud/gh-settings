package settings

import "testing"

func TestEnvironmentPatternRawIdentityRetainsExactRefTypeAndName(t *testing.T) {
	for _, raw := range []string{"branch", "tag", "future", "", " \t", "Branch"} {
		state := EnvironmentPatternState{ID: 17, Pattern: EnvironmentPattern{Type: raw, Name: "Release/*"}}
		if raw != "branch" && raw != "tag" {
			state.Pattern.Type = "branch"
			state.RefTypeNormalization = ObservationNormalization{Path: EnvironmentRefTypePath, RawValue: raw, Fallback: "branch"}
		}
		got, err := state.RawPattern()
		if err != nil || got != (EnvironmentPattern{Type: raw, Name: "Release/*"}) {
			t.Fatalf("raw %q: got %#v, %v", raw, got, err)
		}
		got.Name, got.Type = "changed", "changed"
		again, err := state.RawPattern()
		if err != nil || again.Name != "Release/*" || again.Type != raw {
			t.Fatal("returned identity aliases state", again, err)
		}
	}
}

func TestEnvironmentPatternRawIdentityRejectsForgedOrIncompleteEvidence(t *testing.T) {
	valid := EnvironmentPatternState{ID: 17, Pattern: EnvironmentPattern{Type: "branch", Name: "Release/*"}, RefTypeNormalization: ObservationNormalization{Path: EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}}
	for _, mutate := range []func(*EnvironmentPatternState){
		func(s *EnvironmentPatternState) { s.ID = 0 },
		func(s *EnvironmentPatternState) { s.ID = -1 },
		func(s *EnvironmentPatternState) { s.Pattern.Name = "" },
		func(s *EnvironmentPatternState) { s.Pattern.Name = "bad\nname" },
		func(s *EnvironmentPatternState) { s.Pattern.Name = string([]byte{0xff}) },
		func(s *EnvironmentPatternState) { s.Pattern.Type = "tag" },
		func(s *EnvironmentPatternState) { s.Pattern.Type = "future" },
		func(s *EnvironmentPatternState) { s.RefTypeNormalization.Path = "wrong" },
		func(s *EnvironmentPatternState) { s.RefTypeNormalization.Path = "" },
		func(s *EnvironmentPatternState) { s.RefTypeNormalization.Fallback = "tag" },
		func(s *EnvironmentPatternState) { s.RefTypeNormalization.RawValue = "branch" },
		func(s *EnvironmentPatternState) { s.RefTypeNormalization.RawValue = "tag" },
	} {
		state := valid
		mutate(&state)
		if got, err := state.RawPattern(); err == nil || got != (EnvironmentPattern{}) {
			t.Fatalf("invalid provenance accepted: %#v, %#v, %v", state, got, err)
		}
	}
	for _, state := range []EnvironmentPatternState{{}, {ID: 17, Pattern: EnvironmentPattern{Type: "future", Name: "Release/*"}}, {ID: 17, Pattern: EnvironmentPattern{Type: "branch"}}} {
		if got, err := state.RawPattern(); err == nil || got != (EnvironmentPattern{}) {
			t.Fatal("incomplete raw identity accepted", state, got, err)
		}
	}
}
