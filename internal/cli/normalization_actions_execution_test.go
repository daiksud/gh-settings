package cli_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestActionsExecutionRadioPresentationDistinguishesVisibleProjection(t *testing.T) {
	raw := "FUTURE_POLICY\n::warning::<script>\"&"
	for _, enabled := range []bool{false, true} {
		for _, desired := range []uimodel.ActionsPermissionChoice{uimodel.AllowAllActionsAndReusableWorkflows, uimodel.DisableActions, uimodel.AllowOwnerActionsAndReusableWorkflows, uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows} {
			if !enabled && desired == uimodel.DisableActions {
				continue
			}
			radio, err := settings.NewActionsExecutionChange(settings.ActionsPolicyState{Enabled: enabled, AllowedActions: "all", AllowedActionsNormalization: settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: raw, Fallback: "allow_all_actions_and_reusable_workflows"}}, desired)
			if err != nil {
				t.Fatal(err)
			}
			got := string(cli.RenderComparison(settings.Comparison{Managed: len(radio.NativeFields()), ActionsExecution: []settings.ActionsExecutionChange{radio}}))
			want := "repository.settings.actions.general.actions_permissions: \"disable_actions\" → " + strconv.Quote(string(desired)) + " (write required)\n"
			if enabled {
				want = "Warning: repository.settings.actions.general.actions_permissions: observed " + strconv.Quote(raw) + " is outside UI choices; using \"allow_all_actions_and_reusable_workflows\" (projection only).\nrepository.settings.actions.general.actions_permissions: \"allow_all_actions_and_reusable_workflows\" (normalized observation) → " + strconv.Quote(string(desired)) + " (write required)\n"
			}
			if got != want || strings.Contains(got, "\n::warning::") || strings.Contains(got, "settings.actions.enabled") || strings.Contains(got, "settings.actions.allowed_actions") {
				t.Fatalf("got %q want %q", got, want)
			}
			if !enabled && (strings.Contains(got, "FUTURE_POLICY") || strings.Contains(got, "Warning:") || strings.Contains(got, "normalized observation")) {
				t.Fatal("hidden policy disclosed as projection", got)
			}
		}
	}
}

func TestOrdinaryActionsRenderingUsesRadioChoice(t *testing.T) {
	change, err := settings.NewActionsPolicyChange("actions.enabled", false, true)
	if err != nil {
		t.Fatal(err)
	}
	got := string(cli.RenderComparison(settings.Comparison{Managed: 1, Actions: []settings.ActionsPolicyChange{change}, ActionsExecutionContext: &settings.ActionsExecutionContext{Before: uimodel.DisableActions, After: uimodel.AllowAllActionsAndReusableWorkflows, ObservedAllowed: "all"}}))
	if got != "repository.settings.actions.general.actions_permissions: \"disable_actions\" → \"allow_all_actions_and_reusable_workflows\"\n" {
		t.Fatal(got)
	}
}
