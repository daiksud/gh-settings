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
	"go.yaml.in/yaml/v4"
)

// rulesetReportFixture constructs internal API policy for report tests, not public YAML.
func rulesetReportFixture(body []byte) (settings.Declaration, error) {
	var policies []settings.RulesetDeclaration
	if err := yaml.Unmarshal(body, &policies); err != nil {
		return settings.Declaration{}, err
	}
	declarations, err := settings.NewRulesetDeclarations(policies)
	if err != nil {
		return settings.Declaration{}, err
	}
	return settings.Declaration{Rulesets: &declarations}, nil
}

func TestRulesetPlanReportAndSummary(t *testing.T) {
	d, err := rulesetReportFixture([]byte(`[{name: '<policy>', target: branch, enforcement: active, conditions: {ref_name: {include: ['~ALL'], exclude: []}}, bypass_actors: [], rules: [{type: deletion}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	before := d.Rulesets.Values()[0]
	after := before
	disabled := "disabled"
	after.Enforcement = &disabled
	change, err := settings.NewRulesetChange("update", 7, &before, &after)
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
	if report.Outcome != "changed" || len(report.Rulesets) != 1 || report.Rulesets[0].Before.ID == nil || *report.Rulesets[0].Before.ID != 7 || report.Rulesets[0].After.ID == nil {
		t.Fatal(string(data))
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil || strings.Contains(content, "<policy>") || !strings.Contains(html.UnescapeString(content), `"id":7`) {
		t.Fatal(content, err)
	}
	// A Rulesets operation adds a managed resource independently of label/field changes.
	report.Managed = 0
	if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
		t.Fatal("invalid count")
	}
	create := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Rulesets: []actions.RulesetPlanChange{{Operation: "create", After: &before}}}
	createData, err := json.Marshal(create)
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{"operation":"delete","before":null}`, strings.Replace(string(createData), `"target":"branch"`, `"target":"branch","unknown":true`, 1), strings.Replace(string(createData), `"target":"branch"`, `"Target":"branch"`, 1)} {
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

func TestRulesetPlanReportAndSummaryShowSelectedWholeObjects(t *testing.T) {
	current, err := rulesetReportFixture([]byte(`
- name: default
  target: branch
  enforcement: active
  conditions: {ref_name: {include: ['refs/heads/CANARY-selected-condition'], exclude: []}}
  bypass_actors: [{actor_type: Team, actor_id: 987654321, bypass_mode: always}]
  rules: [{type: deletion}, {type: non_fast_forward}]
`))
	if err != nil {
		t.Fatal(err)
	}
	desired, err := rulesetReportFixture([]byte(`[{id: 7, name: default, enforcement: disabled, conditions: {ref_name: {include: ['~ALL']}}}]`))
	if err != nil {
		t.Fatal(err)
	}
	unlisted := current.Rulesets.Values()[0]
	unlisted.Name = "CANARY-unlisted-object"
	observed, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: current.Rulesets.Values()[0]}, {ID: 8, Policy: unlisted}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := settings.ReconcileRulesets(observed, desired.Rulesets)
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Rulesets: changes}}, nil
	})
	data, err := os.ReadFile(path)
	if status != 0 || err != nil {
		t.Fatal(status, err, stderr.String())
	}
	if strings.Contains(string(data), "CANARY-unlisted-object") {
		t.Fatal("unselected object reported", string(data))
	}
	for _, shown := range []string{"CANARY-selected-condition", "987654321", `"rules"`, `"bypass_actors"`, `"conditions"`, `"id":7`} {
		if !strings.Contains(string(data), shown) {
			t.Fatal("selected whole-object value hidden", shown, string(data))
		}
	}
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil {
		t.Fatal(err)
	}
	visible := html.UnescapeString(content)
	if strings.Contains(visible, "CANARY-unlisted-object") || !strings.Contains(visible, `"enforcement_status":"disabled"`) || !strings.Contains(visible, `"restrict_deletions":false`) || !strings.Contains(visible, `"id":7`) {
		t.Fatal("selected defaults or identity missing", visible)
	}
}
