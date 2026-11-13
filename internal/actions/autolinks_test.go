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

func TestAutolinkReportSummaryAndPrivacy(t *testing.T) {
	id := int64(7)
	before := settings.Autolink{ID: &id, ReferencePrefix: "OLD_PRIVATE-", TargetURL: "https://private.example/<num>", ReferenceFormat: "numeric"}
	after := settings.Autolink{ReferencePrefix: "NEW_PRIVATE-", TargetURL: "https://private.example/<num>?view=A%2FB", ReferenceFormat: "alphanumeric"}
	comparison := settings.Comparison{Managed: 1, Autolinks: []settings.AutolinkChange{{After: &after}, {Before: &before}}}
	path := filepath.Join(t.TempDir(), "plan.json")
	var out, errout bytes.Buffer
	status := actions.Run([]string{"plan", path}, &out, &errout, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: comparison}, nil
	})
	if status != 0 {
		t.Fatal(errout.String())
	}
	data, _ := os.ReadFile(path)
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || report.Managed != 1 || !strings.Contains(html.UnescapeString(content), after.TargetURL) || !strings.Contains(content, "delete (id 7)") {
		t.Fatal(content, err)
	}
	for _, mode := range []string{"zero-count", "no-difference", "both-configs", "creation-id", "delete-no-id", "missing-format"} {
		var invalid actions.PlanReport
		json.Unmarshal(data, &invalid)
		switch mode {
		case "zero-count":
			invalid.Managed = 0
		case "no-difference":
			invalid.Outcome = "no-difference"
		case "both-configs":
			invalid.Autolinks[0].Before = &before
		case "creation-id":
			invalid.Autolinks[0].After.ID = &id
		case "delete-no-id":
			invalid.Autolinks[1].Before.ID = nil
		case "missing-format":
			invalid.Autolinks[0].After.ReferenceFormat = ""
		}
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatal("invalid report accepted", mode)
		}
	}
	t.Setenv("SETTINGS_TARGET", "owner/current")
	summary := filepath.Join(t.TempDir(), "summary.md")
	if status := actions.Run([]string{"summary", path, summary, "status-only"}, &out, &errout, "dev", nil); status != 0 {
		t.Fatal(errout.String())
	}
	private, _ := os.ReadFile(summary)
	if strings.Contains(string(private)+out.String()+errout.String(), "private.example") || strings.Contains(string(private), "PRIVATE") {
		t.Fatal(string(private))
	}
}
