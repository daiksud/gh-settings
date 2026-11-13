package cli

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

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
