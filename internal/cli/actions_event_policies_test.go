package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsEventPolicyExportFlag(t *testing.T) {
	for _, args := range [][]string{{"export", "--actions-policies"}, {"export", "--actions-policies", "--actions"}, {"export", "--actions-policies", "--actions-policies"}} {
		var out, diag bytes.Buffer
		calls := 0
		status := Run(args, &out, &diag, "dev", Operations{Export: func(options ExportOptions) error {
			calls++
			if !options.ActionsPolicies || options.Actions != (len(args) == 3 && args[2] == "--actions") {
				t.Fatal(options)
			}
			return nil
		}})
		duplicate := len(args) == 3 && args[2] == "--actions-policies"
		if duplicate {
			if status == 0 || calls != 0 {
				t.Fatal(status, calls)
			}
		} else if status != 0 || calls != 1 {
			t.Fatal(status, calls, diag.String())
		}
	}
}
func TestActionsEventPolicyPresentationAndRecovery(t *testing.T) {
	c, err := settings.NewActionsEventPolicyChange(nil, settings.ActionsEventPolicy{Name: "policy", Enforcement: "active", Events: []string{"push"}})
	if err != nil {
		t.Fatal(err)
	}
	output := string(RenderComparison(settings.Comparison{Managed: 1, ActionsPolicies: []settings.ActionsEventPolicyChange{c}}))
	if !strings.Contains(output, `repository.settings.actions.policies: create "policy"`) || !strings.Contains(output, `restrict_events.allowed_events=["push"]`) {
		t.Fatal(output)
	}
	diagnostic := createdIDDiagnostic(applyDiagnostic(errTestPolicy{}, app.ApplyResult{Operations: []app.ApplyOperation{{Kind: app.ActionsEventPolicyCreate, Status: app.OperationFailedUnknown}}}), []app.CreatedObject{{Collection: "actions_policies", Name: "policy", ID: 7}})
	if !strings.Contains(diagnostic.Error(), "repository.settings.actions.policies") || !strings.Contains(diagnostic.Error(), "id 7") {
		t.Fatal(diagnostic)
	}
}

type errTestPolicy struct{}

func (errTestPolicy) Error() string { return "failed" }
