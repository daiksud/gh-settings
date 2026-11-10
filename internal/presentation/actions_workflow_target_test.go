package presentation

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsWorkflowTargetUsesLiteralUIKeys(t *testing.T) {
	policy := settings.ActionsEventPolicy{Name: "events", Enforcement: "disabled", Events: []string{"push"}, TargetWorkflows: &settings.ActionsWorkflowTarget{Include: []string{"z.yml", "a.yml"}, Exclude: []string{"skip.yml"}}}
	change, err := settings.NewActionsEventPolicyChange(nil, policy)
	if err != nil {
		t.Fatal(err)
	}
	line := ActionsEventPolicyLine(change)
	want := `target_workflows={"include":["a.yml","z.yml"],"exclude":["skip.yml"]}`
	if !strings.Contains(line, want) {
		t.Fatalf("missing literal UI selection %s in %s", want, line)
	}
	for _, obsolete := range []string{`"Include"`, `"Exclude"`, `workflow_path`} {
		if strings.Contains(line, obsolete) {
			t.Fatal("non-UI presentation", line)
		}
	}
}
