package cli

import (
	"errors"

	"testing"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestAgentVariablePlanAndFailureIdentity(t *testing.T) {
	agent, _ := settings.NewVariableChange("shared", nil, "agent-value")
	workflowBefore := "workflow-before"
	workflow, _ := settings.NewVariableChange("SHARED", &workflowBefore, "workflow-after")
	for _, mixed := range []bool{false, true} {
		comparison := settings.Comparison{Managed: 1, AgentsVariables: []settings.VariableChange{agent}}
		want := "repository.settings.agents.variables: create \"SHARED\" value=\"agent-value\"\n"
		if mixed {
			comparison.Managed = 2
			comparison.Variables = []settings.VariableChange{workflow}
			want = "repository.settings.actions.variables: update \"SHARED\" value: \"workflow-before\" → \"workflow-after\"\n" + want
		}
		if got := string(RenderComparison(comparison)); got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	}
	for _, status := range []app.ApplyOperationStatus{app.OperationFailedRead, app.OperationFailedUnknown} {
		err := applyDiagnostic(errors.New("uncertain"), app.ApplyResult{Operations: []app.ApplyOperation{{Kind: app.AgentVariableUpsert, VariableName: "SHARED", Status: status}}})
		if err.Error() != "repository.settings.agents.variables: uncertain" {
			t.Fatal(err)
		}
	}
}
