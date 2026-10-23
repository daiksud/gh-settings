package actions_test

import (
	"encoding/json"
	"github.com/daiksud/gh-settings/internal/actions"
	"strings"
	"testing"
)

func TestPagesNativeNullableSummaryRejectsForgedValues(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Pages: []actions.PagesPlanChange{{Field: "pages.cname", Before: "example.com", After: nil}}}
	text, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(text, "→ null") {
		t.Fatal(text, err)
	}
	for _, c := range []actions.PagesPlanChange{{Field: "pages.build_type", Before: "workflow", After: "legacy"}, {Field: "pages.https_enforced", Before: true, After: nil}, {Field: "pages.public", Before: true, After: false}, {Field: "pages.cname", Before: "example.com", After: ""}, {Field: "pages.source.branch", BeforeInactive: true, After: "main"}, {Field: "pages.cname", Before: "same.example", After: "same.example"}} {
		bad := r
		bad.Pages = []actions.PagesPlanChange{c}
		if _, err := actions.RenderSummary(bad, summaryMetadata()); err == nil {
			t.Fatal("forged Pages accepted", c)
		}
	}
	for _, raw := range []string{`{"field":"pages.cname","after":"example.com"}`, `{"field":"pages.cname","before":"example.com"}`, `{"field":"pages.cname","before":"example.com","after":null,"before_inactive":null}`} {
		var v actions.PagesPlanChange
		if json.Unmarshal([]byte(raw), &v) == nil {
			t.Fatal("missing versus null lost", raw)
		}
	}
	r.Managed = 3
	r.Pages = []actions.PagesPlanChange{{Field: "pages.build_type", Before: "workflow", After: "legacy"}, {Field: "pages.source.branch", BeforeInactive: true, After: "main"}, {Field: "pages.source.path", BeforeInactive: true, After: "/docs"}}
	text, err = actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(text, "inactive →") {
		t.Fatal(text, err)
	}
}
