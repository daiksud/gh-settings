package actions_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestPagesLifecycleClosedReports(t *testing.T) {
	life := actions.PagesPlanChange{Field: "pages.lifecycle", Before: "absent", After: "present"}
	mode := actions.PagesPlanChange{Field: "pages.build_type", BeforeAbsent: true, After: "legacy"}
	branch := actions.PagesPlanChange{Field: "pages.source.branch", BeforeAbsent: true, After: "Main"}
	folder := actions.PagesPlanChange{Field: "pages.source.path", BeforeAbsent: true, After: "/docs"}
	domain := actions.PagesPlanChange{Field: "pages.cname", BeforeAbsent: true, After: nil}
	https := actions.PagesPlanChange{Field: "pages.https_enforced", BeforeAbsent: true, After: false}
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 6, Changes: []actions.PlanChange{}, Pages: []actions.PagesPlanChange{life, mode, branch, folder, domain, https}}
	text, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(text, "create site") || !strings.Contains(text, "absent → null") || !strings.Contains(text, "absent → false") {
		t.Fatal(text, err)
	}
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) { r.Pages = r.Pages[1:] },
		func(r *actions.PlanReport) { r.Pages = r.Pages[:3] },
		func(r *actions.PlanReport) { r.Pages[1].BeforeAbsent = false },
		func(r *actions.PlanReport) { r.Pages[2].Before = "invented" },
		func(r *actions.PlanReport) { r.Pages[2].After = "none" },
		func(r *actions.PlanReport) { r.Pages[2].BeforeInactive = true },
		func(r *actions.PlanReport) { r.Pages[1].After = "workflow" },
		func(r *actions.PlanReport) { r.Pages[4].After = false },
		func(r *actions.PlanReport) {
			r.Pages[0] = actions.PagesPlanChange{Field: "pages.lifecycle", Before: "present", After: "absent"}
		},
	} {
		bad := report
		bad.Pages = append([]actions.PagesPlanChange(nil), report.Pages...)
		mutate(&bad)
		if _, err := actions.RenderSummary(bad, summaryMetadata()); err == nil {
			t.Fatal("forged lifecycle accepted", bad.Pages)
		}
	}
	report.Managed = 1
	report.Pages = []actions.PagesPlanChange{{Field: "pages.lifecycle", Before: "present", After: "absent"}}
	text, err = actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(text, "delete site") || !strings.Contains(text, "branch: none") {
		t.Fatal(text, err)
	}
	for _, raw := range []string{`{"field":"pages.build_type","before":null,"after":"workflow","before_absent":null}`, `{"field":"pages.build_type","before":null,"after":"workflow","before_absent":"true"}`} {
		var entry actions.PagesPlanChange
		if json.Unmarshal([]byte(raw), &entry) == nil {
			t.Fatal("invalid absence marker accepted", raw)
		}
	}
}
