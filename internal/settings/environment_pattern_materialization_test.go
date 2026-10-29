package settings

import (
	"reflect"
	"testing"
)

func normalizedPattern(id int64, raw, name string) EnvironmentPatternState {
	return EnvironmentPatternState{ID: id, Pattern: EnvironmentPattern{Type: "branch", Name: name}, RefTypeNormalization: ObservationNormalization{Path: EnvironmentRefTypePath, RawValue: raw, Fallback: "branch"}}
}

func TestEnvironmentPatternMaterializationOwnsCompleteCanonicalProjection(t *testing.T) {
	current := []EnvironmentPatternState{normalizedPattern(7, "future", "Release/*"), {ID: 8, Pattern: EnvironmentPattern{Type: "tag", Name: "release/*"}}, {ID: 9, Pattern: EnvironmentPattern{Type: "branch", Name: "release/*"}}}
	before := append([]EnvironmentPatternState{}, current...)
	want := []EnvironmentPattern{{Type: "branch", Name: "Release/*"}, {Type: "branch", Name: "release/*"}, {Type: "tag", Name: "release/*"}}
	if err := ValidateEnvironmentPatternObservation(current); err != nil {
		t.Fatal(err)
	}
	got, err := EffectiveEnvironmentPatterns(current)
	if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(current, before) {
		t.Fatal("projection or input changed", got, err, current)
	}
	got[0].Name = "changed"
	again, err := EffectiveEnvironmentPatterns(current)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatal("output aliases acquired state", again, err)
	}
	got, err = EffectiveEnvironmentPatterns([]EnvironmentPatternState{})
	if err != nil || got == nil || len(got) != 0 {
		t.Fatal("known empty lost", got, err)
	}
}

func TestEnvironmentPatternMaterializationKeepsRawCollisionsWithoutCollapsing(t *testing.T) {
	for _, other := range []EnvironmentPatternState{{ID: 8, Pattern: EnvironmentPattern{Type: "branch", Name: "main"}}, normalizedPattern(8, "other", "main")} {
		current := []EnvironmentPatternState{normalizedPattern(7, "future", "main"), other}
		before := append([]EnvironmentPatternState{}, current...)
		if err := ValidateEnvironmentPatternObservation(current); err != nil {
			t.Fatal("distinct raw identities rejected", err)
		}
		if got, err := EffectiveEnvironmentPatterns(current); err == nil || got != nil || !reflect.DeepEqual(current, before) {
			t.Fatal("collision collapsed or mutated", got, err, current)
		}
	}
}

func TestEnvironmentPatternMaterializationRejectsIncompleteOrAmbiguousAcquisition(t *testing.T) {
	for _, current := range [][]EnvironmentPatternState{
		nil,
		{normalizedPattern(0, "future", "main")},
		{normalizedPattern(7, "future", "main"), normalizedPattern(7, "other", "main")},
		{normalizedPattern(7, "future", "main"), normalizedPattern(8, "future", "main")},
		{{ID: 7, Pattern: EnvironmentPattern{Type: "branch", Name: "main"}}, {ID: 8, Pattern: EnvironmentPattern{Type: "branch", Name: "main"}}},
		{normalizedPattern(7, "future", "")},
	} {
		if err := ValidateEnvironmentPatternObservation(current); err == nil {
			t.Fatal("invalid acquisition accepted", current)
		}
		if got, err := EffectiveEnvironmentPatterns(current); err == nil || got != nil {
			t.Fatal("invalid acquisition materialized", got, err)
		}
	}
}
