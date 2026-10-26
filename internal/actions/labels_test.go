package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestLabelPlanReportAndSummary(t *testing.T) {
	empty, color, id := "", "abcdef", int64(1)
	op := labelOperation(t)
	changes := labelSequence(t,
		op(settings.NewLabelCreate(settings.LabelDeclaration{Name: "<new>", Color: &color})),
		op(settings.NewLabelUpdate(settings.LabelState{ID: 1, Name: "existing", Color: "abcdef", Description: "</pre><script>"}, settings.LabelDeclaration{ID: &id, Name: "existing", Color: &color, Description: &empty})),
	)
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Labels: changes}}, nil
	})
	data, err := os.ReadFile(path)
	if status != 0 || err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d data=%s err=%v stderr=%s", status, data, err, &stderr)
	}
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "changed" || report.Managed != 1 || len(report.Labels) != 2 || len(report.Changes) != 0 {
		t.Fatalf("report=%+v", report)
	}
	if report.Labels[0].After.Color == nil || report.Labels[0].After.Description == nil || report.Labels[1].ID == nil || *report.Labels[1].ID != 1 || report.Labels[1].After.Description == nil || *report.Labels[1].After.Description != "" {
		t.Fatalf("lost attribute presence: %s", data)
	}
	rendered, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || strings.Contains(rendered, "<new>") || strings.Contains(rendered, "</pre><script>") {
		t.Fatalf("rendered=%s err=%v", rendered, err)
	}
	visible := html.UnescapeString(rendered)
	for _, want := range []string{`settings.labels: create "<new>"`, `settings.labels: update "existing" (id 1) description: "</pre><script>" → ""`} {
		if !strings.Contains(visible, want) {
			t.Fatalf("missing %s in %s", want, visible)
		}
	}
}

func TestLabelMalformedReportsPreserveSummary(t *testing.T) {
	for _, labels := range []string{
		`[null]`,
		`[{}]`,
		`[{"operation":"rename","name":"a","after":{}}]`,
		`[{"operation":"create","name":"a"}]`,
		`[{"operation":"create","name":"a","after":null}]`,
		`[{"operation":"create","name":"a","before":{"color":"abcdef","description":""},"after":{}}]`,
		`[{"operation":"delete","name":"a"}]`,
		`[{"operation":"delete","name":"a","before":{"color":"ABCDEF","description":""}}]`,
		`[{"operation":"delete","name":"a","before":{"color":"abcdef","description":""},"after":{}}]`,
		`[{"operation":"update","name":"a","after":{"description":"new"}}]`,
		`[{"operation":"update","name":"a","before":{"color":"abcdef","description":"old"}}]`,
		`[{"operation":"update","name":"a","before":{"color":"abcdef","description":"old"},"after":{}}]`,
		`[{"operation":"create","name":"a","after":{"color":null}}]`,
		`[{"operation":"create","name":"a","after":{"description":null}}]`,
		`[{"operation":"create","name":"a","after":{"color":123456}}]`,
		`[{"operation":"create","name":"a","after":{"new_name":"b"}}]`,
		`[{"operation":"create","name":"a","new_name":"b","after":{}}]`,
		`[{"operation":"create","name":"a","after":{"color":"ABCDEF"}}]`,
		`[{"operation":"delete","name":"a","before":{"color":"abcdef"}}]`,
		`[{"operation":"update","name":"a","before":{"color":"abcdef","description":""},"after":{"description":""}}]`,
		`[{"operation":"create","name":"a","after":{}},{"operation":"delete","name":"A","before":{"color":"abcdef","description":""}}]`,
	} {
		t.Run(labels, func(t *testing.T) {
			setSummaryEnvironment(t)
			dir := t.TempDir()
			report := filepath.Join(dir, "plan.json")
			summary := filepath.Join(dir, "summary.md")
			body := `{"outcome":"changed","target":"owner/current","managed":1,"changes":[],"labels":` + labels + `}`
			if err := os.WriteFile(report, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(summary, []byte("previous\n"), 0600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"summary", report, summary}, &stdout, &stderr, "dev", nil)
			data, err := os.ReadFile(summary)
			if status != 1 || stdout.Len() != 0 || stderr.Len() == 0 || err != nil || string(data) != "previous\n" {
				t.Fatalf("status=%d err=%v stderr=%s summary=%s", status, err, &stderr, data)
			}
		})
	}
}

func labelOperation(t *testing.T) func(settings.LabelChange, error) settings.LabelChange {
	t.Helper()
	return func(change settings.LabelChange, err error) settings.LabelChange {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return change
	}
}

func labelSequence(t *testing.T, changes ...settings.LabelChange) settings.LabelChanges {
	t.Helper()
	result, err := settings.NewLabelChanges(changes)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
