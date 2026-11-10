package cli

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestAgentVariableExportOptions(t *testing.T) {
	for _, tc := range []struct {
		args, want, actions []string
		fail                bool
	}{
		{args: []string{"export"}},
		{args: []string{"export", "--agent-variable", "shared", "--actions", "--agent-variable", "SECOND"}, want: []string{"SHARED", "SECOND"}},
		{args: []string{"export", "--agent-variable", "shared", "--variable", "SHARED"}, want: []string{"SHARED"}, actions: []string{"SHARED"}},
		{args: []string{"export", "--agent-variable"}, fail: true},
		{args: []string{"export", "--agent-variable", "A", "--agent-variable", "a"}, fail: true},
		{args: []string{"export", "--agent-variable", "GITHUB_X"}, fail: true},
		{args: []string{"export", "--agent-variable", "2A"}, fail: true},
		{args: []string{"export", "--agent-variables-all"}, fail: true},
	} {
		var out, diag bytes.Buffer
		calls := 0
		code := Run(tc.args, &out, &diag, "dev", Operations{Export: func(o ExportOptions) error {
			calls++
			if !reflect.DeepEqual(o.AgentsVariables, tc.want) || !reflect.DeepEqual(o.Variables, tc.actions) {
				t.Fatal(o)
			}
			return nil
		}})
		if (code != 0) != tc.fail || calls != map[bool]int{false: 1, true: 0}[tc.fail] {
			t.Fatal(tc.args, code, calls, diag.String())
		}
	}
	var out, diag bytes.Buffer
	if Run([]string{"--help"}, &out, &diag, "dev", Operations{}) != 0 || !strings.Contains(out.String(), "--agent-variable NAME") || !strings.Contains(out.String(), "non-sensitive repository Agents variables") || !strings.Contains(out.String(), "Agent variables read") {
		t.Fatal(out.String(), diag.String())
	}
}

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
