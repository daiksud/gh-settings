package settings

import "testing"

func TestNormalizedCreationPolicyChangeRequiresConsistentEvidence(t *testing.T) {
	for _, test := range []struct{ field, path string }{
		{"repository.pull_request_creation_policy", "repository.settings.general.features.pull_request_permissions.creation_allowed_by"},
		{"repository.issue_creation_policy", "repository.settings.general.features.issue_permissions.creation_allowed_by"},
	} {
		t.Run(test.field, func(t *testing.T) {
			field := FieldID{path: test.field}
			for _, raw := range []string{"FUTURE", "", " \t "} {
				for _, after := range []string{"ALL", "COLLABORATORS_ONLY"} {
					record := ObservationNormalization{Path: test.path, RawValue: raw, Fallback: "all_users"}
					change, err := NewCreationPolicyChangeWithNormalization(field, "ALL", after, record)
					if err != nil || change.Field() != field || change.Normalization() != record {
						t.Fatal("valid required write rejected", raw, after, change, err)
					}
					called := false
					change.Visit(func(bool, bool) { t.Fatal("boolean change") }, func(before, got string) {
						called = true
						if before != "ALL" || got != after {
							t.Fatal("native visitor changed", before, got)
						}
					}, func([]string, []string) { t.Fatal("topics change") })
					if !called {
						t.Fatal("missing native pair")
					}
					record.RawValue = "caller mutation"
					if change.Normalization().RawValue != raw {
						t.Fatal("constructor borrowed provenance")
					}
				}
			}
			for _, bad := range []struct {
				field         FieldID
				before, after string
				record        ObservationNormalization
			}{
				{field, "ALL", "ALL", ObservationNormalization{}},
				{field, "ALL", "ALL", ObservationNormalization{Path: "wrong", RawValue: "FUTURE", Fallback: "all_users"}},
				{field, "ALL", "ALL", ObservationNormalization{Path: test.path, RawValue: "FUTURE", Fallback: "collaborators_only"}},
				{field, "ALL", "ALL", ObservationNormalization{Path: test.path, RawValue: "ALL", Fallback: "all_users"}},
				{field, "ALL", "ALL", ObservationNormalization{Path: test.path, RawValue: "COLLABORATORS_ONLY", Fallback: "all_users"}},
				{field, "COLLABORATORS_ONLY", "ALL", ObservationNormalization{Path: test.path, RawValue: "FUTURE", Fallback: "all_users"}},
				{field, "ALL", "FUTURE", ObservationNormalization{Path: test.path, RawValue: "FUTURE", Fallback: "all_users"}},
				{FieldID{path: "repository.description"}, "ALL", "ALL", ObservationNormalization{Path: test.path, RawValue: "FUTURE", Fallback: "all_users"}},
			} {
				change, err := NewCreationPolicyChangeWithNormalization(bad.field, bad.before, bad.after, bad.record)
				if err == nil || change.Field() != (FieldID{}) || change.Normalization() != (ObservationNormalization{}) {
					t.Fatal("inconsistent evidence manufactured change", bad, change, err)
				}
			}
			ordinary, err := NewStringChange(field, "ALL", "ALL")
			if err != nil || ordinary.Normalization() != (ObservationNormalization{}) {
				t.Fatal("ordinary constructor contract changed", ordinary, err)
			}
		})
	}
}
