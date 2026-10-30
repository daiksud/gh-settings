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
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesFolderReportPreservesProvenanceAndAtomicSourceCount(t *testing.T) {
	for _, branch := range []string{"main", "release/site"} {
		record := settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: "", Fallback: "root"}
		observed, err := settings.NewPagesObservationWithFolderNormalization(settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/"}}, record)
		if err != nil {
			t.Fatal(err)
		}
		managed, changes, err := settings.ComparePages(observed, settings.PagesDeclaration{Source: &settings.PagesSource{Branch: branch, Path: "/"}})
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
		want := 1
		if branch != "main" {
			want = 2
		}
		if report.Outcome != "changed" || report.Managed != 2 || len(report.Pages) != want || report.Pages[want-1].Normalization == nil || *report.Pages[want-1].Normalization != record {
			t.Fatal("lost atomic source evidence", string(data))
		}
		if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
			t.Fatal("source counted twice or report rejected", err)
		}
		invalid := report
		invalid.Managed = 1
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatal("source undercount accepted")
		}
		withHTTPS := report
		withHTTPS.Pages = append(append([]actions.PagesPlanChange{}, report.Pages...), actions.PagesPlanChange{Field: "pages.https_enforced", Before: false, After: true})
		if _, err := actions.RenderSummary(withHTTPS, summaryMetadata()); err == nil {
			t.Fatal("source plus HTTPS undercount accepted")
		}
		withHTTPS.Managed = 3
		if _, err := actions.RenderSummary(withHTTPS, summaryMetadata()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPagesFolderReportRejectsWrongScopeAndInactiveMetadata(t *testing.T) {
	record := settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"}
	base := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, Pages: []actions.PagesPlanChange{{Field: "pages.source.path", Before: "/", After: "/", Normalization: &record}}}
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) { r.Pages[0].Normalization = nil },
		func(r *actions.PlanReport) {
			r.Pages[0].Field = "pages.source.branch"
			r.Pages[0].Before = "main"
			r.Pages[0].After = "next"
		},
		func(r *actions.PlanReport) { r.Pages[0].After = "root" },
		func(r *actions.PlanReport) {
			forged := record
			forged.RawValue = "/"
			r.Pages[0].Normalization = &forged
		},
		func(r *actions.PlanReport) { r.Managed = 4; r.Pages = append(r.Pages, r.Pages[0]) },
		func(r *actions.PlanReport) {
			r.Managed = 3
			r.Pages = []actions.PagesPlanChange{{Field: "pages.build_type", Before: "workflow", After: "legacy"}, {Field: "pages.source.branch", Before: nil, After: "main", BeforeInactive: true}, {Field: "pages.source.path", Before: nil, After: "/", BeforeInactive: true, Normalization: &record}}
		},
	} {
		report := base
		report.Pages = append([]actions.PagesPlanChange{}, base.Pages...)
		mutate(&report)
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
			t.Fatalf("forged Pages metadata accepted: %+v", report)
		}
	}
	for _, section := range []string{"repository", "actions", "security"} {
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}}
		entry := actions.PlanChange{Field: "repository.has_issues", Before: false, After: true, Normalization: &record}
		switch section {
		case "repository":
			report.Changes = []actions.PlanChange{entry}
		case "actions":
			entry.Field = "actions.enabled"
			report.Actions = []actions.PlanChange{entry}
		case "security":
			entry.Field = "security.secret_scanning"
			report.Security = []actions.PlanChange{entry}
		}
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil || !strings.Contains(err.Error(), "normalization") {
			t.Fatal("wrong section ignored", section, err)
		}
	}
	for _, data := range []string{
		`{"field":"pages.source.path","before":"/","after":"/","normalization":{"path":"x","raw_value":null,"fallback":"root"}}`,
		`{"field":"pages.source.path","before":"/","after":"/","normalization":{"path":"x","fallback":"root"}}`,
		`{"field":"pages.source.path","before":"/","after":"/","normalization":{"path":"x","raw_value":"","fallback":false}}`,
		`{"field":"pages.source.path","before":"/","after":"/","normalization":{"path":"x","raw_value":"","fallback":"root","extra":true}}`,
	} {
		var change actions.PagesPlanChange
		if err := json.Unmarshal([]byte(data), &change); err == nil {
			t.Fatal("malformed metadata accepted", data)
		}
	}
}

func TestPagesFolderPresentationShowsProjectionAndKeepsPrivateStatus(t *testing.T) {
	raw := "/<script>PAGES_CANARY</script>\n::warning::untrusted&value"
	for _, desired := range []struct{ api, ui string }{{"/", "root"}, {"/docs", "docs"}} {
		record := settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: raw, Fallback: "root"}
		change, err := settings.NewPagesChangeWithNormalization("pages.source.path", "/", desired.api, false, record)
		if err != nil {
			t.Fatal(err)
		}
		comparison := settings.Comparison{Managed: 2, Pages: []settings.PagesChange{change}}
		output := string(cli.RenderComparison(comparison))
		expected := "Warning: " + settings.PagesFolderPath + ": observed " + strconv.Quote(raw) + " is outside UI choices; using \"root\" (projection only).\n" + settings.PagesFolderPath + ": \"root\" (normalized observation) → " + strconv.Quote(desired.ui) + " (write required)\n"
		if output != expected || strings.Contains(output, "\n::warning::") {
			t.Fatal("misleading or unsafe folder output", output, expected)
		}
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, Pages: []actions.PagesPlanChange{{Field: "pages.source.path", Before: "/", After: desired.api, Normalization: &record}}}
		summary, err := actions.RenderSummary(report, summaryMetadata())
		if err != nil || !strings.Contains(html.UnescapeString(summary), strings.TrimSuffix(expected, "\n")) || strings.Contains(summary, "<script>") {
			t.Fatal(summary, err)
		}
		path, privatePath := filepath.Join(t.TempDir(), "plan.json"), filepath.Join(t.TempDir(), "summary.md")
		data, _ := json.Marshal(report)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SETTINGS_TARGET", "owner/current")
		var stdout, stderr bytes.Buffer
		status := actions.Run([]string{"summary", path, privatePath, "status-only"}, &stdout, &stderr, "dev", nil)
		data, err = os.ReadFile(privatePath)
		if err != nil || status != 0 || stderr.Len() != 0 || stdout.String() != "plan-outcome=changed\n" || !strings.Contains(string(data), "Outcome: changed") || strings.Contains(string(data), "PAGES_CANARY") || strings.Contains(string(data), settings.PagesFolderPath) {
			t.Fatal(status, stdout.String(), stderr.String(), string(data), err)
		}
	}
}
