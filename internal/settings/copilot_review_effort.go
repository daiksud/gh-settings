package settings

import (
	"fmt"
	"slices"
	"strings"
)

const CopilotReviewEffortPath = "repository.settings.copilot.code_review.review_effort_level"

func ValidateCopilotReviewEffort(value string) error {
	switch value {
	case "default", "lite", "balanced", "max":
		return nil
	}
	return fmt.Errorf("%s must be default, lite, balanced or max", CopilotReviewEffortPath)
}

// CopilotReviewEffortObservation retains the API's visible configured value,
// effective effort and current availability separately. A nil configured value
// can hide an unavailable saved override; it does not prove inheritance was saved.
type CopilotReviewEffortObservation struct {
	acquired   bool
	configured *string
	effective  string
	available  []string
}

func NewCopilotReviewEffortObservation(configured *string, effective string, available []string) (CopilotReviewEffortObservation, error) {
	if effective != "lite" && effective != "balanced" && effective != "max" {
		return CopilotReviewEffortObservation{}, fmt.Errorf("unsupported observed effective Copilot review effort %q", effective)
	}
	if configured != nil && *configured != "lite" && *configured != "balanced" && *configured != "max" {
		return CopilotReviewEffortObservation{}, fmt.Errorf("unsupported observed configured Copilot review effort %q", *configured)
	}
	seen := map[string]bool{}
	for _, value := range available {
		if strings.TrimSpace(value) == "" || seen[value] {
			return CopilotReviewEffortObservation{}, fmt.Errorf("invalid available Copilot review efforts")
		}
		seen[value] = true
	}
	if !seen[effective] || configured != nil && !seen[*configured] {
		return CopilotReviewEffortObservation{}, fmt.Errorf("observed Copilot review effort is not available")
	}
	result := CopilotReviewEffortObservation{acquired: true, effective: effective, available: slices.Clone(available)}
	if configured != nil {
		value := *configured
		result.configured = &value
	}
	return result, nil
}
func (o CopilotReviewEffortObservation) Value() (string, error) {
	if !o.acquired {
		return "", fmt.Errorf("Copilot review effort observation was not acquired")
	}
	if o.configured == nil {
		return "default", nil
	}
	return *o.configured, nil
}
func (o CopilotReviewEffortObservation) Configured() (*string, error) {
	if _, err := o.Value(); err != nil {
		return nil, err
	}
	if o.configured == nil {
		return nil, nil
	}
	value := *o.configured
	return &value, nil
}
func (o CopilotReviewEffortObservation) Effective() (string, error) {
	if _, err := o.Value(); err != nil {
		return "", err
	}
	return o.effective, nil
}
func (o CopilotReviewEffortObservation) Available() ([]string, error) {
	if _, err := o.Value(); err != nil {
		return nil, err
	}
	return slices.Clone(o.available), nil
}
func (o CopilotReviewEffortObservation) ValidateDesired(desired string) error {
	if err := ValidateCopilotReviewEffort(desired); err != nil {
		return err
	}
	if _, err := o.Value(); err != nil {
		return err
	}
	if desired != "default" && !slices.Contains(o.available, desired) {
		return fmt.Errorf("requested Copilot review effort %q is not available", desired)
	}
	return nil
}

type CopilotReviewEffortChange struct{ before, after string }

func NewCopilotReviewEffortChange(before, after string) (CopilotReviewEffortChange, error) {
	if err := ValidateCopilotReviewEffort(before); err != nil {
		return CopilotReviewEffortChange{}, err
	}
	if err := ValidateCopilotReviewEffort(after); err != nil {
		return CopilotReviewEffortChange{}, err
	}
	if before == after {
		return CopilotReviewEffortChange{}, fmt.Errorf("unchanged Copilot review effort")
	}
	return CopilotReviewEffortChange{before: before, after: after}, nil
}
func (c CopilotReviewEffortChange) Field() string            { return CopilotReviewEffortPath }
func (c CopilotReviewEffortChange) Values() (string, string) { return c.before, c.after }
func CompareCopilotReviewEffort(o CopilotReviewEffortObservation, desired *string) (int, []CopilotReviewEffortChange, error) {
	if desired == nil {
		return 0, nil, nil
	}
	if err := o.ValidateDesired(*desired); err != nil {
		return 0, nil, err
	}
	before, _ := o.Value()
	if before == *desired {
		return 1, nil, nil
	}
	change, _ := NewCopilotReviewEffortChange(before, *desired)
	return 1, []CopilotReviewEffortChange{change}, nil
}
