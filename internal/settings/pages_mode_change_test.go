package settings

import "testing"

func TestPagesModeChangeRetainsActionableProjectionWithoutInactiveSource(t *testing.T) {
	for _, raw := range []string{"future", "", " \t"} {
		record := ObservationNormalization{Path: PagesModePath, RawValue: raw, Fallback: "deploy_from_a_branch"}
		for _, desired := range []string{"legacy", "workflow"} {
			change, err := NewPagesModeChangeWithNormalization("pages.build_type", "legacy", desired, false, record)
			if err != nil {
				t.Fatal(err)
			}
			before, after := change.Values()
			if change.Field() != "pages.build_type" || before != "legacy" || after != desired || change.BeforeInactive() || change.Normalization() != record {
				t.Fatal(change, before, after)
			}
			if err := ValidatePagesChanges([]PagesChange{change}); err != nil {
				t.Fatal("invented source prerequisite", err)
			}
			copy := change.Normalization()
			copy.RawValue = "mutated"
			if change.Normalization().RawValue != raw {
				t.Fatal("mutable provenance")
			}
			if err := ValidatePagesChanges([]PagesChange{change, change}); err == nil {
				t.Fatal("duplicate accepted")
			}
		}
	}
	if _, err := NewPagesChange("pages.build_type", "legacy", "legacy", false); err == nil {
		t.Fatal("ordinary equality accepted")
	}
}

func TestPagesModeChangeRejectsWrongFieldStateAndProvenance(t *testing.T) {
	valid := ObservationNormalization{Path: PagesModePath, RawValue: "future", Fallback: "deploy_from_a_branch"}
	for _, test := range []struct {
		field         string
		before, after any
		inactive      bool
		record        ObservationNormalization
	}{
		{"pages.source.path", "legacy", "legacy", false, valid},
		{"pages.build_type", "workflow", "legacy", false, valid},
		{"pages.build_type", "future", "legacy", false, valid},
		{"pages.build_type", "legacy", "future", false, valid},
		{"pages.build_type", nil, "legacy", false, valid},
		{"pages.build_type", "legacy", true, false, valid},
		{"pages.build_type", "legacy", "legacy", true, valid},
		{"pages.build_type", "legacy", "legacy", false, ObservationNormalization{}},
		{"pages.build_type", "legacy", "legacy", false, ObservationNormalization{Path: PagesModePath, RawValue: "legacy", Fallback: "deploy_from_a_branch"}},
		{"pages.build_type", "legacy", "legacy", false, ObservationNormalization{Path: PagesModePath, RawValue: "workflow", Fallback: "deploy_from_a_branch"}},
		{"pages.build_type", "legacy", "legacy", false, ObservationNormalization{Path: PagesFolderPath, RawValue: "future", Fallback: "deploy_from_a_branch"}},
		{"pages.build_type", "legacy", "legacy", false, ObservationNormalization{Path: PagesModePath, RawValue: "future", Fallback: "github_actions"}},
	} {
		change, err := NewPagesModeChangeWithNormalization(test.field, test.before, test.after, test.inactive, test.record)
		if err == nil || change.Field() != "" || change.Normalization() != (ObservationNormalization{}) {
			t.Fatal(test, change, err)
		}
	}
	change, err := NewPagesModeChangeWithNormalization("pages.build_type", "legacy", "workflow", false, valid)
	if err != nil {
		t.Fatal(err)
	}
	branch, err := NewPagesChange("pages.source.branch", "main", "other", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePagesChanges([]PagesChange{change, branch}); err == nil {
		t.Fatal("workflow target accepts source")
	}
	toLegacy, err := NewPagesChange("pages.build_type", "workflow", "legacy", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePagesChanges([]PagesChange{toLegacy}); err == nil {
		t.Fatal("real workflow transition omitted inactive pair")
	}
}
