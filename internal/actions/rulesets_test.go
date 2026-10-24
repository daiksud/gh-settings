package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"reflect"
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
	if report.Outcome != "changed" || len(report.Rulesets) != 1 || !reflect.DeepEqual(report.Rulesets[0].Before, &settings.RulesetDeclaration{Name: "<policy>"}) {
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

func TestRulesetPlanReportAndSummaryProjectManagedValues(t *testing.T) {
	current, err := workspace.DecodeConfig([]byte(`schemaVersion: 1
settings:
  rulesets:
    - name: default
      target: branch
      enforcement: active
      conditions: {ref_name: {include: ['refs/heads/CANARY-unmanaged-condition'], exclude: []}}
      bypass_actors: [{actor_type: Team, actor_id: 987654321, bypass_mode: always}]
      rules: [{type: deletion}, {type: non_fast_forward}]
`))
	if err != nil {
		t.Fatal(err)
	}
	desired, err := workspace.DecodeConfig([]byte(`schemaVersion: 1
settings: {rulesets: [{name: default, enforcement: disabled}]}
`))
	if err != nil {
		t.Fatal(err)
	}
	observed, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: current.Rulesets.Values()[0]}}, 0)
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
	for _, hidden := range []string{"CANARY-unmanaged-condition", "987654321", `"rules"`, `"bypass_actors"`, `"conditions"`} {
		if strings.Contains(string(data), hidden) {
			t.Fatal("report leaked unmanaged value", hidden, string(data))
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
	for _, hidden := range []string{"CANARY-unmanaged-condition", "987654321", `&quot;rules&quot;`, `&quot;bypass_actors&quot;`, `&quot;conditions&quot;`} {
		if strings.Contains(content, hidden) {
			t.Fatal("summary leaked unmanaged value", hidden, content)
		}
	}
	visible := html.UnescapeString(content)
	if !strings.Contains(visible, `{"name":"default","enforcement":"active"} → {"name":"default","enforcement":"disabled"}`) {
		t.Fatal("managed enforcement missing", visible)
	}
}
