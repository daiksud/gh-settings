package actions_test

import (
	"bytes"
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebhookReportSelectedProjectionAndPrivacy(t *testing.T) {
	before, after := "https://before.example/private?A=%2f", "https://after.example/private?A=%2F&b=1"
	change := settings.WebhookChange{Before: settings.Webhook{ID: 9007199254740993, PayloadURL: &before}, After: settings.Webhook{ID: 9007199254740993, PayloadURL: &after}}
	path := filepath.Join(t.TempDir(), "report.json")
	var out, errout bytes.Buffer
	status := actions.Run([]string{"plan", path}, &out, &errout, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Webhooks: []settings.WebhookChange{change}}}, nil
	})
	if status != 0 {
		t.Fatal(errout.String())
	}
	data, _ := os.ReadFile(path)
	var report actions.PlanReport
	if e := json.Unmarshal(data, &report); e != nil {
		t.Fatal(e)
	}
	content, e := actions.RenderSummary(report, summaryMetadata())
	if e != nil || !strings.Contains(html.UnescapeString(content), after) || !strings.Contains(content, "9007199254740993") {
		t.Fatal(e, content)
	}
	for _, v := range []string{"secret", "active", "created_at", "insecure_ssl", "content_type"} {
		if strings.Contains(string(data), v) {
			t.Fatal("omitted/private value in report", v)
		}
	}
	for _, mode := range []string{"count", "unchanged", "duplicate", "shape"} {
		var bad actions.PlanReport
		json.Unmarshal(data, &bad)
		switch mode {
		case "count":
			bad.Managed = 0
		case "unchanged":
			bad.Outcome = "no-difference"
		case "duplicate":
			bad.Webhooks = append(bad.Webhooks, bad.Webhooks[0])
		case "shape":
			bad.Webhooks[0].Before.ContentType = &before
		}
		if _, e := actions.RenderSummary(bad, summaryMetadata()); e == nil {
			t.Fatal("invalid report", mode)
		}
	}
	// Existing collection differences are additive to the new managed family.
	report.Autolinks = []settings.AutolinkChange{{After: &settings.Autolink{ReferencePrefix: "EXT-", TargetURL: "https://example.test/<num>", ReferenceFormat: "numeric"}}}
	if _, e := actions.RenderSummary(report, summaryMetadata()); e == nil {
		t.Fatal("undercounted mixed families")
	}
	report.Managed = 2
	if _, e := actions.RenderSummary(report, summaryMetadata()); e != nil {
		t.Fatal(e)
	}
	// Runner and webhook changes retain independent collection evidence after composition.
	if err := json.Unmarshal([]byte(`[{"field":"repository.settings.actions.runners","before":{"id":23,"custom_labels":["old"]},"after":{"id":23,"custom_labels":["GPU"]}}]`), &report.Runners); err != nil {
		t.Fatal(err)
	}
	if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
		t.Fatal("undercounted runner/webhook/autolink families")
	}
	report.Managed = 3
	combined, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || !strings.Contains(combined, "GPU") || !strings.Contains(html.UnescapeString(combined), after) {
		t.Fatal("combined family presentation lost", err, combined)
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	summary := filepath.Join(t.TempDir(), "summary.md")
	if status := actions.Run([]string{"summary", path, summary, "status-only"}, &out, &errout, "dev", nil); status != 0 {
		t.Fatal(errout.String())
	}
	hidden, _ := os.ReadFile(summary)
	if strings.Contains(string(hidden)+out.String()+errout.String(), "private") || strings.Contains(string(hidden), "9007199254740993") {
		t.Fatal("status-only leaked")
	}
}
