package actions

import (
	"encoding/json"

	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
	"testing"
)

func TestRunnerReportClosedBoundary(t *testing.T) {
	raw := `[{"field":"repository.settings.actions.runners","before":{"id":23,"custom_labels":["old"]},"after":{"id":23,"custom_labels":["GPU"]}}]`
	var changes []RunnerPlanChange
	json.Unmarshal([]byte(raw), &changes)
	v, e := validateRunnersReport(changes)
	if e != nil || len(v) != 1 {
		t.Fatal(v, e)
	}
	for _, bad := range []string{strings.Replace(raw, `"id":23`, `"ID":23`, 1), strings.Replace(raw, `"custom_labels":["old"]`, `"custom_labels":["old"],"name":"private"`, 1), strings.Replace(raw, `["GPU"]`, `null`, 1), strings.Replace(raw, `["GPU"]`, `[null]`, 1), strings.Replace(raw, `["GPU"]`, `["gpu","GPU"]`, 1), strings.Replace(raw, `["GPU"]`, `["OLD"]`, 1), strings.Replace(raw, `"id":23`, `"id":24`, 1), strings.Replace(raw, `"id":23`, `"id":"23"`, 1)} {
		if json.Unmarshal([]byte(bad), &changes) != nil {
			continue
		}
		if _, e := validateRunnersReport(changes); e == nil {
			t.Fatal("accepted", bad)
		}
	}
	json.Unmarshal([]byte(raw), &changes)
	if _, e := validateRunnersReport(append(changes, changes...)); e == nil {
		t.Fatal("duplicate runner")
	}
	report := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []PlanChange{}, Runners: changes}
	if _, e := validatePlanReport(report, "owner/current"); e != nil {
		t.Fatal(e)
	}
	report.Managed = 0
	if _, e := validatePlanReport(report, "owner/current"); e == nil {
		t.Fatal("uncounted runner family")
	}
	report.Managed = 1
	report.Outcome = "no-difference"
	if _, e := validatePlanReport(report, "owner/current"); e == nil {
		t.Fatal("false no-difference")
	}
	if !validFailureResource("read-failure", settings.RunnersPath) {
		t.Fatal("missing closed family path")
	}
}

func TestRunnerReportExactNumericIDsAndOuterShape(t *testing.T) {
	raw := `{"runners":[{"field":"repository.settings.actions.runners","before":{"id":9007199254740993,"custom_labels":["old"]},"after":{"id":9007199254740993,"custom_labels":["new"]}}]}`
	var report PlanReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatal(err)
	}
	changes, err := validateRunnersReport(report.Runners)
	if err != nil {
		t.Fatal(err)
	}
	before, after := changes[0].Values()
	if before.ID != 9007199254740993 || after.ID != before.ID {
		t.Fatal("numeric ID rounded", before.ID, after.ID)
	}
	for _, bad := range []string{`{"runners":null}`, strings.Replace(raw, `"field":`, `"Field":`, 1), strings.Replace(raw, `"after":`, `"normalization":null,"after":`, 1), strings.Replace(raw, `9007199254740993`, `1.5`, 1), strings.Replace(raw, `9007199254740993`, `null`, 1)} {
		var decoded PlanReport
		if json.Unmarshal([]byte(bad), &decoded) == nil {
			t.Fatal("accepted malformed runner report", bad)
		}
	}
}
