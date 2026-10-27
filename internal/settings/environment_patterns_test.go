package settings

import "testing"

func TestEnvironmentPatternIdentityAndExplicitCollection(t *testing.T) {
	current := []EnvironmentPatternState{{ID: 1, Pattern: EnvironmentPattern{Type: "branch", Name: "release/*"}}, {ID: 2, Pattern: EnvironmentPattern{Type: "tag", Name: "release/*"}}}
	if changes, err := ReconcileEnvironmentPatterns(current, nil); err != nil || len(changes) != 0 {
		t.Fatalf("omitted: %v %v", changes, err)
	}
	empty := []EnvironmentPattern{}
	if changes, err := ReconcileEnvironmentPatterns(current, &empty); err != nil || len(changes) != 2 || changes[0].ID != 1 || changes[1].ID != 2 {
		t.Fatalf("empty: %v %v", changes, err)
	}
	desired := []EnvironmentPattern{{Type: "tag", Name: "release/*"}, {Type: "branch", Name: "Release/*"}}
	changes, err := ReconcileEnvironmentPatterns(current, &desired)
	if err != nil || len(changes) != 2 || changes[0].Operation != "create" || changes[0].Pattern.Name != "Release/*" || changes[1].ID != 1 {
		t.Fatalf("typed case-sensitive identity: %v %v", changes, err)
	}
}

func TestEnvironmentPatternsRejectIncompleteOrAmbiguousObservation(t *testing.T) {
	desired := []EnvironmentPattern{}
	for _, entries := range [][]EnvironmentPatternState{
		{{ID: 1, Pattern: EnvironmentPattern{Name: "main"}}},
		{{ID: 0, Pattern: EnvironmentPattern{Type: "branch", Name: "main"}}},
		{{ID: 1, Pattern: EnvironmentPattern{Type: "branch", Name: "main"}}, {ID: 1, Pattern: EnvironmentPattern{Type: "tag", Name: "main"}}},
		{{ID: 1, Pattern: EnvironmentPattern{Type: "branch", Name: "main"}}, {ID: 2, Pattern: EnvironmentPattern{Type: "branch", Name: "main"}}},
	} {
		if _, err := ReconcileEnvironmentPatterns(entries, &desired); err == nil {
			t.Fatalf("accepted invalid observation: %+v", entries)
		}
	}
	for _, patterns := range [][]EnvironmentPattern{
		{{Type: "branch", Name: ""}},
		{{Type: "unknown", Name: "main"}},
		{{Type: "tag", Name: "main"}, {Type: "tag", Name: "main"}},
		{{Type: "branch", Name: "main\n"}},
	} {
		if _, err := ReconcileEnvironmentPatterns(nil, &patterns); err == nil {
			t.Fatalf("accepted invalid declaration: %+v", patterns)
		}
	}
}
