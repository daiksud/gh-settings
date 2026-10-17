package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

// LabelAttributes holds only the writable attributes represented by this report.
type LabelAttributes struct {
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (attributes *LabelAttributes) UnmarshalJSON(data []byte) error {
	type plain LabelAttributes
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for key, value := range raw {
		if key != "color" && key != "description" {
			return fmt.Errorf("unknown label attribute %q", key)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("null label attribute %q", key)
		}
	}
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*attributes = LabelAttributes(value)
	return nil
}

// LabelPlanChange is same-version presentation data, never apply input.
type LabelPlanChange struct {
	Operation string           `json:"operation"`
	Name      string           `json:"name"`
	Before    *LabelAttributes `json:"before,omitempty"`
	After     *LabelAttributes `json:"after,omitempty"`
}

func (change *LabelPlanChange) UnmarshalJSON(data []byte) error {
	type plain LabelPlanChange
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*change = LabelPlanChange(value)
	return nil
}

func labelPlanChange(change settings.LabelChange) LabelPlanChange {
	result := LabelPlanChange{Operation: change.Operation, Name: change.Name}
	if change.Before != nil {
		color, description := change.Before.Color, change.Before.Description
		result.Before = &LabelAttributes{Color: &color, Description: &description}
	}
	if change.After != nil {
		result.After = &LabelAttributes{Color: change.After.Color, Description: change.After.Description}
	}
	return result
}

func validateLabelReport(changes []LabelPlanChange) error {
	native := make([]settings.LabelChange, 0, len(changes))
	for _, change := range changes {
		value := settings.LabelChange{Operation: change.Operation, Name: change.Name}
		if change.Before != nil {
			if change.Before.Color == nil || change.Before.Description == nil {
				return fmt.Errorf("incomplete label before value")
			}
			value.Before = &settings.LabelState{Name: change.Name, Color: *change.Before.Color, Description: *change.Before.Description}
		}
		if change.After != nil {
			value.After = &settings.LabelDeclaration{Name: change.Name, Color: change.After.Color, Description: change.After.Description}
		}
		native = append(native, value)
	}
	return settings.ValidateLabelChanges(native)
}

func labelSummaryLine(change LabelPlanChange) string {
	prefix := fmt.Sprintf("settings.labels: %s %q", change.Operation, change.Name)
	switch change.Operation {
	case "delete":
		return prefix + " (removes issue/PR associations)"
	case "create":
		var attributes []string
		if change.After.Color != nil {
			attributes = append(attributes, fmt.Sprintf("color=%q", *change.After.Color))
		}
		if change.After.Description != nil {
			attributes = append(attributes, fmt.Sprintf("description=%q", *change.After.Description))
		}
		if len(attributes) > 0 {
			prefix += " " + strings.Join(attributes, " ")
		}
	case "update":
		var attributes []string
		if change.After.Color != nil {
			attributes = append(attributes, fmt.Sprintf("color: %q → %q", *change.Before.Color, *change.After.Color))
		}
		if change.After.Description != nil {
			attributes = append(attributes, fmt.Sprintf("description: %q → %q", *change.Before.Description, *change.After.Description))
		}
		prefix += " " + strings.Join(attributes, ", ")
	}
	return prefix
}
