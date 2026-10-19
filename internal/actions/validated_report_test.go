package actions

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestValidatedReportRetainsNativeValues(t *testing.T) {
	var report PlanReport
	input := `{"outcome":"changed","target":"owner/current","managed":4,"changes":[{"field":"repository.description","before":"before","after":"after"},{"field":"repository.has_issues","before":true,"after":false},{"field":"topics.names","before":["old"],"after":[]}],"labels":[{"operation":"create","name":"bug","after":{"color":"abcdef","description":""}}]}`
	if err := json.Unmarshal([]byte(input), &report); err != nil {
		t.Fatal(err)
	}
	metadata := RunMetadata{
		Actor: "actor", TriggeringActor: "rerun", Event: "pull_request", RunID: "123", RunAttempt: "1",
		WorkflowRef: "owner/workflows/.github/workflows/settings.yml@refs/heads/main", WorkflowSHA: strings.Repeat("a", 40),
		ConfigSHA: strings.Repeat("b", 40), Target: "owner/current", ActionRepository: "owner/action", ActionRef: "v1",
		CLIVersion: "dev", CLISHA256: strings.Repeat("c", 64),
	}
	want, err := RenderSummary(report, metadata)
	if err != nil {
		t.Fatal(err)
	}
	validated, err := validatePlanReport(report, metadata.Target)
	if err != nil {
		t.Fatal(err)
	}
	report.Changes[2].Before.([]any)[0] = "invalid\nname"
	report.Changes[2].After = []any{"new"}
	report.Changes[0] = PlanChange{}
	*report.Labels[0].After.Color = "bad"
	*report.Labels[0].After.Description = "changed"
	report.Labels[0].Name = ""
	report.Outcome, report.Diagnostic, report.Managed = "invalid", "changed", 0
	got := renderSummary(validated, metadata, []string{"owner", "workflows", ".github/workflows/settings.yml"})
	if got != want {
		t.Fatalf("raw input mutation changed typed rendering:\n%s", got)
	}
}

func TestInvalidReportReturnsNoPartialTypedValues(t *testing.T) {
	report := PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []PlanChange{
		{Field: "repository.description", Before: "before", After: "after"},
		{Field: "repository.has_issues", Before: true, After: "false"},
	}}
	validated, err := validatePlanReport(report, "owner/current")
	if err == nil || !reflect.DeepEqual(validated, validatedPlanReport{}) {
		t.Fatalf("partial report=%+v error=%v", validated, err)
	}
}
