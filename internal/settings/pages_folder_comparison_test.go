package settings

import "testing"

func TestPagesComparisonKeepsNormalizedFolderAndAtomicSourceCount(t *testing.T) {
	record := ObservationNormalization{Path: PagesFolderPath, RawValue: "/future", Fallback: "root"}
	observed, err := NewPagesObservationWithFolderNormalization(PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/"}, HTTPSEnforced: true}, record)
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range []string{"main", "release/site"} {
		for _, path := range []string{"/", "/docs"} {
			managed, changes, err := ComparePages(observed, PagesDeclaration{Source: &PagesSource{Branch: branch, Path: path}})
			wantChanges := 1
			if branch != "main" {
				wantChanges = 2
			}
			if err != nil || managed != 2 || len(changes) != wantChanges || changes[len(changes)-1].Field() != "pages.source.path" || changes[len(changes)-1].FolderNormalization() != record {
				t.Fatal(branch, path, managed, changes, err)
			}
			before, after := changes[len(changes)-1].Values()
			if before != "/" || after != path {
				t.Fatal(before, after)
			}
			if branch != "main" && (changes[0].Field() != "pages.source.branch" || changes[0].FolderNormalization() != (ObservationNormalization{})) {
				t.Fatal("branch order or ownership changed", changes)
			}
		}
	}
	for _, declaration := range []PagesDeclaration{{}, {HTTPSEnforced: securityBool(false)}, {BuildType: actionsUIPagesString("workflow")}} {
		managed, changes, err := ComparePages(observed, declaration)
		want := 0
		if !declaration.Empty() {
			want = 1
		}
		if err != nil || managed != want || len(changes) != want {
			t.Fatal("omitted source managed", managed, changes, err)
		}
		for _, change := range changes {
			if change.FolderNormalization() != (ObservationNormalization{}) {
				t.Fatal("unmanaged folder printed", change)
			}
		}
	}
	canonical, err := NewPagesObservation(PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/"}})
	if err != nil {
		t.Fatal(err)
	}
	if managed, changes, err := ComparePages(canonical, PagesDeclaration{Source: &PagesSource{Branch: "main", Path: "/"}}); err != nil || managed != 2 || len(changes) != 0 {
		t.Fatal(managed, changes, err)
	}
	workflow, err := NewPagesObservation(PagesState{BuildType: "workflow"})
	if err != nil {
		t.Fatal(err)
	}
	managed, changes, err := ComparePages(workflow, PagesDeclaration{BuildType: actionsUIPagesString("legacy"), Source: &PagesSource{Branch: "main", Path: "/"}})
	if err != nil || managed != 3 || len(changes) != 3 || !changes[1].BeforeInactive() || !changes[2].BeforeInactive() {
		t.Fatal("inactive transition changed", managed, changes, err)
	}
	for _, change := range changes {
		if change.FolderNormalization() != (ObservationNormalization{}) {
			t.Fatal(change)
		}
	}
}

func TestPagesComparisonDiscardsLaterInvalidFolderEvidence(t *testing.T) {
	for _, record := range []ObservationNormalization{
		{Path: "wrong", RawValue: "/future", Fallback: "root"},
		{Path: PagesFolderPath, RawValue: "/docs", Fallback: "root"},
		{Path: PagesFolderPath, RawValue: "/future", Fallback: "docs"},
	} {
		observed, err := NewPagesObservation(PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/"}})
		if err != nil {
			t.Fatal(err)
		}
		observed.folderNormalization = record
		managed, changes, err := ComparePages(observed, PagesDeclaration{Source: &PagesSource{Branch: "changed", Path: "/docs"}})
		if err == nil || managed != 0 || len(changes) != 0 {
			t.Fatal("earlier branch comparison escaped", managed, changes, err)
		}
	}
	workflow, err := NewPagesObservation(PagesState{BuildType: "workflow"})
	if err != nil {
		t.Fatal(err)
	}
	workflow.folderNormalization = ObservationNormalization{Path: PagesFolderPath, RawValue: "/future", Fallback: "root"}
	if managed, changes, err := ComparePages(workflow, PagesDeclaration{BuildType: actionsUIPagesString("legacy"), Source: &PagesSource{Branch: "main", Path: "/"}}); err == nil || managed != 0 || len(changes) != 0 {
		t.Fatal("mode or inactive branch escaped", managed, changes, err)
	}
}

func actionsUIPagesString(value string) *string { return &value }
