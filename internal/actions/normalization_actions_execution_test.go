package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func executionReportSelected(t *testing.T) []settings.ActionsPolicyChange {
	t.Helper()
	var changes []settings.ActionsPolicyChange
	for _, test := range []struct {
		field string
		after any
	}{{"actions.selected_actions.github_owned_allowed", false}, {"actions.selected_actions.verified_allowed", false}, {"actions.selected_actions.patterns_allowed", []string{}}} {
		c, err := settings.NewActionsPolicyChange(test.field, nil, test.after)
		if err != nil {
			t.Fatal(err)
		}
		changes = append(changes, c)
	}
	return changes
}

func TestActionsExecutionPrivateReportRetainsClosedRadioAndNativeOwnership(t *testing.T) {
	raw := "<script>EXECUTION_CANARY</script>\n::warning::untrusted"
	for _, enabled := range []bool{false, true} {
		for _, desired := range []uimodel.ActionsPermissionChoice{uimodel.AllowAllActionsAndReusableWorkflows, uimodel.DisableActions, uimodel.AllowOwnerActionsAndReusableWorkflows, uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows} {
			if !enabled && desired == uimodel.DisableActions {
				continue
			}
			record := settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: raw, Fallback: "allow_all_actions_and_reusable_workflows"}
			radio, err := settings.NewActionsExecutionChange(settings.ActionsPolicyState{Enabled: enabled, AllowedActions: "all", AllowedActionsNormalization: record}, desired)
			if err != nil {
				t.Fatal(err)
			}
			managed := len(radio.NativeFields())
			var native []settings.ActionsPolicyChange
			if desired == uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows {
				native = executionReportSelected(t)
				managed += 3
			}
			path := filepath.Join(t.TempDir(), "plan.json")
			var stdout, stderr bytes.Buffer
			status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
				return app.PlanResult{Target: repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: managed, Actions: native, ActionsExecution: []settings.ActionsExecutionChange{radio}}}, nil
			})
			data, err := os.ReadFile(path)
			if err != nil || status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatal(status, stdout.String(), stderr.String(), err)
			}
			var report actions.PlanReport
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatal(err)
			}
			if report.Outcome != "changed" || report.Managed != managed || len(report.ActionsExecution) != 1 || report.ActionsExecution[0].Before != radio.Before() || report.ActionsExecution[0].After != desired || report.ActionsExecution[0].Normalization != record || len(report.Actions) != len(native) {
				t.Fatal("lost scalar radio evidence/count", string(data))
			}
			summary, err := actions.RenderSummary(report, summaryMetadata())
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(html.UnescapeString(summary), strconv.Quote(string(desired))+" (write required)") || strings.Contains(summary, "<script>") {
				t.Fatal(summary)
			}
			warning := "Warning: repository.settings.actions.general.actions_permissions: observed " + strconv.Quote(raw) + " is outside UI choices; using \"allow_all_actions_and_reusable_workflows\" (projection only)."
			if enabled {
				if strings.Count(html.UnescapeString(summary), warning) != 1 || !strings.Contains(summary, "normalized observation") {
					t.Fatal("visible projection missing", summary)
				}
			} else if strings.Contains(summary, "EXECUTION_CANARY") || strings.Contains(summary, "Warning:") || strings.Contains(summary, "normalized observation") || !strings.Contains(html.UnescapeString(summary), `"disable_actions" →`) {
				t.Fatal("latent policy disclosed", summary)
			}
			invalid := report
			invalid.Managed = managed - 1
			if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
				t.Fatal("native ownership undercount accepted", invalid.Managed)
			}
			private := filepath.Join(t.TempDir(), "status.md")
			t.Setenv("SETTINGS_TARGET", "owner/current")
			stdout.Reset()
			stderr.Reset()
			status = actions.Run([]string{"summary", path, private, "status-only"}, &stdout, &stderr, "dev", nil)
			data, err = os.ReadFile(private)
			if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(data), "EXECUTION_CANARY") || strings.Contains(string(data), settings.ActionsPermissionsPath) {
				t.Fatal(status, stdout.String(), stderr.String(), string(data), err)
			}
		}
	}
}

func TestActionsExecutionReportRejectsConflictsForgedProofAndIncompleteSelection(t *testing.T) {
	record := settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: "", Fallback: "allow_all_actions_and_reusable_workflows"}
	base := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 2, Changes: []actions.PlanChange{}, ActionsExecution: []actions.ActionsExecutionPlanChange{{Before: uimodel.AllowAllActionsAndReusableWorkflows, After: uimodel.AllowAllActionsAndReusableWorkflows, Normalization: record}}}
	if _, err := actions.RenderSummary(base, summaryMetadata()); err != nil {
		t.Fatal("empty acquired policy rejected", err)
	}
	for _, mutate := range []func(*actions.PlanReport){
		func(r *actions.PlanReport) {
			r.ActionsExecution[0].Before = uimodel.AllowOwnerActionsAndReusableWorkflows
		},
		func(r *actions.PlanReport) { r.ActionsExecution[0].After = "all" },
		func(r *actions.PlanReport) { r.ActionsExecution[0].Normalization.RawValue = "all" },
		func(r *actions.PlanReport) { r.ActionsExecution[0].Normalization.Path = "wrong" },
		func(r *actions.PlanReport) { r.ActionsExecution[0].Normalization.Fallback = "disable_actions" },
		func(r *actions.PlanReport) {
			r.Managed = 4
			r.ActionsExecution = append(r.ActionsExecution, r.ActionsExecution[0])
		},
		func(r *actions.PlanReport) {
			r.Managed = 3
			r.Actions = []actions.PlanChange{{Field: "actions.enabled", Before: true, After: false}}
		},
		func(r *actions.PlanReport) {
			r.Managed = 3
			r.Actions = []actions.PlanChange{{Field: "actions.allowed_actions", Before: "all", After: "local_only"}}
		},
		func(r *actions.PlanReport) {
			r.Managed = 5
			r.ActionsExecution[0].After = uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows
		},
		func(r *actions.PlanReport) {
			r.Managed = 3
			r.Actions = []actions.PlanChange{{Field: "actions.selected_actions.verified_allowed", Before: true, After: false}}
		},
		func(r *actions.PlanReport) {
			r.ActionsExecution[0].Before = uimodel.DisableActions
			r.ActionsExecution[0].After = uimodel.DisableActions
		},
	} {
		r := base
		r.ActionsExecution = append([]actions.ActionsExecutionPlanChange{}, base.ActionsExecution...)
		mutate(&r)
		if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
			t.Fatal("forged radio accepted", r)
		}
	}
	for _, raw := range []string{
		`{"after":"disable_actions","normalization":{"path":"x","raw_value":"","fallback":"x"}}`,
		`{"before":"disable_actions","normalization":{"path":"x","raw_value":"","fallback":"x"}}`,
		`{"before":"disable_actions","after":null,"normalization":{"path":"x","raw_value":"","fallback":"x"}}`,
		`{"before":"disable_actions","after":"disable_actions"}`,
		`{"before":null,"after":"disable_actions","normalization":{"path":"x","raw_value":"","fallback":"x"}}`,
		`{"before":"disable_actions","after":false,"normalization":{"path":"x","raw_value":"","fallback":"x"}}`,
		`{"before":"disable_actions","after":"disable_actions","normalization":null}`,
		`{"before":"disable_actions","after":"disable_actions","normalization":{"path":"x","fallback":"x"}}`,
		`{"before":"disable_actions","after":"disable_actions","normalization":{"path":"x","raw_value":"","fallback":"x"},"extra":true}`,
	} {
		var entry actions.ActionsExecutionPlanChange
		if err := json.Unmarshal([]byte(raw), &entry); err == nil {
			t.Fatal("malformed entry accepted", raw)
		}
	}
}
