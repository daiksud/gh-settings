package settings

import "fmt"

const (
	CopilotMCPPath          = "repository.settings.copilot.code_review.general.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests"
	CopilotPublicSearchPath = "repository.settings.copilot.code_review.general.search_public_repositories"
)

// CopilotContextObservation holds independently observable context sources for code review.
// Missing evidence for a source cannot establish an explicit off setting.
type CopilotContextObservation struct {
	MCP          *bool
	PublicSearch *bool
}

func (o CopilotContextObservation) Value(field string) (bool, error) {
	var value *bool
	switch field {
	case CopilotMCPPath:
		value = o.MCP
	case CopilotPublicSearchPath:
		value = o.PublicSearch
	default:
		return false, fmt.Errorf("unknown Copilot context field %q", field)
	}
	if value == nil {
		return false, fmt.Errorf("%s is unavailable", field)
	}
	return *value, nil
}

type CopilotToggleChange struct {
	field         string
	before, after bool
}

func NewCopilotToggleChange(field string, before, after bool) (CopilotToggleChange, error) {
	if field != CopilotMCPPath && field != CopilotPublicSearchPath {
		return CopilotToggleChange{}, fmt.Errorf("unsupported Copilot context field %q", field)
	}
	if before == after {
		return CopilotToggleChange{}, fmt.Errorf("unchanged %s", field)
	}
	return CopilotToggleChange{field: field, before: before, after: after}, nil
}
func (c CopilotToggleChange) Field() string        { return c.field }
func (c CopilotToggleChange) Values() (bool, bool) { return c.before, c.after }

func CompareCopilotToggle(o CopilotContextObservation, field string, desired *bool) (int, []CopilotToggleChange, error) {
	if desired == nil {
		return 0, nil, nil
	}
	before, err := o.Value(field)
	if err != nil {
		return 0, nil, err
	}
	if before == *desired {
		return 1, nil, nil
	}
	change, err := NewCopilotToggleChange(field, before, *desired)
	if err != nil {
		return 0, nil, err
	}
	return 1, []CopilotToggleChange{change}, nil
}
