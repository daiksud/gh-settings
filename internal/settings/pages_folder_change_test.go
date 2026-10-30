package settings

import "testing"

func TestNormalizedPagesFolderChangeKeepsRequiredWrites(t *testing.T) {
	for _, raw := range []string{"/future", "", " \t"} {
		for _, desired := range []string{"/", "/docs"} {
			record := ObservationNormalization{Path: PagesFolderPath, RawValue: raw, Fallback: "root"}
			change, err := NewPagesChangeWithNormalization("pages.source.path", "/", desired, false, record)
			if err != nil {
				t.Fatal(err)
			}
			before, after := change.Values()
			if change.Field() != "pages.source.path" || before != "/" || after != desired || change.BeforeInactive() || change.Normalization() != record {
				t.Fatal(change)
			}
			if err := ValidatePagesChanges([]PagesChange{change}); err != nil {
				t.Fatal(err)
			}
			record.RawValue = "mutated input"
			owned := change.Normalization()
			owned.RawValue = "mutated output"
			if change.Normalization().RawValue != raw {
				t.Fatal("aliased metadata")
			}
		}
	}
	if _, err := NewPagesChange("pages.source.path", "/", "/", false); err == nil {
		t.Fatal("ordinary equality allowed")
	}
}

func TestNormalizedPagesFolderChangeRejectsWrongScopeAndEvidence(t *testing.T) {
	record := ObservationNormalization{Path: PagesFolderPath, RawValue: "/future", Fallback: "root"}
	for _, test := range []struct {
		field         string
		before, after any
		inactive      bool
		record        ObservationNormalization
	}{
		{"pages.source.path", "/docs", "/", false, record},
		{"pages.source.path", nil, "/", true, record},
		{"pages.source.path", "/", "/", true, record},
		{"pages.source.path", "/", false, false, record},
		{"pages.source.path", false, "/", false, record},
		{"pages.source.path", "/", "root", false, record},
		{"pages.source.path", "/", "/future", false, record},
		{"pages.source.path", "/", "/", false, ObservationNormalization{}},
		{"pages.source.path", "/", "/", false, ObservationNormalization{Path: PagesFolderPath, RawValue: "/", Fallback: "root"}},
		{"pages.source.path", "/", "/", false, ObservationNormalization{Path: PagesFolderPath, RawValue: "/docs", Fallback: "root"}},
		{"pages.source.path", "/", "/", false, ObservationNormalization{Path: PagesFolderPath, RawValue: "/future", Fallback: "docs"}},
		{"pages.source.path", "/", "/", false, ObservationNormalization{Path: "other", RawValue: "/future", Fallback: "root"}},
		{"pages.source.branch", "main", "next", false, record},
		{"pages.build_type", "legacy", "workflow", false, record},
		{"pages.cname", nil, "site.example.com", false, record},
	} {
		change, err := NewPagesChangeWithNormalization(test.field, test.before, test.after, test.inactive, test.record)
		if err == nil || change.Field() != "" || change.Normalization() != (ObservationNormalization{}) {
			t.Fatal(test, change, err)
		}
	}
	change, err := NewPagesChangeWithNormalization("pages.source.path", "/", "/", false, record)
	if err != nil {
		t.Fatal(err)
	}
	if ValidatePagesChanges([]PagesChange{change, change}) == nil {
		t.Fatal("duplicate normalized folder accepted")
	}
	for _, pair := range [][2]string{{"workflow", "legacy"}, {"legacy", "workflow"}} {
		mode, err := NewPagesChange("pages.build_type", pair[0], pair[1], false)
		if err != nil {
			t.Fatal(err)
		}
		if ValidatePagesChanges([]PagesChange{mode, change}) == nil {
			t.Fatal("normalized folder bypassed transition guards", pair)
		}
	}
}
