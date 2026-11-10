package settings

import "fmt"

const CopilotMCPPath = "repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests"

// CopilotMCPObservation retains acquisition independently of false.
// It describes stored configuration, never Copilot access or review execution.
type CopilotMCPObservation struct{ acquired, enabled bool }

func NewCopilotMCPObservation(enabled bool) CopilotMCPObservation {
	return CopilotMCPObservation{acquired: true, enabled: enabled}
}
func (o CopilotMCPObservation) Value() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("Copilot MCP tools observation was not acquired")
	}
	return o.enabled, nil
}

// CopilotMCPChange owns one changed boolean for the fixed UI control.
type CopilotMCPChange struct{ before, after bool }

func NewCopilotMCPChange(before, after bool) (CopilotMCPChange, error) {
	if before == after {
		return CopilotMCPChange{}, fmt.Errorf("unchanged Copilot MCP tools")
	}
	return CopilotMCPChange{before: before, after: after}, nil
}
func (c CopilotMCPChange) Field() string        { return CopilotMCPPath }
func (c CopilotMCPChange) Values() (bool, bool) { return c.before, c.after }

func CompareCopilotMCP(o CopilotMCPObservation, desired *bool) (int, []CopilotMCPChange, error) {
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
	change, _ := NewCopilotMCPChange(before, *desired)
	return 1, []CopilotMCPChange{change}, nil
}
