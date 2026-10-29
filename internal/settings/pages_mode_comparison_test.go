package settings

import "testing"

func pagesModeComparisonObservation(t *testing.T) PagesObservation {
	t.Helper()
	o, err := NewPagesObservationWithModeNormalization(PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/"}, HTTPSEnforced: true}, ObservationNormalization{Path: PagesModePath, RawValue: "FUTURE_MODE", Fallback: "deploy_from_a_branch"}, ObservationNormalization{Path: PagesFolderPath, RawValue: "/future", Fallback: "root"})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestPagesModeComparisonKeepsRequiredWriteAndIndependentFolderEvidence(t *testing.T) {
	o := pagesModeComparisonObservation(t)
	for _, mode := range []string{"legacy", "workflow"} {
		managed, changes, err := ComparePages(o, PagesDeclaration{BuildType: pagesPtr(mode)})
		if err != nil || managed != 1 || len(changes) != 1 {
			t.Fatal(managed, changes, err)
		}
		before, after := changes[0].Values()
		if changes[0].Field() != "pages.build_type" || before != "legacy" || after != mode || changes[0].Normalization() != o.ModeNormalization() {
			t.Fatal(changes)
		}
		if err := ValidatePagesChanges(changes); err != nil {
			t.Fatal(err)
		}
	}
	for _, branch := range []string{"main", "other"} {
		managed, changes, err := ComparePages(o, PagesDeclaration{BuildType: pagesPtr("legacy"), Source: &PagesSource{Branch: branch, Path: "/"}})
		want := 2
		if branch != "main" {
			want = 3
		}
		if err != nil || managed != 3 || len(changes) != want || changes[0].Field() != "pages.build_type" || changes[0].Normalization() != o.ModeNormalization() || changes[len(changes)-1].Field() != "pages.source.path" || changes[len(changes)-1].Normalization() != o.FolderNormalization() {
			t.Fatal(managed, changes, err)
		}
		if err := ValidatePagesChanges(changes); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPagesModeComparisonOmissionAndInvalidEvidenceRemainDistinct(t *testing.T) {
	o := pagesModeComparisonObservation(t)
	managed, changes, err := ComparePages(o, PagesDeclaration{HTTPSEnforced: pagesPtr(false)})
	if err != nil || managed != 1 || len(changes) != 1 || changes[0].Field() != "pages.https_enforced" || changes[0].Normalization() != (ObservationNormalization{}) {
		t.Fatal(managed, changes, err)
	}
	if managed, changes, err := ComparePages(o, PagesDeclaration{}); err != nil || managed != 0 || len(changes) != 0 {
		t.Fatal(managed, changes, err)
	}
	if managed, changes, err := ComparePages(o, PagesDeclaration{Source: &PagesSource{Branch: "main", Path: "/"}}); err == nil || managed != 0 || changes != nil {
		t.Fatal("unknown source prerequisite lost", managed, changes, err)
	}
	for _, record := range []ObservationNormalization{
		{Path: PagesFolderPath, RawValue: "FUTURE_MODE", Fallback: "deploy_from_a_branch"},
		{Path: PagesModePath, RawValue: "legacy", Fallback: "deploy_from_a_branch"},
		{Path: PagesModePath, RawValue: "FUTURE_MODE", Fallback: "github_actions"},
	} {
		o.modeNormalization = record
		managed, changes, err := ComparePages(o, PagesDeclaration{BuildType: pagesPtr("legacy"), HTTPSEnforced: pagesPtr(false)})
		if err == nil || managed != 0 || changes != nil {
			t.Fatal("invalid managed metadata retained partial result", managed, changes, err)
		}
	}
}
