package settings

import "fmt"

const CopilotInstructionsPath = "repository.settings.copilot.code_review.use_custom_instructions"

// CopilotInstructionsObservation retains acquisition independently of false.
// It describes stored configuration, never Copilot access or review execution.
type CopilotInstructionsObservation struct{ acquired, enabled bool }

func NewCopilotInstructionsObservation(enabled bool) CopilotInstructionsObservation {
	return CopilotInstructionsObservation{acquired: true, enabled: enabled}
}
func (o CopilotInstructionsObservation) Value() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("Copilot custom instructions observation was not acquired")
	}
	return o.enabled, nil
}

// CopilotInstructionsChange owns one changed boolean for the fixed UI control.
type CopilotInstructionsChange struct{ before, after bool }

func NewCopilotInstructionsChange(before, after bool) (CopilotInstructionsChange, error) {
	if before == after {
		return CopilotInstructionsChange{}, fmt.Errorf("unchanged Copilot custom instructions")
	}
	return CopilotInstructionsChange{before: before, after: after}, nil
}
func (c CopilotInstructionsChange) Field() string        { return CopilotInstructionsPath }
func (c CopilotInstructionsChange) Values() (bool, bool) { return c.before, c.after }

func CompareCopilotInstructions(o CopilotInstructionsObservation, desired *bool) (int, []CopilotInstructionsChange, error) {
	if desired == nil {
		return 0, nil, nil
	}
	before, err := o.Value()
	if err != nil {
		return 0, nil, err
	}
	if before == *desired {
		return 1, nil, nil
	}
	change, _ := NewCopilotInstructionsChange(before, *desired)
	return 1, []CopilotInstructionsChange{change}, nil
}
