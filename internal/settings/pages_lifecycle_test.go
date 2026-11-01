package settings

import "testing"

func TestPagesLifecycleIntentAndAcquiredAbsence(t *testing.T) {
	absent := NewAbsentPagesObservation()
	if v, err := absent.Absent(); err != nil || !v {
		t.Fatal(v, err)
	}
	if _, err := absent.Values(); err == nil {
		t.Fatal("absence became configured values")
	}
	if _, err := (PagesObservation{}).Absent(); err == nil {
		t.Fatal("unacquired became absent")
	}
	exported, err := absent.Declaration()
	if err != nil || !exported.Delete() || exported.Validate() != nil {
		t.Fatal(exported, err)
	}
	for _, d := range []PagesDeclaration{
		{CNAME: &PagesDomain{}}, {HTTPSEnforced: pagesPtr(false)}, {BuildType: pagesPtr("legacy")}, {Source: &PagesSource{Branch: "main", Path: "/"}},
	} {
		if _, _, err := ComparePages(absent, d); err == nil {
			t.Fatalf("incomplete creation accepted: %+v", d)
		}
	}
	for _, d := range []PagesDeclaration{
		{Source: &PagesSource{Branch: "none"}},
		{BuildType: pagesPtr("workflow"), Source: &PagesSource{Branch: "none"}},
		{BuildType: pagesPtr("legacy"), Source: &PagesSource{Branch: "none", Path: "/"}},
		{BuildType: pagesPtr("legacy"), Source: &PagesSource{Branch: "none"}, CNAME: &PagesDomain{}},
		{BuildType: pagesPtr("legacy"), Source: &PagesSource{Branch: "none"}, HTTPSEnforced: pagesPtr(false)},
	} {
		if d.Validate() == nil {
			t.Fatalf("delete conflict accepted: %+v", d)
		}
	}
	present, _ := NewPagesObservation(PagesState{BuildType: "workflow", Public: true})
	if _, err := PreparePages(present, exported); err == nil {
		t.Fatal("deletion became configured state")
	}
	if n, c, err := ComparePages(absent, exported); err != nil || n != 1 || len(c) != 0 {
		t.Fatal(n, c, err)
	}
	if n, c, err := ComparePages(present, exported); err != nil || n != 1 || len(c) != 1 || ValidatePagesChanges(c) != nil {
		t.Fatal(n, c, err)
	}
	for _, d := range []PagesDeclaration{
		{BuildType: pagesPtr("workflow")},
		{BuildType: pagesPtr("legacy"), Source: &PagesSource{Branch: "None", Path: "/docs"}, CNAME: &PagesDomain{}, HTTPSEnforced: pagesPtr(false)},
	} {
		n, c, err := ComparePages(absent, d)
		if err != nil || n != len(c) || n < 2 || ValidatePagesChanges(c) != nil {
			t.Fatal(n, c, err)
		}
		for _, change := range c[1:] {
			b, _ := change.Values()
			if !change.BeforeAbsent() || b != nil {
				t.Fatal(change)
			}
		}
		if d.Delete() {
			t.Fatal("branch case was folded")
		}
	}
}
