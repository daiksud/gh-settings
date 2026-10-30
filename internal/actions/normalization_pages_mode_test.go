package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesModePrivateReportRetainsClosedEvidenceAndAtomicSourceCount(t *testing.T) {
	for _, test := range []struct {
		name             string
		source, folder   bool
		branch           string
		managed, changes int
	}{
		{"mode only", false, false, "main", 1, 1},
		{"branch only source comparison", true, false, "other", 3, 2},
		{"normalized folder only source comparison", true, true, "main", 3, 2},
		{"both source comparisons", true, true, "other", 3, 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			modeRecord := settings.ObservationNormalization{Path: settings.PagesModePath, RawValue: "", Fallback: "deploy_from_a_branch"}
			folderRecord := settings.ObservationNormalization{}
			if test.folder {
				folderRecord = settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"}
			}
			o, err := settings.NewPagesObservationWithModeNormalization(settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/"}}, modeRecord, folderRecord)
			if err != nil {
				t.Fatal(err)
			}
			mode := "legacy"
			d := settings.PagesDeclaration{BuildType: &mode}
			if test.source {
				d.Source = &settings.PagesSource{Branch: test.branch, Path: "/"}
			}
			managed, changes, err := settings.ComparePages(o, d)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "plan.json")
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
				return app.PlanResult{Target: repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: managed, Pages: changes}}, nil
			})
			if status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatal(status, stdout.String(), stderr.String())
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var report actions.PlanReport
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.Managed != test.managed || len(report.Pages) != test.changes || report.Pages[0].Normalization == nil || *report.Pages[0].Normalization != modeRecord || report.Pages[0].Before != "legacy" || report.Pages[0].After != "legacy" {
				t.Fatal(string(data))
			}
			if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
				t.Fatal("valid source count/evidence rejected", err)
			}
			if test.source {
				invalid := report
				invalid.Managed = 2
				if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
					t.Fatal("atomic source undercount accepted")
				}
			}
			withHTTPS := report
			withHTTPS.Pages = append(append([]actions.PagesPlanChange{}, report.Pages...), actions.PagesPlanChange{Field: "pages.https_enforced", Before: false, After: true})
			if _, err := actions.RenderSummary(withHTTPS, summaryMetadata()); err == nil {
				t.Fatal("extra comparison undercount accepted")
			}
		})
	}
}

func TestPagesModeReportRejectsForgedEvidenceAndHidesPrivateValues(t *testing.T) {
	raw := "<script>MODE_CANARY</script>\n::warning::untrusted"
	record := settings.ObservationNormalization{Path: settings.PagesModePath, RawValue: raw, Fallback: "deploy_from_a_branch"}
	base := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Pages: []actions.PagesPlanChange{{Field: "pages.build_type", Before: "legacy", After: "workflow", Normalization: &record}}}
	summary, err := actions.RenderSummary(base, summaryMetadata())
	warning := "Warning: repository.settings.pages.build_and_deployment.source: observed " + strconv.Quote(raw) + " is outside UI choices; using \"deploy_from_a_branch\" (projection only)."
	if err != nil || strings.Count(html.UnescapeString(summary), warning) != 1 || !strings.Contains(html.UnescapeString(summary), `"deploy_from_a_branch" (normalized observation) → "github_actions" (write required) (branch source becomes inactive)`) || strings.Contains(summary, "<script>") {
		t.Fatal(summary, err)
	}
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) { r.Pages[0].Before = "workflow" },
		func(r *actions.PlanReport) { r.Pages[0].After = "github_actions" },
		func(r *actions.PlanReport) { r.Pages[0].BeforeInactive = true },
		func(r *actions.PlanReport) { r.Pages[0].Field = "pages.source.path" },
		func(r *actions.PlanReport) { bad := record; bad.RawValue = "legacy"; r.Pages[0].Normalization = &bad },
		func(r *actions.PlanReport) {
			bad := record
			bad.Fallback = "github_actions"
			r.Pages[0].Normalization = &bad
		},
		func(r *actions.PlanReport) { r.Managed = 2; r.Pages = append(r.Pages, r.Pages[0]) },
		func(r *actions.PlanReport) {
			r.Managed = 3
			r.Pages = append(r.Pages, actions.PagesPlanChange{Field: "pages.source.branch", Before: "main", After: "other"})
		},
	} {
		r := base
		r.Pages = append([]actions.PagesPlanChange{}, base.Pages...)
		mutate(&r)
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatal("forged mode evidence accepted", r)
		}
	}
	path, private := filepath.Join(t.TempDir(), "plan.json"), filepath.Join(t.TempDir(), "status.md")
	data, _ := json.Marshal(base)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"summary", path, private, "status-only"}, &stdout, &stderr, "dev", nil)
	data, err = os.ReadFile(private)
	if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(data), "MODE_CANARY") || strings.Contains(string(data), settings.PagesModePath) {
		t.Fatal(status, stdout.String(), stderr.String(), string(data), err)
	}
	for _, section := range []string{"repository", "actions", "security"} {
		r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}}
		entry := actions.PlanChange{Field: "repository.has_issues", Before: false, After: true, Normalization: &record}
		switch section {
		case "repository":
			r.Changes = []actions.PlanChange{entry}
		case "actions":
			entry.Field = "actions.enabled"
			r.Actions = []actions.PlanChange{entry}
		case "security":
			entry.Field = "security.secret_scanning"
			r.Security = []actions.PlanChange{entry}
		}
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatal("mode metadata wrongsection accepted", section)
		}
	}
}
