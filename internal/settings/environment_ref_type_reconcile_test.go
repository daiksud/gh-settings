package settings

import (
	"reflect"
	"testing"
)

func TestEnvironmentPatternReconciliationUsesRawIdentityBeforeFallback(t *testing.T) {
	desired := []EnvironmentPattern{{Type: "branch", Name: "main"}}
	unknown := normalizedPattern(31, "future", "main")
	for _, test := range []struct {
		current []EnvironmentPatternState
		want    []EnvironmentPatternChange
	}{
		{nil, []EnvironmentPatternChange{{Operation: "create", Pattern: desired[0]}}},
		{[]EnvironmentPatternState{unknown}, []EnvironmentPatternChange{{Operation: "create", Pattern: desired[0]}, {Operation: "delete", ID: 31, Pattern: EnvironmentPattern{Type: "future", Name: "main"}}}},
		{[]EnvironmentPatternState{unknown, {ID: 32, Pattern: desired[0]}}, []EnvironmentPatternChange{{Operation: "delete", ID: 31, Pattern: EnvironmentPattern{Type: "future", Name: "main"}}}},
		{[]EnvironmentPatternState{unknown, normalizedPattern(17, "other", "main")}, []EnvironmentPatternChange{{Operation: "create", Pattern: desired[0]}, {Operation: "delete", ID: 17, Pattern: EnvironmentPattern{Type: "other", Name: "main"}}, {Operation: "delete", ID: 31, Pattern: EnvironmentPattern{Type: "future", Name: "main"}}}},
	} {
		got, err := ReconcileEnvironmentPatterns(test.current, &desired)
		if err != nil || !reflect.DeepEqual(got, test.want) {
			t.Fatal("fallback adopted an ID or collapsed raw identities", got, test.want, err)
		}
	}
	if got, err := ReconcileEnvironmentPatterns([]EnvironmentPatternState{unknown}, nil); err != nil || len(got) != 0 {
		t.Fatal("omission became pattern ownership", got, err)
	}
}

func TestEnvironmentPatternReconciliationRejectsAmbiguousRawEvidenceWithoutOperations(t *testing.T) {
	desired := []EnvironmentPattern{{Type: "branch", Name: "new"}}
	for _, current := range [][]EnvironmentPatternState{
		{normalizedPattern(31, "future", "main"), normalizedPattern(32, "future", "main")},
		{normalizedPattern(31, "future", "main"), normalizedPattern(31, "other", "main")},
		{normalizedPattern(0, "future", "main")},
		{normalizedPattern(31, "branch", "main")},
		{normalizedPattern(31, "future", "")},
	} {
		if got, err := ReconcileEnvironmentPatterns(current, &desired); err == nil || got != nil {
			t.Fatal("invalid raw evidence yielded operations", got, err)
		}
	}
}
