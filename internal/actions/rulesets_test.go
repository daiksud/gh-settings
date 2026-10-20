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
	"github.com/daiksud/gh-settings/internal/workspace"
)

func TestRulesetPlanReportAndSummary(t *testing.T) {
	d, err := workspace.DecodeConfig([]byte(`schemaVersion: 1
settings: {rulesets: [{name: '<policy>', target: branch, enforcement: active, conditions: {ref_name: {include: ['~ALL'], exclude: []}}, bypass_actors: [], rules: [{type: deletion}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	before := d.Rulesets.Values()[0]
	change, err := settings.NewRulesetChange("delete", 7, &before, nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Rulesets: []settings.RulesetChange{change}}}, nil
	})
	data, err := os.ReadFile(path)
	if status != 0 || err != nil {
		t.Fatal(status, err, stderr.String())
	}
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "changed" || len(report.Rulesets) != 1 || report.Rulesets[0].Before == nil {
		t.Fatal(string(data))
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || strings.Contains(content, "<policy>") || !strings.Contains(html.UnescapeString(content), "removes repository protection") {
		t.Fatal(content, err)
	}
	// A Rulesets operation adds a managed resource independently of label/field changes.
	report.Managed = 0
	if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
		t.Fatal("invalid count")
	}
	for _, body := range []string{`{"operation":"delete","before":null}`, strings.Replace(string(data), `"target":"branch"`, `"target":"branch","unknown":true`, 1), strings.Replace(string(data), `"target":"branch"`, `"Target":"branch"`, 1)} {
		var r actions.PlanReport
		if strings.HasPrefix(body, `{"operation"`) {
			var c actions.RulesetPlanChange
			if err := json.Unmarshal([]byte(body), &c); err == nil {
				t.Fatal("null accepted")
			}
		} else if err := json.Unmarshal([]byte(body), &r); err == nil {
			t.Fatal("unknown policy accepted")
		}
	}
}
