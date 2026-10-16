package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/daiksud/gh-settings/internal/app"
)

// PlanReport is same-version, ephemeral presentation data, never apply input.
type PlanReport struct {
	Outcome    string       `json:"outcome"`
	Target     string       `json:"target,omitempty"`
	Managed    int          `json:"managed"`
	Changes    []PlanChange `json:"changes"`
	Diagnostic string       `json:"diagnostic,omitempty"`
}

// PlanChange retains the native string/string or bool/bool comparison values.
type PlanChange struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
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
			report.Diagnostic = "configuration validation failed"
		case "read-failure":
			report.Outcome = "read-failure"
			report.Target = failure.Target.Owner + "/" + failure.Target.Name
			report.Diagnostic = "repository read failed"
		case "target-failure":
			report.Outcome = "target-failure"
			report.Diagnostic = "repository target resolution failed"
		default:
			return planErr
		}
	}
	if result.Comparison.Managed > 0 {
		report.Target = result.Target.Owner + "/" + result.Target.Name
		report.Outcome = "changed"
		if len(result.Comparison.Changes) == 0 {
			report.Outcome = "no-difference"
		}
	}
	for _, change := range result.Comparison.Changes {
		report.Changes = append(report.Changes, PlanChange{Field: change.Field, Before: change.Before, After: change.After})
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
