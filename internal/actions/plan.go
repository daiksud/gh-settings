package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

const (
	invalidConfigDiagnostic = "configuration validation failed"
	targetFailureDiagnostic = "repository target resolution failed"
	readFailureDiagnostic   = "repository read failed"
)

// PlanReport is same-version, ephemeral presentation data, never apply input.
type PlanReport struct {
	Outcome          string                       `json:"outcome"`
	Target           string                       `json:"target,omitempty"`
	Managed          int                          `json:"managed"`
	Changes          []PlanChange                 `json:"changes"`
	CommitMessages   []CommitMessagePlanChange    `json:"commit_messages,omitempty"`
	Diagnostic       string                       `json:"diagnostic,omitempty"`
	Labels           []LabelPlanChange            `json:"labels,omitempty"`
	Actions          []PlanChange                 `json:"actions,omitempty"`
	ActionsExecution []ActionsExecutionPlanChange `json:"actions_execution,omitempty"`
	Security         []PlanChange                 `json:"security,omitempty"`
	Rulesets         []RulesetPlanChange          `json:"rulesets,omitempty"`
	Environments     []EnvironmentPlanChange      `json:"environments,omitempty"`
	Pages            []PagesPlanChange            `json:"pages,omitempty"`
}

// PlanChange retains the native string, boolean, integer or string-list comparison values.
type PlanChange struct {
	Field                       string                                `json:"field"`
	Before                      any                                   `json:"before"`
	After                       any                                   `json:"after"`
	Normalization               *settings.ObservationNormalization    `json:"normalization,omitempty"`
	SecurityStatusNormalization *settings.SecurityStatusNormalization `json:"security_status_normalization,omitempty"`
}

func writePlanReport(path string, plan func() (app.PlanResult, error)) error {
	result, planErr := plan()
	report := PlanReport{Outcome: "unmanaged", Managed: result.Comparison.Managed, Changes: make([]PlanChange, 0, len(result.Comparison.Changes))}
	if planErr != nil {
		var failure *app.PlanError
		if !errors.As(planErr, &failure) {
			return planErr
		}
		switch failure.Stage {
		case "invalid-config":
			report.Outcome = "invalid-config"
			report.Diagnostic = invalidConfigDiagnostic
		case "read-failure":
			report.Outcome = "read-failure"
			report.Target = failure.Target.Owner + "/" + failure.Target.Name
			report.Diagnostic = readFailureDiagnostic
		case "target-failure":
			report.Outcome = "target-failure"
			report.Diagnostic = targetFailureDiagnostic
		default:
			return planErr
		}
	}
	if result.Comparison.Managed > 0 {
		report.Target = result.Target.Owner + "/" + result.Target.Name
		report.Outcome = "changed"
		if !result.Comparison.HasChanges() {
			report.Outcome = "no-difference"
		}
	}
	for _, change := range result.Comparison.Changes {
		entry := PlanChange{Field: change.Field().Path()}
		if normalization := change.Normalization(); normalization != (settings.ObservationNormalization{}) {
			entry.Normalization = &normalization
		}
		change.Visit(
			func(before, after bool) { entry.Before, entry.After = before, after },
			func(before, after string) { entry.Before, entry.After = before, after },
			func(before, after []string) { entry.Before, entry.After = before, after },
		)
		report.Changes = append(report.Changes, entry)
	}
	for _, change := range result.Comparison.CommitMessages {
		_, after := change.Values()
		report.CommitMessages = append(report.CommitMessages, CommitMessagePlanChange{Menu: change.Menu(), After: after, Normalization: change.Normalization()})
	}
	for _, change := range result.Comparison.Labels.Values() {
		report.Labels = append(report.Labels, labelPlanChange(change))
	}
	for _, change := range result.Comparison.Rulesets {
		display := change.Display()
		entry := RulesetPlanChange{Operation: display.Operation(), Before: display.Before(), After: display.After()}
		if record := display.EnforcementNormalization(); record != (settings.ObservationNormalization{}) {
			entry.EnforcementNormalization = &record
		}
		report.Rulesets = append(report.Rulesets, entry)
	}
	for _, change := range result.Comparison.Environments {
		report.Environments = append(report.Environments, EnvironmentPlanChange{Operation: change.Operation(), Before: change.Before(), After: change.After()})
	}
	for _, change := range result.Comparison.Actions {
		before, after := change.Values()
		entry := PlanChange{Field: change.Field(), Before: before, After: after}
		if normalization := change.Normalization(); normalization != (settings.ObservationNormalization{}) {
			entry.Normalization = &normalization
		}
		report.Actions = append(report.Actions, entry)
	}
	for _, change := range result.Comparison.ActionsExecution {
		report.ActionsExecution = append(report.ActionsExecution, ActionsExecutionPlanChange{Before: change.Before(), After: change.After(), Normalization: change.Normalization()})
	}
	for _, change := range result.Comparison.Security {
		before, after := change.Values()
		entry := PlanChange{Field: change.Field(), Before: before, After: after}
		if normalization := change.Normalization(); normalization != (settings.SecurityStatusNormalization{}) {
			entry.SecurityStatusNormalization = &normalization
		}
		report.Security = append(report.Security, entry)
	}
	for _, c := range result.Comparison.Pages {
		b, a := c.Values()
		entry := PagesPlanChange{Field: c.Field(), Before: b, After: a, BeforeInactive: c.BeforeInactive()}
		if record := c.Normalization(); record != (settings.ObservationNormalization{}) {
			entry.Normalization = &record
		}
		report.Pages = append(report.Pages, entry)
	}
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode plan report: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("write plan report: %w", err)
	}
	return planErr
}
