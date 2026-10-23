package settings

import "testing"

func pagesPtr[T any](v T) *T { return &v }
func TestPagesOwnershipTransitionsAndCopies(t *testing.T) {
	source := &PagesSource{Branch: "main", Path: "/docs"}
	domain := "EXAMPLE.COM"
	o, err := NewPagesObservation(PagesState{BuildType: "legacy", Source: source, CNAME: &domain, HTTPSEnforced: true, Public: true})
	if err != nil {
		t.Fatal(err)
	}
	source.Branch = "changed"
	domain = "changed.example"
	d, err := o.Declaration()
	if err != nil || *d.CNAME.Value != "example.com" || d.Source.Branch != "main" {
		t.Fatal(d, err)
	}
	n, c, err := ComparePages(o, d)
	if err != nil || n != 5 || len(c) != 0 {
		t.Fatal(n, c, err)
	}
	d.Source.Branch = "edited"
	s, _ := o.Values()
	if s.Source.Branch != "main" {
		t.Fatal("observation mutated")
	}
	n, c, err = ComparePages(o, PagesDeclaration{CNAME: &PagesDomain{}, HTTPSEnforced: pagesPtr(false)})
	if err != nil || n != 2 || len(c) != 2 {
		t.Fatal(n, c, err)
	}
	b, a := c[0].Values()
	if b != "example.com" || a != nil {
		t.Fatal(b, a)
	}
	if _, _, err := ComparePages(PagesObservation{}, PagesDeclaration{}); err == nil {
		t.Fatal("unknown observation accepted")
	}
	workflow, _ := NewPagesObservation(PagesState{BuildType: "workflow", Source: &PagesSource{Branch: "stale", Path: "/"}})
	if _, err := PreparePages(workflow, PagesDeclaration{BuildType: pagesPtr("legacy")}); err == nil {
		t.Fatal("stale source activated")
	}
	if _, err := PreparePages(workflow, PagesDeclaration{Source: &PagesSource{Branch: "main", Path: "/"}}); err == nil {
		t.Fatal("workflow source accepted")
	}
	n, c, err = ComparePages(workflow, PagesDeclaration{BuildType: pagesPtr("legacy"), Source: &PagesSource{Branch: "main", Path: "/"}})
	if err != nil || n != 3 || len(c) != 3 || !c[1].BeforeInactive() || ValidatePagesChanges(c) != nil {
		t.Fatal(n, c, err)
	}
	if ValidatePagesChanges(c[1:]) == nil {
		t.Fatal("inactive source without mode accepted")
	}
	result, err := PreparePages(o, PagesDeclaration{BuildType: pagesPtr("workflow")})
	if err != nil || result.Source != nil || !result.Public {
		t.Fatal(result, err)
	}
	for _, bad := range []string{"", "null", "https://example.com", "example.com.", "*.example.com", "a..com", "-a.example", "a_.example", "é.example"} {
		if ValidatePagesDomain(bad) == nil {
			t.Fatal("invalid domain", bad)
		}
	}
	for _, bad := range []PagesDeclaration{{BuildType: pagesPtr("")}, {Source: &PagesSource{Branch: "HEAD", Path: "/"}}, {BuildType: pagesPtr("workflow"), Source: &PagesSource{Branch: "main", Path: "/"}}} {
		if bad.Validate() == nil {
			t.Fatal("invalid declaration", bad)
		}
	}
}
